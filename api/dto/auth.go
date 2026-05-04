package dto

import "bethapi/api/models"

type SignupRequest struct {
	Email    string `json:"email" form:"email" validate:"required,email"`
	Password string `json:"password" form:"password" validate:"required,min=8"`
	Name     string `json:"name" form:"name" validate:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" form:"email" validate:"required,email"`
	Password string `json:"password" form:"password" validate:"required"`
}

type OTPRequest struct {
	Email string `json:"email" form:"email" validate:"required,email"`
}

type OTPVerifyRequest struct {
	Email string `json:"email" form:"email" validate:"required,email"`
	Code  string `json:"code" form:"code" validate:"required,len=6"`
}

type AuthResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}
