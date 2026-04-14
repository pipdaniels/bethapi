package handlers

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
)

func (h *GenerateHandler) JobStream(c echo.Context) error {
	jobID := c.Param("id")

	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
	c.Response().Header().Set(echo.HeaderConnection, "keep-alive")
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")

	// Set a timeout or wait for job completion
	// In a real app, I'd use Redis Pub/Sub to listen for job updates
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	fmt.Fprintf(c.Response(), "data: %s\n\n", "{\"status\": \"initializing\", \"progress\": 0}")
	c.Response().Flush()

	for {
		select {
		case <-ticker.C:
			// Mock progress for now
			fmt.Fprintf(c.Response(), "data: %s\n\n", fmt.Sprintf("{\"job_id\": \"%s\", \"status\": \"processing\", \"progress\": 45}", jobID))
			c.Response().Flush()
		case <-c.Request().Context().Done():
			return nil
		}
	}
}
