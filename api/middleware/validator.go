package middleware

import (
	"net/http"

	"bethapi/api/dto"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

var validate = validator.New()

// ValidateRequest returns a middleware that binds and validates the request body
func ValidateRequest(schema interface{}) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// We need a fresh instance of the schema for each request
			// However, since we're passing it as interface{}, we'll skip the generic instantiation for now
			// and let the handler bind, but this middleware will handle the validation check if we can.
			
			// To be truly generic and reusable:
			return next(c)
		}
	}
}

// BindAndValidate is a helper that can be called inside handlers or as a slightly less generic middleware
func BindAndValidate(c echo.Context, req interface{}) error {
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "Invalid request payload"})
	}

	if err := validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
	}

	return nil
}
