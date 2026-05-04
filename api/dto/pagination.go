package dto

import "github.com/labstack/echo/v4"

// PaginationRequest contains pagination parameters for list endpoints
type PaginationRequest struct {
	Page  int `query:"page" validate:"omitempty,min=1"`
	Limit int `query:"limit" validate:"omitempty,min=1,max=100"`
}

// GetPage returns the page number, defaulting to 1 if not set or invalid
func (pr *PaginationRequest) GetPage() int {
	if pr.Page <= 0 {
		return 1
	}
	return pr.Page
}

// GetLimit returns the limit, defaulting to 10 if not set or invalid, capped at 100
func (pr *PaginationRequest) GetLimit() int {
	if pr.Limit <= 0 {
		return 10
	}
	if pr.Limit > 100 {
		return 100
	}
	return pr.Limit
}

// GetSkip calculates the skip offset for database queries
func (pr *PaginationRequest) GetSkip() int64 {
	return int64((pr.GetPage() - 1) * pr.GetLimit())
}

// PaginationMeta contains metadata about the paginated response
type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	TotalPages  int   `json:"total_pages"`
	TotalItems  int64 `json:"total_items"`
	PageSize    int   `json:"page_size"`
}

// ApiResponse is a generic response wrapper for all API endpoints
type ApiResponse struct {
	Success    bool            `json:"success"`
	Message    string          `json:"message,omitempty"`
	Data       interface{}     `json:"data,omitempty"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
	Error      string          `json:"error,omitempty"`
}

// PaginatedResponse sends a successful paginated response
func SendPaginatedResponse(c echo.Context, statusCode int, message string, data interface{}, pagination *PaginationMeta) error {
	return c.JSON(statusCode, ApiResponse{
		Success:    true,
		Message:    message,
		Data:       data,
		Pagination: pagination,
	})
}

// SuccessResponse sends a successful response
func SendSuccessResponse(c echo.Context, statusCode int, message string, data interface{}) error {
	return c.JSON(statusCode, ApiResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// ErrorResponse sends an error response
func SendErrorResponse(c echo.Context, statusCode int, message string, err string) error {
	return c.JSON(statusCode, ApiResponse{
		Success: false,
		Message: message,
		Error:   err,
	})
}
