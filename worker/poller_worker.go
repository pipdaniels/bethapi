package worker

import (
	"context"
	"encoding/json"
	"log"

	"bethapi/api/repository"
	"bethapi/billing"
	"github.com/hibiken/asynq"
	"google.golang.org/genai"
)

type PollerWorker struct {
	genAIClient   *genai.Client
	jobRepo       *repository.JobRepository
	asynqClient   *asynq.Client
	creditService *billing.CreditService
}

func NewPollerWorker(genAIClient *genai.Client, jobRepo *repository.JobRepository, asynqClient *asynq.Client, creditService *billing.CreditService) *PollerWorker {
	return &PollerWorker{
		genAIClient:   genAIClient,
		jobRepo:       jobRepo,
		asynqClient:   asynqClient,
		creditService: creditService,
	}
}

// CheckProcessingJobs iterates through all jobs in 'processing' state and updates their status.
func (w *PollerWorker) CheckProcessingJobs(ctx context.Context) error {
	jobs, err := w.jobRepo.FindProcessingJobs(ctx)
	if err != nil {
		log.Printf("Poller: failed to fetch processing jobs: %v", err)
		return err
	}

	for _, job := range jobs {
		if job.ProviderJobID == "" {
			continue
		}

		log.Printf("Poller: Checking status for Job %s (LRO: %s)", job.ID.Hex(), job.ProviderJobID)

		// Poll the operation status
		op := &genai.GenerateVideosOperation{Name: job.ProviderJobID}
		result, err := w.genAIClient.Operations.GetVideosOperation(ctx, op, nil)
		if err != nil {
			log.Printf("Poller: Error polling LRO %s: %v", job.ProviderJobID, err)
			continue
		}

		// Check if Done
		// In some SDK versions, it's result.Done (bool)
		// I'll check the metadata/result fields
		// Wait, research said GenerateVideosOperation has a way to check if done.
		
		if result.Done {
			log.Printf("Poller: Job %s is DONE!", job.ID.Hex())
			
			// Transfer to Finalizer
			var videoURL string
			if result.Response != nil && len(result.Response.GeneratedVideos) > 0 {
				videoURL = result.Response.GeneratedVideos[0].Video.URI
			}

			if videoURL == "" {
				log.Printf("Poller: Job %s completed but no video URL found", job.ID.Hex())
				_ = w.jobRepo.MarkFailed(ctx, job.ID, "Completion returned no video URL")
				
				// Attempt Refund
				if refundErr := w.creditService.RefundCredits(ctx, job.ID); refundErr != nil {
					log.Printf("Warning: failed to refund credits for job %s: %v", job.ID.Hex(), refundErr)
				}
				continue
			}

			w.enqueueFinalizer(job.ID.Hex(), videoURL)
		} else if result.Error != nil {
			log.Printf("Poller: Job %s failed at provider level: %v", job.ID.Hex(), result.Error)
			errMsg := "Unknown provider error"
			if m, ok := result.Error["message"].(string); ok {
				errMsg = m
			}
			_ = w.jobRepo.MarkFailed(ctx, job.ID, errMsg)
			
			// Attempt Refund
			if refundErr := w.creditService.RefundCredits(ctx, job.ID); refundErr != nil {
				log.Printf("Warning: failed to refund credits for job %s: %v", job.ID.Hex(), refundErr)
			}
		} else {
			log.Printf("Poller: Job %s still in progress...", job.ID.Hex())
		}
	}

	return nil
}

func (w *PollerWorker) enqueueFinalizer(jobID, videoURL string) {
	payload, _ := json.Marshal(VideoFinalizePayload{
		JobID:    jobID,
		VideoURL: videoURL,
	})
	
	task := asynq.NewTask(TypeVideoFinalize, payload)
	if _, err := w.asynqClient.Enqueue(task); err != nil {
		log.Printf("Poller: Failed to enqueue finalizer for %s: %v", jobID, err)
	}
}
