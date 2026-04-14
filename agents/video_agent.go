package agents

import (
	"context"

	"bethapi/agents/tools"
	"bethapi/api/repository"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

type VideoAgent struct {
	agent agent.Agent
}

func NewVideoAgent(genAIClient *genai.Client, userRepo *repository.UserRepository) *VideoAgent {
	// Create tools
	enhancer, _ := tools.NewPromptEnhancerTool(genAIClient)
	tracker, _ := tools.NewUsageTrackerTool(userRepo)

	// Create ADK Model wrapper
	// ADK's gemini.NewModel takes a *genai.ClientConfig
	adkModel, _ := gemini.NewModel(context.Background(), "gemini-2.0-flash", &genai.ClientConfig{
		// You can pass specific config here if needed, 
		// but typically it uses the same as your genAIClient
	})

	// Create LLM Agent
	va, _ := llmagent.New(llmagent.Config{
		Name:        "VideoOrchestrator",
		Description: "Professional video production agent",
		Instruction: "Follow these steps: 1. Enhance the prompt using prompt_enhancer. 2. Track usage.",
		Model:       adkModel,
		Tools:       []tool.Tool{enhancer, tracker},
	})

	return &VideoAgent{agent: va}
}

func (va *VideoAgent) Run(ctx context.Context, userID string, jobID string, prompt string) (string, error) {
	return "Video generation started for job " + jobID, nil
}
