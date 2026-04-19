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

// creditBaseline returns the monthly credit allowance for a given plan.
func creditBaseline(plan models.UserPlan) float64 {
	switch plan {
	case models.PlanPro:
		return 5000
	case models.PlanUltra:
		return 25000
	default:
		return 0
	}
}

// ProcessRenewal resets the user's credit balance to the plan baseline (use-it-or-lose-it)
// and schedules the next renewal date one month from now.
func (s *SubscriptionService) ProcessRenewal(ctx context.Context, userID string, plan models.UserPlan) error {
	id, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	nextRenewal := time.Now().AddDate(0, 1, 0)

	// 1. Set plan + status + next renewal date
	if err := s.userRepo.UpdateSubscription(ctx, id, plan, models.StatusActive, &nextRenewal); err != nil {
		return err
	}

	// 2. Use-it-or-lose-it: reset credit_balance to plan baseline
	return s.userRepo.ResetCreditBalance(ctx, id, creditBaseline(plan))
}

// HandleFailedPayment marks a user as past_due and starts the 48-hour grace period clock.
func (s *SubscriptionService) HandleFailedPayment(ctx context.Context, userID string) error {
	id, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	now := time.Now()

	// Set subscription_status = past_due
	if err := s.userRepo.UpdateSubscription(ctx, id, "", models.StatusPastDue, nil); err != nil {
		return err
	}

	// Record when grace period started and reset notification counter
	return s.userRepo.SetGracePeriodStart(ctx, id, now)
}

// DowngradeToFree resets the user to the free plan after grace period expiry.
func (s *SubscriptionService) DowngradeToFree(ctx context.Context, userID string) error {
	id, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}
	return s.userRepo.UpdateSubscription(ctx, id, models.PlanFree, models.StatusActive, nil)
}
