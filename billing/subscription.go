package billing

import (
	"context"
	"time"

	"bethapi/api/models"
	"bethapi/api/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SubscriptionService struct {
	userRepo *repository.UserRepository
}

func NewSubscriptionService(userRepo *repository.UserRepository) *SubscriptionService {
	return &SubscriptionService{userRepo: userRepo}
}

func (s *SubscriptionService) ProcessRenewal(ctx context.Context, userID string, plan models.UserPlan) error {
	// "Use it or lose it" - Reset balance to baseline
	var credits float64
	switch plan {
	case models.PlanPro:
		credits = 5000
	case models.PlanUltra:
		credits = 25000
	default:
		credits = 0
	}

	// Update user balance and renewal date (1 month from now)
	nextRenewal := time.Now().AddDate(0, 1, 0)
	
	id, _ := primitive.ObjectIDFromHex(userID)
	_ = credits // Used when resetting balance to plan baseline
	return s.userRepo.UpdateSubscription(ctx, id, plan, models.StatusActive, &nextRenewal)
	// TODO: Additionally reset credit_balance to 'credits' baseline here
}

func (s *SubscriptionService) HandleFailedPayment(ctx context.Context, userID string) error {
	// Set status to past_due and record start of grace period
	now := time.Now()
	id, _ := primitive.ObjectIDFromHex(userID)
	return s.userRepo.UpdateSubscription(ctx, id, "", models.StatusPastDue, &now)
	// now should be used for grace_period_started
}

func (s *SubscriptionService) DowngradeToFree(ctx context.Context, userID string) error {
	// Reset to free plan
	id, _ := primitive.ObjectIDFromHex(userID)
	return s.userRepo.UpdateSubscription(ctx, id, models.PlanFree, models.StatusActive, nil)
}
