package services

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"bethapi/api/dto"
	"bethapi/api/models"
	"bethapi/api/repository"
	"bethapi/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	googleoauth2 "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

type AuthService struct {
	userRepo    *repository.UserRepository
	redisClient *redis.Client
	oauthConfig *oauth2.Config
}

func NewAuthService(userRepo *repository.UserRepository, redisClient *redis.Client) *AuthService {
	oauthConfig := &oauth2.Config{
		ClientID:     config.AppConfig.GoogleClientID,
		ClientSecret: config.AppConfig.GoogleClientSecret,
		RedirectURL:  config.AppConfig.GoogleRedirectURL,
		Endpoint:     google.Endpoint,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}

	return &AuthService{
		userRepo:    userRepo,
		redisClient: redisClient,
		oauthConfig: oauthConfig,
	}
}

func (s *AuthService) Signup(ctx context.Context, req dto.SignupRequest) (*models.User, error) {
	existing, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, errors.New("user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:         req.Email,
		Password:      string(hashedPassword),
		Name:          req.Name,
		CreditBalance: 50.0, // Free trial credits
		APIKey:        generateAPIKey(),
	}

	err = s.userRepo.Create(ctx, user)
	return user, err
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (string, *models.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return "", nil, errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	token, err := s.GenerateToken(user)
	return token, user, err
}

func (s *AuthService) GenerateToken(user *models.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":    user.ID.Hex(),
		"email": user.Email,
		"exp":   time.Now().Add(time.Hour * 72).Unix(),
	})

	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}

// OTP Methods

func (s *AuthService) GenerateAndSendOTP(ctx context.Context, email string) error {
	// 1. Only existing users
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil || user == nil {
		return errors.New("user not found - OTP is only for existing users")
	}

	// 2. Generate 6-digit code
	code := fmt.Sprintf("%06d", rand.Intn(1000000))

	// 3. Store in Redis
	key := fmt.Sprintf("otp:%s", email)
	expiry := time.Duration(config.AppConfig.OTPExpiryMinutes) * time.Minute
	if err := s.redisClient.Set(ctx, key, code, expiry).Err(); err != nil {
		return fmt.Errorf("failed to store OTP: %w", err)
	}

	// 4. Send Email
	return Email.SendOTP(email, code)
}

func (s *AuthService) VerifyOTP(ctx context.Context, email string, code string) (string, *models.User, error) {
	key := fmt.Sprintf("otp:%s", email)
	val, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		return "", nil, errors.New("OTP expired or not found")
	}

	if val != code {
		return "", nil, errors.New("invalid OTP")
	}

	// Code match - clean up
	s.redisClient.Del(ctx, key)

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, err
	}

	token, err := s.GenerateToken(user)
	return token, user, err
}

// Google OAuth Methods

func (s *AuthService) GetGoogleLoginURL(state string) string {
	return s.oauthConfig.AuthCodeURL(state)
}

func (s *AuthService) HandleGoogleAuth(ctx context.Context, code string) (string, *models.User, error) {
	token, err := s.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return "", nil, fmt.Errorf("code exchange failed: %w", err)
	}

	// Fetch User info from Google
	oauth2Service, err := googleoauth2.NewService(ctx, option.WithTokenSource(s.oauthConfig.TokenSource(ctx, token)))
	if err != nil {
		return "", nil, fmt.Errorf("failed to create oauth2 service: %w", err)
	}

	userinfo, err := oauth2Service.Userinfo.Get().Do()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get userinfo: %w", err)
	}

	// Check if user exists
	user, err := s.userRepo.GetByEmail(ctx, userinfo.Email)
	if err != nil {
		// New user from Google
		user = &models.User{
			Email:         userinfo.Email,
			Name:          userinfo.Name,
			CreditBalance: 50.0,
			APIKey:        generateAPIKey(),
			Verified:      true,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return "", nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	// Generate Beth token
	bethToken, err := s.GenerateToken(user)
	return bethToken, user, err
}

func generateAPIKey() string {
	return "sk_" + uuid.New().String()
}
