package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"bethapi/api/models"
	"bethapi/api/repository"
	"bethapi/api/services"

	"github.com/hibiken/asynq"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const TypeVideoComposition = "video:compose"

type CompositionTaskPayload struct {
	TraceID string `json:"trace_id"`
	JobID   string `json:"job_id"`
}

type CompositionWorker struct {
	jobRepo      *repository.JobRepository
	userRepo     *repository.UserRepository
	videoService *services.VideoService
	storage      *services.StorageService
}

func NewCompositionWorker(jobRepo *repository.JobRepository, userRepo *repository.UserRepository, videoService *services.VideoService, storage *services.StorageService) *CompositionWorker {
	return &CompositionWorker{
		jobRepo:      jobRepo,
		userRepo:     userRepo,
		videoService: videoService,
		storage:      storage,
	}
}

func (w *CompositionWorker) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p CompositionTaskPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	log.Printf("[TRACE-%s] Starting video composition for JobID: %s", p.TraceID, p.JobID)

	jobID, err := primitive.ObjectIDFromHex(p.JobID)
	if err != nil {
		return fmt.Errorf("invalid job ID: %w", err)
	}

	// 1. Fetch Job
	job, err := w.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	w.jobRepo.UpdateStatus(ctx, jobID, models.JobStatusProcessing, 0.1)

	// 2. Create Temp Workspace
	tempDir, err := os.MkdirTemp("", "composition-"+p.JobID)
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 3. Download and Standardize Clips
	compositionClips := []services.CompositionClip{}
	for i, clipData := range job.CompositionData {
		log.Printf("[TRACE-%s] Processing clip %d/%d (Source Job: %s)", p.TraceID, i+1, len(job.CompositionData), clipData.JobID)
		
		sourceJobID, _ := primitive.ObjectIDFromHex(clipData.JobID)
		sourceJob, err := w.jobRepo.GetByID(ctx, sourceJobID)
		if err != nil {
			return fmt.Errorf("failed to get source job %s: %w", clipData.JobID, err)
		}

		if sourceJob.VideoURL == "" {
			return fmt.Errorf("source job %s has no video URL", clipData.JobID)
		}

		// Download to temp
		key := filepath.Base(sourceJob.VideoURL)
		localPath := filepath.Join(tempDir, fmt.Sprintf("source_%d_%s", i, key))

		if err := w.storage.Download(ctx, "vids/"+key, localPath); err != nil {
			return fmt.Errorf("failed to download clip %s: %w", key, err)
		}

		// Standardize
		standardPath := filepath.Join(tempDir, fmt.Sprintf("std_%d_%s", i, key))
		if err := w.videoService.StandardizeClip(ctx, localPath, standardPath); err != nil {
			return fmt.Errorf("failed to standardize clip %d: %w", i, err)
		}

		compositionClips = append(compositionClips, services.CompositionClip{
			Path:       standardPath,
			Transition: clipData.Transition,
			Duration:   clipData.Duration,
		})

		w.jobRepo.UpdateStatus(ctx, jobID, models.JobStatusProcessing, 0.1+(float64(i+1)/float64(len(job.CompositionData)))*0.4)
	}

	// 4. Run Composition
	log.Printf("[TRACE-%s] Running final FFmpeg composition for JobID: %s", p.TraceID, p.JobID)
	finalLocalPath := filepath.Join(tempDir, "final_composition.mp4")
	if err := w.videoService.ComposeVideos(ctx, compositionClips, finalLocalPath); err != nil {
		return fmt.Errorf("video composition failed: %w", err)
	}

	w.jobRepo.UpdateStatus(ctx, jobID, models.JobStatusProcessing, 0.8)

	// 5. Upload Final Video
	log.Printf("[TRACE-%s] Uploading composed video for JobID: %s", p.TraceID, p.JobID)
	finalKey := fmt.Sprintf("vids/composed_%s.mp4", p.JobID)
	videoData, err := os.ReadFile(finalLocalPath)
	if err != nil {
		return fmt.Errorf("failed to read final video: %w", err)
	}

	if err := w.storage.Upload(ctx, finalKey, videoData, "video/mp4"); err != nil {
		return fmt.Errorf("failed to upload final video: %w", err)
	}

	finalURL := w.storage.GetPublicURL(finalKey)
	w.jobRepo.MarkCompleted(ctx, jobID, finalURL, 0) // Composition credits TBD

	log.Printf("[TRACE-%s] Composition completed successfully for JobID: %s", p.TraceID, p.JobID)
	return nil
}
