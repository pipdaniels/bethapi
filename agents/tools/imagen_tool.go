package tools

import (
	"fmt"
	"time"

	"bethapi/api/repository"
	"bethapi/api/services"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
)

// ImagenToolOutput returns a list of public R2 URLs for the generated storyboard frames.
type ImagenToolOutput struct {
	StoryboardURLs []string `json:"storyboard_urls"`
}

func NewImagenTool(client *genai.Client, jobRepo *repository.JobRepository) (tool.Tool, error) {
	return functiontool.New[string, ImagenToolOutput](functiontool.Config{
		Name:        "imagen_generate",
		Description: "Generates multiple high-quality storyboard frames for the video scene",
	}, func(tc tool.Context, prompt string) (ImagenToolOutput, error) {
		model := "imagen-3.0-generate-002"

		resp, err := client.Models.GenerateImages(tc, model, prompt, &genai.GenerateImagesConfig{
			NumberOfImages: 4,
			AspectRatio:    "16:9",
		})
		if err != nil {
			return ImagenToolOutput{}, fmt.Errorf("imagen generation failed: %w", err)
		}

		var urls []string
		for i, genImg := range resp.GeneratedImages {
			if genImg.Image == nil || len(genImg.Image.ImageBytes) == 0 {
				continue
			}

			// Generate a unique key for the R2 upload
			key := fmt.Sprintf("storyboards/%d-%d.png", time.Now().UnixNano(), i)
			
			// Upload to R2
			err := services.Storage.Upload(tc, key, genImg.Image.ImageBytes, "image/png")
			if err != nil {
				return ImagenToolOutput{}, fmt.Errorf("failed to upload storyboard %d: %w", i, err)
			}

			// Get public URL
			urls = append(urls, services.Storage.GetPublicURL(key))
		}

		return ImagenToolOutput{StoryboardURLs: urls}, nil
	})
}
