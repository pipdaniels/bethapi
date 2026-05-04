package dto

type GenerateRequest struct {
	Prompt         string  `json:"prompt" form:"prompt" validate:"required,min=10"`
	GenerationMode string  `json:"generation_mode" form:"generation_mode" validate:"omitempty,oneof=fast standard"`
	Duration       float64 `json:"duration" form:"duration"`
	AspectRatio    string  `json:"aspect_ratio" form:"aspect_ratio"`
	RefImageURL    string  `json:"reference_image_url" form:"reference_image_url,omitempty"`
}

type GenerateResponse struct {
	JobID string `json:"job_id"`
}

type JobStatusResponse struct {
	JobID       string  `json:"job_id"`
	Status      string  `json:"status"`
	Progress    float64 `json:"progress"`
	VideoURL    string  `json:"video_url,omitempty"`
	CreditsUsed float64 `json:"credits_used"`
}

type ComposeRequest struct {
	Clips []struct {
		JobID      string  `json:"job_id" validate:"required"`
		Transition string  `json:"transition,omitempty"` // e.g., "fade", "wipeleft"
		Duration   float64 `json:"duration,omitempty"`   // transition duration in seconds
	} `json:"clips" validate:"required,min=1"`
	OutputName string `json:"output_name" form:"output_name,omitempty"`

	// Note: Clips must be sent as JSON due to nested array structure
}

type VideoResponse struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Length    float64 `json:"length"`
	VideoURL  string  `json:"video_url"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
}
