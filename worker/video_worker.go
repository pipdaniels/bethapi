package worker

import (
	"context"
	"encoding/json"
	"log"

	"bethapi/agents"
	"bethapi/api/models"
	"bethapi/api/repository"
	"bethapi/api/services"
	"bethapi/billing"
	"github.com/hibiken/asynq"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/genai"
)

const (
	TypeVideoGeneration = "video:generate"
)

type VideoGenerationPayload struct {
	TraceID string `json:"trace_id"`
	UserID  string `json:"user_id"`
	JobID   string `json:"job_id"`
	Prompt  string `json:"prompt"`
}

type VideoWorker struct {
	agent         *agents.VideoAgent
	jobRepo       *repository.JobRepository
	creditService *billing.CreditService
}

func NewVideoWorker(genAIClient *genai.Client, userRepo *repository.UserRepository, jobRepo *repository.JobRepository, creditService *billing.CreditService) *VideoWorker {
	return &VideoWorker{
		agent:         agents.NewVideoAgent(genAIClient, userRepo, jobRepo),
		jobRepo:       jobRepo,
		creditService: creditService,
	}
}

// publishState is a helper that updates MongoDB and then pushes the new state
// to the Redis pub/sub channel in one call.
func (w *VideoWorker) publishState(ctx context.Context, job *models.Job) {
	services.PublishJobUpdate(ctx, job)
}

func (w *VideoWorker) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p VideoGenerationPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	log.Printf("[TRACE-%s] Starting video submission for JobID: %s", p.TraceID, p.JobID)

	jobObjID, err := primitive.ObjectIDFromHex(p.JobID)
	if err != nil {
		return err
	}

	// 1. Transition to 'processing' (Submitter Phase)
	if err := w.jobRepo.UpdateStatus(ctx, jobObjID, models.JobStatusProcessing, 0.1); err != nil {
		log.Printf("[TRACE-%s] Warning: failed to update job status for %s: %v", p.TraceID, p.JobID, err)
	}
	
	if job, err := w.jobRepo.GetByID(ctx, jobObjID); err == nil {
		w.publishState(ctx, job)
	}

	// 2. Run the agent (Enhancer -> Storyboard -> Veo Submit -> Usage Tracker)
	summary, err := w.agent.Run(ctx, p.UserID, p.JobID, p.Prompt)
	if err != nil {
		log.Printf("[TRACE-%s] Error during agent submission for JobID %s: %v", p.TraceID, p.JobID, err)

		// Transition to failed
		_ = w.jobRepo.MarkFailed(ctx, jobObjID, err.Error())
		
		// Attempt Refund
		if refundErr := w.creditService.RefundCredits(ctx, jobObjID); refundErr != nil {
			log.Printf("[TRACE-%s] Warning: failed to refund credits for failed job %s: %v", p.TraceID, p.JobID, refundErr)
		}

		if job, fetchErr := w.jobRepo.GetByID(ctx, jobObjID); fetchErr == nil {
			w.publishState(ctx, job)
		}
		return err
	}

	log.Printf("[TRACE-%s] Job %s submitted successfully to provider. Summary: %s", p.TraceID, p.JobID, summary)
	
	// Final state for this worker: The job is still 'processing' until the Poller picks it up.
	if job, err := w.jobRepo.GetByID(ctx, jobObjID); err == nil {
		w.publishState(ctx, job)
	}

	return nil
}
