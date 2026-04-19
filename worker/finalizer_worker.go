package worker

import (
	"bethapi/api/repository"
	"bethapi/api/services"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/hibiken/asynq"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	TypeVideoFinalize = "video:finalize"
)

type VideoFinalizePayload struct {
	JobID    string `json:"job_id"`
	VideoURL string `json:"video_url"` // Provider's temporary URL or GCS URI
}

type FinalizerWorker struct {
	jobRepo  *repository.JobRepository
	userRepo *repository.UserRepository
}

func NewFinalizerWorker(jobRepo *repository.JobRepository, userRepo *repository.UserRepository) *FinalizerWorker {
	return &FinalizerWorker{
		jobRepo:  jobRepo,
		userRepo: userRepo,
	}
}

func (w *FinalizerWorker) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p VideoFinalizePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	log.Printf("Finalizing video for JobID: %s", p.JobID)

	jobObjID, err := primitive.ObjectIDFromHex(p.JobID)
	if err != nil {
		return err
	}

	job, err := w.jobRepo.GetByID(ctx, jobObjID)
	if err != nil {
		return err
	}

	// 1. Download video from provider if it's a URL
	resp, err := http.Get(p.VideoURL)
	if err != nil {
		return fmt.Errorf("failed to download video from provider: %w", err)
	}
	defer resp.Body.Close()

	videoBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read video bytes: %w", err)
	}

	// 2. Upload to R2
	key := fmt.Sprintf("videos/%s.mp4", p.JobID)
	err = services.Storage.Upload(ctx, key, videoBytes, "video/mp4")
	if err != nil {
		return fmt.Errorf("failed to upload video to R2: %w", err)
	}

	finalPublicURL := services.Storage.GetPublicURL(key)

	// 3. Update Job Record
	// Note: CreditsUsed should already be partially tracked or we calculate final here.
	// For now, we mark as completed. UsageTrackerTool in Submitter already deducted initial costs.
	// Finalizer could deduct additional costs if duration was unknown.
	if err := w.jobRepo.MarkCompleted(ctx, jobObjID, finalPublicURL, job.CreditsUsed); err != nil {
		return err
	}

	// 4. Publish final state for SSE
	updatedJob, _ := w.jobRepo.GetByID(ctx, jobObjID)
	services.PublishJobUpdate(ctx, updatedJob)

	log.Printf("Job %s finalized successfully. Public URL: %s", p.JobID, finalPublicURL)
	return nil
}
