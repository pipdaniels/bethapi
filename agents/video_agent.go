package agents

import (
	"context"
	"fmt"

	"bethapi/agents/tools"
	"bethapi/api/repository"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

type VideoAgent struct {
	runner *runner.Runner
}

func NewVideoAgent(genAIClient *genai.Client, userRepo *repository.UserRepository, jobRepo *repository.JobRepository) *VideoAgent {
	// 1. Create tools
	enhancer, _ := tools.NewPromptEnhancerTool(genAIClient)
	imagen, _ := tools.NewImagenTool(genAIClient, jobRepo)
	veo, _ := tools.NewVeoTool(genAIClient, jobRepo)
	tracker, _ := tools.NewUsageTrackerTool(userRepo, jobRepo)

	// 2. Create ADK Model wrapper
	adkModel, _ := gemini.NewModel(context.Background(), "gemini-2.0-flash", &genai.ClientConfig{})

	// 3. Create LLM Agent
	va, _ := llmagent.New(llmagent.Config{
		Name:        "VideoSubmitter",
		Description: "Orchestrates the submission of video generation tasks including prompt enhancement and storyboarding.",
		Instruction: `
			1. Use 'prompt_enhancer' to expand the user's simple prompt into a cinematic description.
			2. Use 'imagen_generate' with the enhanced prompt to createStoryboard multiple storyboard frames.
			3. Use 'veo_submit' with the enhanced prompt and the storyboard URLs to trigger the video generation.
			4. Use 'usage_tracker' to record the prompt tokens and the number of Imagen frames generated (4).
			5. Return a final summary including the provider_job_id from veo_submit and the storyboard URLs.
		`,
		Model: adkModel,
		Tools: []tool.Tool{enhancer, imagen, veo, tracker},
	})

	// 4. Initialize Runner
	// We'll use the default session service if NewInMemoryService is tricky
	r, _ := runner.New(runner.Config{
		AppName:           "BethAPI",
		Agent:             va,
		AutoCreateSession: true,
	})

	return &VideoAgent{runner: r}
}

func (va *VideoAgent) Run(ctx context.Context, userID string, jobID string, prompt string) (string, error) {
	// 1. Create user message
	msg := &genai.Content{
		Parts: []*genai.Part{{Text: prompt}},
	}

	// 2. Run the agent using the runner
	// ADK runner.Run handles session creation if userID/sessionID (jobID) are provided.
	var finalMessage string
	runStream := va.runner.Run(ctx, userID, jobID, msg, agent.RunConfig{})

	for event, err := range runStream {
		if err != nil {
			return "", err
		}
		if event.Author != "user" && event.LLMResponse != nil && !event.LLMResponse.Partial {
			// Extract text parts from the final message
			msgText := ""
			for _, part := range event.Content.Parts {
				if part.Text != "" {
					msgText += part.Text
				}
			}
			finalMessage = msgText
		}
	}

	if finalMessage == "" {
		return "", fmt.Errorf("agent failed to produce a final response")
	}

	return finalMessage, nil
}
