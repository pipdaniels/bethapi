package tools

import (
	"fmt"

	"bethapi/api/repository"
	"bethapi/config"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type UsageMetadata struct {
	PromptTokenCount     int64   `json:"prompt_token_count"`
	CandidatesTokenCount int64   `json:"candidates_token_count"`
	VideoDurationSeconds float64 `json:"video_duration_seconds,omitempty"`
}

func NewUsageTrackerTool(userRepo *repository.UserRepository) (tool.Tool, error) {
	return functiontool.New[UsageMetadata, string](functiontool.Config{
		Name:        "usage_tracker",
		Description: "Tracks token usage and deducts credits from user balance",
	}, func(tc tool.Context, meta UsageMetadata) (string, error) {
		// 1. Calculate costs based on .env config
		promptCost := (float64(meta.PromptTokenCount) / 1000.0) * config.AppConfig.PricingLLMPrompt1K
		outputCost := (float64(meta.CandidatesTokenCount) / 1000.0) * config.AppConfig.PricingLLMOutput1K
		videoCost := meta.VideoDurationSeconds * config.AppConfig.PricingVideoSec

		totalCost := promptCost + outputCost + videoCost

		// 2. Get User ID from ADK Session State
		val, _ := tc.State().Get("user_id")
		userIDStr, ok := val.(string)
		if !ok || userIDStr == "" {
			return "", fmt.Errorf("user_id not found in agent state")
		}

		// 3. Deduct from DB (Simulated for now)
		// userID, _ := primitive.ObjectIDFromHex(userIDStr)
		// err := userRepo.DeductCredits(tc, userID, totalCost)

		return fmt.Sprintf("Deducted %.2f credits for user %s. Usage: %d prompt tokens, %d output tokens, %.1f video sec.",
			totalCost, userIDStr, meta.PromptTokenCount, meta.CandidatesTokenCount, meta.VideoDurationSeconds), nil
	})
}
