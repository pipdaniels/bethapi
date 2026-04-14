package dto

type GenerateRequest struct {
	Prompt       string  `json:"prompt" validate:"required"`
	Duration     float64 `json:"duration"`
	AspectRatio  string  `json:"aspect_ratio"`
	RefImageURL  string  `json:"reference_image_url,omitempty"`
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
