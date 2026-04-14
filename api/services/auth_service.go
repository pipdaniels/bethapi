package services

import (
	"context"
	"errors"
	"time"

	"bethapi/api/dto"
	"bethapi/api/models"
	"bethapi/api/repository"
	"bethapi/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo *repository.UserRepository
}

func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
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

func generateAPIKey() string {
	return "sk_" + uuid.New().String()
}
