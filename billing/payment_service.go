package billing

import (
	"fmt"
	"log"

	"bethapi/api/repository"
	"bethapi/config"

	"github.com/labstack/echo/v4"
)

type PaymentService struct {
	userRepo      *repository.UserRepository
	creditService *CreditService
	providers     map[string]PaymentProvider
}

func NewPaymentService(userRepo *repository.UserRepository, creditService *CreditService) *PaymentService {
	providers := make(map[string]PaymentProvider)

	// Register providers if keys are available
	if config.AppConfig.StripeSecretKey != "" {
		providers["stripe"] = NewStripeProvider()
	}
	if config.AppConfig.FlutterwaveSecretKey != "" {
		providers["flutterwave"] = NewFlutterwaveProvider()
	}
	if config.AppConfig.PaystackSecretKey != "" {
		providers["paystack"] = NewPaystackProvider()
	}

	return &PaymentService{
		userRepo:      userRepo,
		creditService: creditService,
		providers:     providers,
	}
}

// GetActiveProvider returns the primary provider based on availability.
func (s *PaymentService) GetActiveProvider() (PaymentProvider, error) {
	// Priority: Stripe > Flutterwave > Paystack
	if p, ok := s.providers["stripe"]; ok {
		return p, nil
	}
	if p, ok := s.providers["flutterwave"]; ok {
		return p, nil
	}
	if p, ok := s.providers["paystack"]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("no payment provider configured")
}

func (s *PaymentService) GetProvider(name string) (PaymentProvider, bool) {
	p, ok := s.providers[name]
	return p, ok
}

func (s *PaymentService) ProcessWebhook(c echo.Context, providerName string) error {
	p, ok := s.providers[providerName]
	if !ok {
		return fmt.Errorf("unknown provider: %s", providerName)
	}

	event, err := p.VerifyWebhook(c)
	if err != nil {
		return fmt.Errorf("webhook verification failed: %w", err)
	}

	// Double check if status is successful (VerifyWebhook should handle this but extra safety)
	if event.Status != "successful" {
		log.Printf("Ignoring unsuccessful payment event: %s", event.Status)
		return nil
	}

	ctx := c.Request().Context()

	// Fulfill Credits
	uid, _ := s.userRepo.GetByStringID(ctx, event.UserID) // Should use GetByStringID or similar
	if uid == nil {
		return fmt.Errorf("user not found for payment: %s", event.UserID)
	}

	description := fmt.Sprintf("Topup via %s (Ref: %s)", providerName, event.ExternalRef)
	if event.Type == "subscription" {
		description = fmt.Sprintf("Subscription Renewal: %s", event.ExternalRef)

		// If it's a subscription, also save the payment token for auto-charge
		if event.PaymentMethod != "" {
			err := s.userRepo.SavePaymentMethod(ctx, uid.ID, event.PaymentMethod, providerName)
			if err != nil {
				log.Printf("Failed to save payment method for user %s: %v", event.UserID, err)
			}
		}
	}

	return s.creditService.AddCredits(ctx, uid.ID, event.Credits, description)
}

// GetConversionRate returns the fixed rate from config for a given currency.
func (s *PaymentService) GetConversionRate(currency string) float64 {
	switch currency {
	case "NGN":
		return config.AppConfig.RateNGN
	case "KES":
		return config.AppConfig.RateKES
	case "GHS":
		return config.AppConfig.RateGHS
	default:
		return 1.0 // USD or other base
	}
}
