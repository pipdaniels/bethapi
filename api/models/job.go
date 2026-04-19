package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

type JobType string

const (
	JobTypeGeneration  JobType = "generation"
	JobTypeComposition JobType = "composition"
)

type CompositionClipData struct {
	JobID      string  `bson:"job_id" json:"job_id"`
	Transition string  `bson:"transition,omitempty" json:"transition,omitempty"`
	Duration   float64 `bson:"duration,omitempty" json:"duration,omitempty"`
}

type Job struct {
	ID               primitive.ObjectID    `bson:"_id,omitempty" json:"id"`
	UserID           primitive.ObjectID    `bson:"user_id" json:"user_id"`
	Type             JobType               `bson:"type" json:"type"`
	GenerationMode   string                `bson:"generation_mode,omitempty" json:"generation_mode,omitempty"`
	CreditsDeducted  float64               `bson:"credits_deducted,omitempty" json:"credits_deducted,omitempty"`
	Prompt           string                `bson:"prompt,omitempty" json:"prompt,omitempty"`
	Status           JobStatus             `bson:"status" json:"status"`
	Progress         float64               `bson:"progress" json:"progress"`
	CompositionData  []CompositionClipData `bson:"composition_data,omitempty" json:"composition_data,omitempty"`
	EnhancedPrompt   string             `bson:"enhanced_prompt,omitempty" json:"enhanced_prompt,omitempty"`
	StoryboardURLs   []string           `bson:"storyboard_urls,omitempty" json:"storyboard_urls,omitempty"`
	VideoURL         string             `bson:"video_url,omitempty" json:"video_url,omitempty"`
	ProviderJobID    string             `bson:"provider_job_id,omitempty" json:"provider_job_id,omitempty"`
	CreditsUsed      float64            `bson:"credits_used" json:"credits_used"`
	AspectRatio      string             `bson:"aspect_ratio,omitempty" json:"aspect_ratio,omitempty"`
	Duration         float64            `bson:"duration,omitempty" json:"duration,omitempty"`
	ErrorMessage     string             `bson:"error_message,omitempty" json:"error_message,omitempty"`
	CreatedAt        time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time          `bson:"updated_at" json:"updated_at"`
	CompletedAt      *time.Time         `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
}
