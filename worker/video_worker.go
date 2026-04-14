package worker

import (
	"context"
	"encoding/json"
	"log"

	"bethapi/agents"
	"bethapi/api/repository"
	"github.com/hibiken/asynq"
	"google.golang.org/genai"
)

const (
	TypeVideoGeneration = "video:generate"
)

type VideoGenerationPayload struct {
	UserID string `json:"user_id"`
	JobID  string `json:"job_id"`
	Prompt string `json:"prompt"`
}

type VideoWorker struct {
	agent *agents.VideoAgent
}

func NewVideoWorker(genAIClient *genai.Client, userRepo *repository.UserRepository) *VideoWorker {
	return &VideoWorker{
		agent: agents.NewVideoAgent(genAIClient, userRepo),
	}
}

func (w *VideoWorker) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p VideoGenerationPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	log.Printf("Processing video generation for JobID: %s", p.JobID)

	_, err := w.agent.Run(ctx, p.UserID, p.JobID, p.Prompt)
	if err != nil {
		log.Printf("Error running agent for JobID %s: %v", p.JobID, err)
		return err
	}

	return nil
}
