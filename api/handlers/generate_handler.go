package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"bethapi/api/dto"
	"bethapi/api/middleware"
	"bethapi/api/models"
	"bethapi/api/repository"
	"bethapi/worker"

	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GenerateHandler struct {
	asynqClient *asynq.Client
	jobRepo     *repository.JobRepository
}

func NewGenerateHandler(asynqClient *asynq.Client, jobRepo *repository.JobRepository) *GenerateHandler {
	return &GenerateHandler{
		asynqClient: asynqClient,
		jobRepo:     jobRepo,
	}
}

func (h *GenerateHandler) Generate(c echo.Context) error {
	var req dto.GenerateRequest
	if err := middleware.BindAndValidate(c, &req); err != nil {
		return err
	}

	user := c.Get("user").(models.User)

	// Set default mode if empty
	mode := req.GenerationMode
	if mode == "" {
		mode = "standard"
	}

	// 1. Create Job record in MongoDB
	job := &models.Job{
		ID:             primitive.NewObjectID(),
		UserID:         user.ID,
		Prompt:         req.Prompt,
		GenerationMode: mode,
		Status:         models.JobStatusPending,
		Progress:       0,
		AspectRatio:    req.AspectRatio,
		Duration:       req.Duration,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := h.jobRepo.Create(c.Request().Context(), job); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: "Failed to create job record"})
	}

	jobID := job.ID.Hex()
	requestID := c.Response().Header().Get(echo.HeaderXRequestID)

	// 2. Determine Queue based on Plan
	queue := "default"
	if user.Plan == models.PlanPro || user.Plan == models.PlanUltra {
		queue = "critical"
	} else if user.Plan == models.PlanFree {
		queue = "low"
	}

	// 3. Enqueue task to Asynq
	payload, _ := json.Marshal(worker.VideoGenerationPayload{
		TraceID:   requestID,
		UserID:    user.ID.Hex(),
		JobID:     jobID,
		Prompt:    req.Prompt,
	})

	task := asynq.NewTask(worker.TypeVideoGeneration, payload)
	_, err := h.asynqClient.Enqueue(task, asynq.Queue(queue))
	if err != nil {
		_ = h.jobRepo.MarkFailed(c.Request().Context(), job.ID, "Failed to enqueue task: "+err.Error())
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: "Failed to queue generation job"})
	}

	return c.JSON(http.StatusAccepted, dto.GenerateResponse{JobID: jobID})
}

func (h *GenerateHandler) Compose(c echo.Context) error {
	var req dto.ComposeRequest
	if err := middleware.BindAndValidate(c, &req); err != nil {
		return err
	}

	user := c.Get("user").(models.User)
	requestID := c.Response().Header().Get(echo.HeaderXRequestID)

	// 1. Convert DTO Clips to Model Clips
	compositionData := make([]models.CompositionClipData, len(req.Clips))
	for i, clip := range req.Clips {
		compositionData[i] = models.CompositionClipData{
			JobID:      clip.JobID,
			Transition: clip.Transition,
			Duration:   clip.Duration,
		}
	}

	// 2. Create Job record in MongoDB
	job := &models.Job{
		ID:              primitive.NewObjectID(),
		UserID:          user.ID,
		Type:            models.JobTypeComposition,
		Status:          models.JobStatusPending,
		Progress:        0,
		CompositionData: compositionData,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := h.jobRepo.Create(c.Request().Context(), job); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: "Failed to create composition job record"})
	}

	jobID := job.ID.Hex()

	// 3. Enqueue task to Asynq
	payload, _ := json.Marshal(worker.CompositionTaskPayload{
		TraceID:   requestID,
		JobID:     jobID,
	})

	task := asynq.NewTask(worker.TypeVideoComposition, payload)
	_, err := h.asynqClient.Enqueue(task)
	if err != nil {
		_ = h.jobRepo.MarkFailed(c.Request().Context(), job.ID, "Failed to enqueue task: "+err.Error())
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: "Failed to queue composition job"})
	}

	return c.JSON(http.StatusAccepted, dto.GenerateResponse{JobID: jobID})
}

// ListVideos returns a paginated list of videos created by the authenticated user
func (h *GenerateHandler) ListVideos(c echo.Context) error {
	var paginationReq dto.PaginationRequest
	if err := c.Bind(&paginationReq); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ApiResponse{
			Success: false,
			Error:   "Invalid pagination parameters",
		})
	}

	user := c.Get("user").(models.User)
	ctx := c.Request().Context()

	// Get paginated jobs for the user, only generation types with video URLs
	jobs, totalCount, err := h.jobRepo.ListByUserPaginated(ctx, user.ID, paginationReq.GetSkip(), paginationReq.GetLimit())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ApiResponse{
			Success: false,
			Error:   "Failed to retrieve videos",
		})
	}

	// Convert jobs to video responses, filtering for completed generation jobs
	videos := make([]dto.VideoResponse, 0)
	for _, job := range jobs {
		if job.Type == models.JobTypeGeneration && job.Status == models.JobStatusCompleted && job.VideoURL != "" {
			videos = append(videos, dto.VideoResponse{
				ID:        job.ID.Hex(),
				Title:     job.Prompt, // Use prompt as title, can be enhanced later
				Length:    job.Duration,
				VideoURL:  job.VideoURL,
				Status:    string(job.Status),
				CreatedAt: job.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	// Calculate pagination metadata
	pageSize := paginationReq.GetLimit()
	totalPages := int((totalCount + int64(pageSize-1)) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	pagination := &dto.PaginationMeta{
		CurrentPage: paginationReq.GetPage(),
		TotalPages:  totalPages,
		TotalItems:  totalCount,
		PageSize:    pageSize,
	}

	return dto.SendPaginatedResponse(c, http.StatusOK, "Videos retrieved successfully", videos, pagination)
}

