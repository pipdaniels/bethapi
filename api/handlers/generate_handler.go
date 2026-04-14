package handlers

import (
	"encoding/json"
	"net/http"

	"bethapi/api/dto"
	"bethapi/api/models"
	"bethapi/worker"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
)

type GenerateHandler struct {
	asynqClient *asynq.Client
}

func NewGenerateHandler(asynqClient *asynq.Client) *GenerateHandler {
	return &GenerateHandler{asynqClient: asynqClient}
}

func (h *GenerateHandler) Generate(c echo.Context) error {
	var req dto.GenerateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "Invalid input"})
	}

	user := c.Get("user").(models.User)
	jobID := uuid.New().String()

	// 1. Create Job record in Mongo (Future: Implement Job Repository)
	// database.GetCollection("jobs").InsertOne(c.Request().Context(), bson.M{"_id": jobID, "user_id": user.ID, "status": "pending", "prompt": req.Prompt})

	// 2. Enqueue task to Asynq
	payload, _ := json.Marshal(worker.VideoGenerationPayload{
		UserID: user.ID.Hex(),
		JobID:  jobID,
		Prompt: req.Prompt,
	})

	task := asynq.NewTask(worker.TypeVideoGeneration, payload)
	_, err := h.asynqClient.Enqueue(task)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: "Failed to queue generation job"})
	}

	return c.JSON(http.StatusAccepted, dto.GenerateResponse{JobID: jobID})
}
