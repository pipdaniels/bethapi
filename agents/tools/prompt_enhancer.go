package tools

import (
	"fmt"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
)

func NewPromptEnhancerTool(client *genai.Client) (tool.Tool, error) {
	return functiontool.New[string, string](functiontool.Config{
		Name:        "prompt_enhancer",
		Description: "Rewrites a simple prompt into a detailed video scene description",
	}, func(tc tool.Context, userPrompt string) (string, error) {
		prompt := fmt.Sprintf("Act as a professional cinematographer. Re-write the following prompt into a highly detailed scene description for a high-quality video generator. Include lighting, camera movement, and textures: %s", userPrompt)

		// tool.Context implements context.Context
		resp, err := client.Models.GenerateContent(tc, "gemini-2.0-flash", genai.Text(prompt), nil)
		if err != nil {
			return "", err
		}

		if len(resp.Candidates) == 0 {
			return "", fmt.Errorf("no response from Gemini")
		}

		enhancedPrompt := ""
		for _, part := range resp.Candidates[0].Content.Parts {
			enhancedPrompt += fmt.Sprintf("%v", part)
		}

		return enhancedPrompt, nil
	})
}
