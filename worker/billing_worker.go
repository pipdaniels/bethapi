package worker

import (
	"context"
	"log"
	"time"

	"bethapi/api/models"
	"bethapi/api/repository"
	"bethapi/api/services"
	"bethapi/billing"
	"bethapi/config"
)

// BillingWorker handles subscription renewals and the 48-hour grace period
// notification chain. It is intended to run on a regular schedule (e.g. every hour).
type BillingWorker struct {
	userRepo       *repository.UserRepository
	email          *services.EmailService
	subService     *billing.SubscriptionService
	paymentService *billing.PaymentService
}

func NewBillingWorker(userRepo *repository.UserRepository, email *services.EmailService, paymentService *billing.PaymentService) *BillingWorker {
	return &BillingWorker{
		userRepo:       userRepo,
		email:          email,
		subService:     billing.NewSubscriptionService(userRepo),
		paymentService: paymentService,
	}
}

// CheckRenewals finds every active paid subscription whose renewal date has passed
// and triggers a renewal attempt. In production this would call the payment processor;
// here it calls ProcessRenewal directly (webhook-based flow can replace this).
func (w *BillingWorker) CheckRenewals(ctx context.Context) {
	users, err := w.userRepo.FindExpiringSubscriptions(ctx)
	if err != nil {
		log.Printf("BillingWorker.CheckRenewals: query error: %v", err)
		return
	}

	for _, user := range users {
		log.Printf("BillingWorker: attempting renewal for %s (plan=%s)", user.Email, user.Plan)

		if user.PaymentToken == "" {
			log.Printf("BillingWorker: No payment token for %s — starting grace period", user.Email)
			w.subService.HandleFailedPayment(ctx, user.ID.Hex())
			continue
		}

		// Calculate amount based on plan
		var amount float64
		if user.Plan == models.PlanPro {
			amount = config.AppConfig.PricePro
		} else {
			amount = config.AppConfig.PriceUltra
		}

		// Attempt auto-charge
		provider, ok := w.paymentService.GetProvider(user.PaymentProvider)
		if !ok {
			log.Printf("BillingWorker: Unknown provider %s for user %s", user.PaymentProvider, user.Email)
			w.subService.HandleFailedPayment(ctx, user.ID.Hex())
			continue
		}

		// Subscriptions are usually in USD base, but we could convert if needed.
		// For simplicity, we auto-charge in USD if it's the base.
		_, err := provider.ChargeSavedCard(ctx, user.PaymentToken, amount, "USD")
		if err != nil {
			log.Printf("BillingWorker: Charge failed for %s: %v — starting grace period", user.Email, err)
			w.subService.HandleFailedPayment(ctx, user.ID.Hex())
			continue
		}

		// On success, reset credits and extend date
		if err := w.subService.ProcessRenewal(ctx, user.ID.Hex(), user.Plan); err != nil {
			log.Printf("BillingWorker: could not finalize renewal for %s: %v", user.Email, err)
		} else {
			log.Printf("BillingWorker: auto-renewal succeeded for %s", user.Email)
		}
	}
}

// HandleGracePeriods processes every past_due user and either:
//   - Sends the next warning email (every 12 h, up to 4 times), or
//   - Downgrades to the free plan once the 48-hour grace window has expired.
func (w *BillingWorker) HandleGracePeriods(ctx context.Context) {
	gracePeriod := time.Duration(config.AppConfig.GracePeriodHours) * time.Hour
	maxNotifications := config.AppConfig.RenewalNotifyCount

	users, err := w.userRepo.FindPastDueUsers(ctx)
	if err != nil {
		log.Printf("BillingWorker.HandleGracePeriods: query error: %v", err)
		return
	}

	now := time.Now()

	for _, user := range users {
		if user.GracePeriodStarted == nil {
			// Grace period was never recorded — set it now and send first notification.
			if err := w.userRepo.SetGracePeriodStart(ctx, user.ID, now); err != nil {
				log.Printf("BillingWorker: SetGracePeriodStart error for %s: %v", user.Email, err)
			}
			w.sendNotification(ctx, user, int(gracePeriod.Hours()))
			continue
		}

		elapsed := now.Sub(*user.GracePeriodStarted)

		if elapsed >= gracePeriod {
			// ─── Grace period expired → downgrade ───────────────────────────────
			log.Printf("BillingWorker: grace period expired for %s — downgrading to free", user.Email)
			if err := w.subService.DowngradeToFree(ctx, user.ID.Hex()); err != nil {
				log.Printf("BillingWorker: DowngradeToFree error for %s: %v", user.Email, err)
			}
		} else {
			// ─── Still within grace period → fire next notification if due ───────
			// Notifications are evenly spaced: every (gracePeriod / maxNotifications).
			notifyInterval := gracePeriod / time.Duration(maxNotifications)
			expectedNotifications := int(elapsed/notifyInterval) + 1

			if expectedNotifications > user.NotificationCount && user.NotificationCount < maxNotifications {
				hoursLeft := int((gracePeriod - elapsed).Hours())
				w.sendNotification(ctx, user, hoursLeft)
			}
		}
	}
}

// sendNotification fires a grace period warning email and increments the counter in DB.
func (w *BillingWorker) sendNotification(ctx context.Context, user models.User, hoursLeft int) {
	attempt := user.NotificationCount + 1
	log.Printf("BillingWorker: sending grace warning #%d to %s (%d h left)", attempt, user.Email, hoursLeft)

	if err := w.email.SendGracePeriodWarning(user.Email, hoursLeft, attempt); err != nil {
		log.Printf("BillingWorker: email send error for %s: %v", user.Email, err)
		return
	}

	if err := w.userRepo.IncrementNotificationCount(ctx, user.ID); err != nil {
		log.Printf("BillingWorker: IncrementNotificationCount error for %s: %v", user.Email, err)
	}
}
