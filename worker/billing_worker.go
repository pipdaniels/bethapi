package worker

import (
	"context"
	"log"
	"time"

	"bethapi/api/database"
	"bethapi/api/models"
	"bethapi/api/repository"
	"bethapi/api/services"

	"go.mongodb.org/mongo-driver/bson"
)

type BillingWorker struct {
	userRepo *repository.UserRepository
	email    *services.EmailService
}

func NewBillingWorker(userRepo *repository.UserRepository, email *services.EmailService) *BillingWorker {
	return &BillingWorker{
		userRepo: userRepo,
		email:    email,
	}
}

func (w *BillingWorker) CheckRenewals(ctx context.Context) {
	// 1. Find past_due subscriptions or those expiring now
	now := time.Now()
	cursor, err := database.GetCollection("users").Find(ctx, bson.M{
		"subscription_status": models.StatusActive,
		"renews_at":           bson.M{"$lte": now},
	})
	if err != nil {
		log.Printf("Error finding expiring subscriptions: %v", err)
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			continue
		}

		// Trigger renewal attempt (In reality, notify payment processor or wait for webhook)
		log.Printf("Attempting renewal for user %s", user.Email)
	}
}

func (w *BillingWorker) HandleGracePeriods(ctx context.Context) {
	now := time.Now()
	// Find users who are past_due and within 48h grace
	cursor, err := database.GetCollection("users").Find(ctx, bson.M{
		"subscription_status": models.StatusPastDue,
	})
	if err != nil {
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			continue
		}

		if user.GracePeriodStarted == nil {
			continue
		}

		elapsed := now.Sub(*user.GracePeriodStarted)

		if elapsed >= 48*time.Hour {
			// Downgrade
			log.Printf("Grace period expired for %s. Downgrading...", user.Email)
			// r.DowngradeToFree(...)
		} else {
			// Check if we should send next notification (every 12h)
			expectedNotifications := int(elapsed.Hours()/12) + 1
			if expectedNotifications > user.NotificationCount && user.NotificationCount < 4 {
				hoursLeft := 48 - int(elapsed.Hours())
				w.email.SendGracePeriodWarning(user.Email, hoursLeft, user.NotificationCount+1)
				// Update notification count in DB
			}
		}
	}
}
