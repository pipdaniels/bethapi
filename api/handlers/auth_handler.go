package handlers

import (
	"fmt"
	"net/http"
	"time"

	"bethapi/api/dto"
	"bethapi/api/middleware"
	"bethapi/api/services"
	"bethapi/config"

	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) Signup(c echo.Context) error {
	var req dto.SignupRequest
	if err := middleware.BindAndValidate(c, &req); err != nil {
		return err
	}

	user, err := h.authService.Signup(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: err.Error()})
	}

	token, _ := h.authService.GenerateToken(user)
	return c.JSON(http.StatusCreated, dto.AuthResponse{
		Token: token,
		User:  *user,
	})
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := middleware.BindAndValidate(c, &req); err != nil {
		return err
	}

	token, user, err := h.authService.Login(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, dto.AuthResponse{
		Token: token,
		User:  *user,
	})
}

// OTP Handlers

func (h *AuthHandler) SendOTP(c echo.Context) error {
	var req dto.OTPRequest
	if err := middleware.BindAndValidate(c, &req); err != nil {
		return err
	}

	err := h.authService.GenerateAndSendOTP(c.Request().Context(), req.Email)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "OTP sent successfully"})
}

func (h *AuthHandler) VerifyOTP(c echo.Context) error {
	var req dto.OTPVerifyRequest
	if err := middleware.BindAndValidate(c, &req); err != nil {
		return err
	}

	token, user, err := h.authService.VerifyOTP(c.Request().Context(), req.Email, req.Code)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, dto.AuthResponse{
		Token: token,
		User:  *user,
	})
}

// Google OAuth Handlers

func (h *AuthHandler) GoogleLogin(c echo.Context) error {
	state := fmt.Sprintf("%d", time.Now().UnixNano()) // In production, use a more secure state (e.g. signed cookie)
	url := h.authService.GetGoogleLoginURL(state)
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

func (h *AuthHandler) GoogleCallback(c echo.Context) error {
	code := c.QueryParam("code")
	if code == "" {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "Missing code"})
	}

	token, _, err := h.authService.HandleGoogleAuth(c.Request().Context(), code)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Message: err.Error()})
	}

	// Set HttpOnly Cookie
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		Domain:   config.AppConfig.CookieDomain,
		HttpOnly: true,
		Secure:   config.AppConfig.Env == "production",
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(72 * time.Hour),
	}
	c.SetCookie(cookie)

	return c.Redirect(http.StatusTemporaryRedirect, config.AppConfig.FrontendURL)
}

func (h *AuthHandler) GetMe(c echo.Context) error {
	user := c.Get("user")
	return c.JSON(http.StatusOK, user)
}
