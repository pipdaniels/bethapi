package dto

type GenerateRequest struct {
	Prompt         string  `json:"prompt" validate:"required,min=10"`
	GenerationMode string  `json:"generation_mode" validate:"omitempty,oneof=fast standard"`
	Duration       float64 `json:"duration"`
	AspectRatio    string  `json:"aspect_ratio"`
	RefImageURL    string  `json:"reference_image_url,omitempty"`
}

type GenerateResponse struct {
	JobID string `json:"job_id"`
}

type JobStatusResponse struct {
	JobID      string  `json:"job_id"`
	Status     string  `json:"status"`
	Progress   float64 `json:"progress"`
	VideoURL   string  `json:"video_url,omitempty"`
	CreditsUsed float64 `json:"credits_used"`
}

type ComposeRequest struct {
	Clips []struct {
		JobID      string  `json:"job_id" validate:"required"`
		Transition string  `json:"transition,omitempty"` // e.g., "fade", "wipeleft"
		Duration   float64 `json:"duration,omitempty"`   // transition duration in seconds
	} `json:"clips" validate:"required,min=1"`
	OutputName string `json:"output_name,omitempty"`
}
