package handlers

import (
	"net/http"

	"bethapi/api/dto"
	"bethapi/api/services"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	authService *services.AuthService
	validator   *validator.Validate
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		validator:   validator.New(),
	}
}

func (h *AuthHandler) Signup(c echo.Context) error {
	var req dto.SignupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "Invalid input"})
	}

	if err := h.validator.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
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
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "Invalid input"})
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

func (h *AuthHandler) GetMe(c echo.Context) error {
	user := c.Get("user")
	return c.JSON(http.StatusOK, user)
}
