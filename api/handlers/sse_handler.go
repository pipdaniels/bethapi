package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"bethapi/api/dto"
	"bethapi/api/models"
	"bethapi/api/services"
	"github.com/labstack/echo/v4"
)

func (h *GenerateHandler) JobStream(c echo.Context) error {
	jobID := c.Param("id")
	ctx := c.Request().Context()

	// Set SSE headers
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
	c.Response().Header().Set(echo.HeaderConnection, "keep-alive")
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")

	// 1. Validate the job exists and send current state immediately
	job, err := h.jobRepo.GetByStringID(ctx, jobID)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Message: "Job not found"})
	}

	sendEvent := func(j *models.Job) {
		data, _ := json.Marshal(j)
		fmt.Fprintf(c.Response(), "data: %s\n\n", string(data))
		c.Response().Flush()
	}

	sendEvent(job)

	// If already terminal, close the stream right away
	if job.Status == models.JobStatusCompleted || job.Status == models.JobStatusFailed {
		return nil
	}

	// 2. Subscribe to the Redis pub/sub channel for this job
	sub := services.SubscribeJobUpdates(ctx, jobID)
	defer sub.Close()

	ch := sub.Channel()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				// Channel closed (Redis disconnected)
				return nil
			}
			var updated models.Job
			if err := json.Unmarshal([]byte(msg.Payload), &updated); err != nil {
				continue
			}
			sendEvent(&updated)

			// Close stream once job reaches a terminal state
			if updated.Status == models.JobStatusCompleted || updated.Status == models.JobStatusFailed {
				return nil
			}

		case <-ctx.Done():
			// Client disconnected
			return nil
		}
	}
}
