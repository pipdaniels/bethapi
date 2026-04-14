package middleware

import (
	"net/http"
	"strings"

	"bethapi/api/database"
	"bethapi/api/dto"
	"bethapi/api/models"
	"bethapi/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
)

func JWTMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Message: "Missing authorization header"})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Message: "Invalid authorization header format"})
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.AppConfig.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Message: "Invalid or expired token"})
		}

		claims := token.Claims.(jwt.MapClaims)
		email := claims["email"].(string)

		// Get user from DB and attach to context
		var user models.User
		err = database.GetCollection("users").FindOne(c.Request().Context(), bson.M{"email": email}).Decode(&user)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Message: "User not found"})
		}

		c.Set("user", user)
		return next(c)
	}
}

func APIKeyMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		apiKey := c.Request().Header.Get("X-API-KEY")
		if apiKey == "" {
			return next(c) // Fallback to JWT if needed, or handle separately in routes
		}

		var user models.User
		err := database.GetCollection("users").FindOne(c.Request().Context(), bson.M{"api_key": apiKey}).Decode(&user)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Message: "Invalid API Key"})
		}

		c.Set("user", user)
		return next(c)
	}
}

func CombinedAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Try API Key first
		apiKey := c.Request().Header.Get("X-API-KEY")
		if apiKey != "" {
			var user models.User
			err := database.GetCollection("users").FindOne(c.Request().Context(), bson.M{"api_key": apiKey}).Decode(&user)
			if err == nil {
				c.Set("user", user)
				return next(c)
			}
		}

		// Fallback to JWT
		return JWTMiddleware(next)(c)
	}
}

func CreditCheckMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)
		if user.CreditBalance <= 0 {
			return c.JSON(http.StatusPaymentRequired, dto.ErrorResponse{Message: "Insufficient credits. Please top up."})
		}
		return next(c)
	}
}
