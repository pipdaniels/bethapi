package tools

import (
	"fmt"

	"bethapi/api/repository"
	"bethapi/config"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UsageMetadata struct {
	PromptTokenCount     int64   `json:"prompt_token_count"`
	CandidatesTokenCount int64   `json:"candidates_token_count"`
	StoryboardFrameCount int     `json:"storyboard_frame_count,omitempty"`
	VideoDurationSeconds float64 `json:"video_duration_seconds,omitempty"`
}

func NewUsageTrackerTool(userRepo *repository.UserRepository, jobRepo *repository.JobRepository) (tool.Tool, error) {
	return functiontool.New[UsageMetadata, string](functiontool.Config{
		Name:        "usage_tracker",
		Description: "Tracks token usage and deducts credits from user balance",
	}, func(tc tool.Context, meta UsageMetadata) (string, error) {
		// 1. Get Job ID from ADK Session State
		valJob, _ := tc.State().Get("job_id")
		jobIDStr, okJob := valJob.(string)
		if !okJob || jobIDStr == "" {
			return "", fmt.Errorf("job_id not found in agent state")
		}
		jobID, _ := primitive.ObjectIDFromHex(jobIDStr)

		// 2. Fetch Job to get Mode
		job, err := jobRepo.GetByID(tc, jobID)
		if err != nil {
			return "", fmt.Errorf("failed to fetch job: %w", err)
		}

		// 3. Determine Video Price
		videoPrice := config.AppConfig.PricingVideoSecStd
		if job.GenerationMode == "fast" {
			videoPrice = config.AppConfig.PricingVideoSecFast
		}

		// 4. Calculate costs
		promptCost := (float64(meta.PromptTokenCount) / 1000.0) * config.AppConfig.PricingLLMPrompt1K
		outputCost := (float64(meta.CandidatesTokenCount) / 1000.0) * config.AppConfig.PricingLLMOutput1K
		storyboardCost := float64(meta.StoryboardFrameCount) * config.AppConfig.PricingImagen
		videoCost := meta.VideoDurationSeconds * videoPrice

		totalCost := promptCost + outputCost + storyboardCost + videoCost

		// 5. Get User ID from ADK Session State
		val, _ := tc.State().Get("user_id")
		userIDStr, ok := val.(string)
		if !ok || userIDStr == "" {
			return "", fmt.Errorf("user_id not found in agent state")
		}
		userID, _ := primitive.ObjectIDFromHex(userIDStr)

		// 6. Deduct from User and Record in Job
		if err := userRepo.DeductCredits(tc, userID, totalCost); err != nil {
			return "", fmt.Errorf("credit deduction failed: %w", err)
		}

		_ = jobRepo.UpdateCreditsDeducted(tc, jobID, totalCost)

		return fmt.Sprintf("Deducted %.2f credits for user %s. Usage: %d prompt tokens, %d output tokens, %d storyboard frames, %.1f video sec.",
			totalCost, userIDStr, meta.PromptTokenCount, meta.CandidatesTokenCount, meta.StoryboardFrameCount, meta.VideoDurationSeconds), nil
	})
}
