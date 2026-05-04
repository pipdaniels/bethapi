package repository

import (
	"context"
	"time"

	"bethapi/api/database"
	"bethapi/api/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type JobRepository struct {
	collection *mongo.Collection
}

func NewJobRepository() *JobRepository {
	return &JobRepository{
		collection: database.GetCollection("jobs"),
	}
}

// Create inserts a new job record into MongoDB.
func (r *JobRepository) Create(ctx context.Context, job *models.Job) error {
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()
	_, err := r.collection.InsertOne(ctx, job)
	return err
}

// GetByID retrieves a job by its ObjectID.
func (r *JobRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Job, error) {
	var job models.Job
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&job)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// GetByStringID retrieves a job by its hex string ID.
func (r *JobRepository) GetByStringID(ctx context.Context, id string) (*models.Job, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, objID)
}

// UpdateStatus sets the job's status and optionally the progress.
func (r *JobRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.JobStatus, progress float64) error {
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"progress":   progress,
			"updated_at": time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// MarkCompleted sets the job status to completed, saves the video URL, credits used, and records the completion time.
func (r *JobRepository) MarkCompleted(ctx context.Context, id primitive.ObjectID, videoURL string, creditsUsed float64) error {
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"status":       models.JobStatusCompleted,
			"progress":     1.0,
			"video_url":    videoURL,
			"credits_used": creditsUsed,
			"completed_at": now,
			"updated_at":   now,
		},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// MarkFailed sets the job status to failed with an error message.
func (r *JobRepository) MarkFailed(ctx context.Context, id primitive.ObjectID, errorMessage string) error {
	update := bson.M{
		"$set": bson.M{
			"status":        models.JobStatusFailed,
			"error_message": errorMessage,
			"updated_at":    time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// ListByUser returns all jobs for a given user, ordered newest first.
func (r *JobRepository) ListByUser(ctx context.Context, userID primitive.ObjectID) ([]*models.Job, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var jobs []*models.Job
	if err := cursor.All(ctx, &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

// UpdateProviderJobID updates the provider's job ID (e.g., LRO name) for a job.
func (r *JobRepository) UpdateProviderJobID(ctx context.Context, id primitive.ObjectID, providerJobID string) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"provider_job_id": providerJobID,
			"updated_at":      time.Now(),
		},
	})
	return err
}

// UpdateStoryboards updates the storyboard URLs for a job.
func (r *JobRepository) UpdateStoryboards(ctx context.Context, id primitive.ObjectID, urls []string) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"storyboard_urls": urls,
			"updated_at":      time.Now(),
		},
	})
	return err
}

// FindProcessingJobs returns all jobs that are currently being processed by the AI provider.
func (r *JobRepository) FindProcessingJobs(ctx context.Context) ([]*models.Job, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"status": models.JobStatusProcessing})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var jobs []*models.Job
	if err := cursor.All(ctx, &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *JobRepository) UpdateCreditsDeducted(ctx context.Context, id primitive.ObjectID, credits float64) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"credits_deducted": credits,
			"updated_at":       time.Now(),
		},
	})
	return err
}

// ListByUserPaginated returns paginated jobs for a given user, ordered newest first
func (r *JobRepository) ListByUserPaginated(ctx context.Context, userID primitive.ObjectID, skip int64, limit int) ([]*models.Job, int64, error) {
	opts := options.Find().SetSkip(skip).SetLimit(int64(limit)).SetSort(bson.M{"created_at": -1})

	// Get total count for pagination metadata
	totalCount, err := r.collection.CountDocuments(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, 0, err
	}

	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var jobs []*models.Job
	if err := cursor.All(ctx, &jobs); err != nil {
		return nil, 0, err
	}

	return jobs, totalCount, nil
}
