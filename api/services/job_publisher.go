package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"bethapi/api/database"
	"bethapi/api/models"
	"github.com/redis/go-redis/v9"
)

// JobChannelName returns the Redis pub/sub channel name for a given job.
func JobChannelName(jobID string) string {
	return fmt.Sprintf("job:%s", jobID)
}

// PublishJobUpdate publishes the full job state to its Redis channel.
// The worker calls this after every status transition.
func PublishJobUpdate(ctx context.Context, job *models.Job) {
	data, err := json.Marshal(job)
	if err != nil {
		log.Printf("PublishJobUpdate: marshal error for job %s: %v", job.ID.Hex(), err)
		return
	}
	channel := JobChannelName(job.ID.Hex())
	if err := database.RedisClient.Publish(ctx, channel, string(data)).Err(); err != nil {
		log.Printf("PublishJobUpdate: publish error for job %s: %v", job.ID.Hex(), err)
	}
}

// SubscribeJobUpdates returns a Redis PubSub subscription for the given job ID.
// The caller is responsible for calling sub.Close() when done.
func SubscribeJobUpdates(ctx context.Context, jobID string) *redis.PubSub {
	channel := JobChannelName(jobID)
	return database.RedisClient.Subscribe(ctx, channel)
}
