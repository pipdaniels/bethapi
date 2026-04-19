package tools

import (
	"fmt"
	"io"
	"net/http"

	"bethapi/api/repository"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// VeoToolOutput returns the LRO Name for the video generation job.
type VeoToolOutput struct {
	ProviderJobID string `json:"provider_job_id"`
}

func NewVeoTool(client *genai.Client, jobRepo *repository.JobRepository) (tool.Tool, error) {
	return functiontool.New[map[string]interface{}, VeoToolOutput](functiontool.Config{
		Name:        "veo_submit",
		Description: "Submits an enhanced prompt and storyboard frames to the Veo 3.0 video generation engine",
	}, func(tc tool.Context, params map[string]interface{}) (VeoToolOutput, error) {
		prompt, _ := params["prompt"].(string)
		storyboardURLs, _ := params["storyboard_urls"].([]string)
		jobIDStr, _ := tc.State().Get("job_id")

		// Model Selection based on Job Mode
		model := "veo-3.0-generate-preview" 
		if id, ok := jobIDStr.(string); ok && id != "" {
			objID, _ := primitive.ObjectIDFromHex(id)
			if job, err := jobRepo.GetByID(tc, objID); err == nil && job.GenerationMode == "fast" {
				model = "veo-3.0-generate-preview" // Can be switched to a 'fast' variant
			}
		}
		
		var startImage *genai.Image
		var lastImage *genai.Image

		if len(storyboardURLs) > 0 {
			// Download the FIRST frame for the start of the video
			resp, err := http.Get(storyboardURLs[0])
			if err == nil {
				defer resp.Body.Close()
				imgBytes, _ := io.ReadAll(resp.Body)
				startImage = &genai.Image{
					ImageBytes: imgBytes,
					MIMEType:   "image/png",
				}
			}

			// Download the LAST frame for the end of the video (if multiple frames exist)
			if len(storyboardURLs) > 1 {
				lResp, err := http.Get(storyboardURLs[len(storyboardURLs)-1])
				if err == nil {
					defer lResp.Body.Close()
					lBytes, _ := io.ReadAll(lResp.Body)
					lastImage = &genai.Image{
						ImageBytes: lBytes,
						MIMEType:   "image/png",
					}
				}
			}
		}

		// Submit the generation request
		operation, err := client.Models.GenerateVideos(tc, model, prompt, startImage, &genai.GenerateVideosConfig{
			AspectRatio: "16:9",
			LastFrame:   lastImage, // Chronological reference (end frame)
		})
		if err != nil {
			return VeoToolOutput{}, fmt.Errorf("veo submission failed: %w", err)
		}

		// Update Job record with ProviderJobID
		if id, ok := jobIDStr.(string); ok && id != "" {
			objID, _ := primitive.ObjectIDFromHex(id)
			_ = jobRepo.UpdateProviderJobID(tc, objID, operation.Name)
		}

		return VeoToolOutput{ProviderJobID: operation.Name}, nil
	})
}
