package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bethapi/config"

	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/paymentintent"
	"github.com/stripe/stripe-go/v81/webhook"
)

type CheckoutRequest struct {
	UserID         string
	Email          string
	Amount         float64
	Currency       string
	IsSubscription bool
	Credits        float64
}

type WebhookEvent struct {
	ExternalRef   string
	Amount        float64
	Currency      string
	Status        string
	PaymentMethod string
	UserID        string
	Credits       float64
	Type          string // "topup" or "subscription"
}

type PaymentProvider interface {
	Name() string
	CreateCheckoutSession(ctx context.Context, req CheckoutRequest) (string, error)
	VerifyWebhook(c echo.Context) (*WebhookEvent, error)
	ChargeSavedCard(ctx context.Context, token string, amount float64, currency string) (string, error)
}

// Stripe Implementation
type StripeProvider struct {
	secretKey     string
	webhookSecret string
}

func NewStripeProvider() *StripeProvider {
	stripe.Key = config.AppConfig.StripeSecretKey
	return &StripeProvider{
		secretKey:     config.AppConfig.StripeSecretKey,
		webhookSecret: config.AppConfig.StripeWebhookSecret,
	}
}

func (p *StripeProvider) Name() string { return "stripe" }

func (p *StripeProvider) CreateCheckoutSession(ctx context.Context, req CheckoutRequest) (string, error) {
	// Map credits/type to metadata for webhook retrieval
	params := &stripe.CheckoutSessionParams{
		SuccessURL:         stripe.String(config.AppConfig.FrontendURL + "/payment/success"),
		CancelURL:          stripe.String(config.AppConfig.FrontendURL + "/payment/cancel"),
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(strings.ToLower(req.Currency)),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(fmt.Sprintf("Beth AI Credits - %.0f", req.Credits)),
					},
					UnitAmount: stripe.Int64(int64(req.Amount * 100)),
				},
				Quantity: stripe.Int64(1),
			},
		},
		CustomerEmail: stripe.String(req.Email),
		Metadata: map[string]string{
			"user_id": req.UserID,
			"credits": fmt.Sprintf("%.2f", req.Credits),
			"type":    "topup",
		},
	}

	if req.IsSubscription {
		params.Metadata["type"] = "subscription"
		// To save card for future use (auto-charge), we use payment_intent_data
		params.PaymentIntentData = &stripe.CheckoutSessionPaymentIntentDataParams{
			SetupFutureUsage: stripe.String(string(stripe.PaymentIntentSetupFutureUsageOffSession)),
		}
	}

	s, err := session.New(params)
	if err != nil {
		return "", err
	}
	return s.URL, nil
}

func (p *StripeProvider) VerifyWebhook(c echo.Context) (*WebhookEvent, error) {
	const MaxBodyBytes = int64(65536)
	payload, err := io.ReadAll(io.LimitReader(c.Request().Body, MaxBodyBytes))
	if err != nil {
		return nil, err
	}

	sig := c.Request().Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, sig, p.webhookSecret)
	if err != nil {
		return nil, err
	}

	if event.Type != "checkout.session.completed" {
		return nil, fmt.Errorf("ignoring event type: %s", event.Type)
	}

	var sess stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
		return nil, err
	}

	credits, _ := strconv.ParseFloat(sess.Metadata["credits"], 64)

	return &WebhookEvent{
		ExternalRef:   sess.ID,
		Amount:        float64(sess.AmountTotal) / 100,
		Currency:      string(sess.Currency),
		Status:        "successful",
		PaymentMethod: sess.PaymentIntent.ID, // Or SetupIntent
		UserID:        sess.Metadata["user_id"],
		Credits:       credits,
		Type:          sess.Metadata["type"],
	}, nil
}

func (p *StripeProvider) ChargeSavedCard(ctx context.Context, token string, amount float64, currency string) (string, error) {
	params := &stripe.PaymentIntentParams{
		Amount:        stripe.Int64(int64(amount * 100)),
		Currency:      stripe.String(strings.ToLower(currency)),
		PaymentMethod: stripe.String(token),
		Confirm:       stripe.Bool(true),
		OffSession:    stripe.Bool(true),
	}
	pi, err := paymentintent.New(params)
	if err != nil {
		return "", err
	}
	return pi.ID, nil
}

// Flutterwave Implementation
type FlutterwaveProvider struct {
	secretKey     string
	webhookSecret string
}

func NewFlutterwaveProvider() *FlutterwaveProvider {
	return &FlutterwaveProvider{
		secretKey:     config.AppConfig.FlutterwaveSecretKey,
		webhookSecret: config.AppConfig.FlutterwaveWebhookSecret,
	}
}

func (p *FlutterwaveProvider) Name() string { return "flutterwave" }

func (p *FlutterwaveProvider) CreateCheckoutSession(ctx context.Context, req CheckoutRequest) (string, error) {
	// Standard Flutterwave Checkouts use a structured JSON payload
	url := "https://api.flutterwave.com/v3/payments"
	payload := map[string]interface{}{
		"tx_ref":          fmt.Sprintf("beth_%s_%d", req.UserID, time.Now().Unix()),
		"amount":          fmt.Sprintf("%.2f", req.Amount),
		"currency":        req.Currency,
		"redirect_url":    config.AppConfig.FrontendURL + "/payment/flutterwave/callback",
		"payment_options": "card,ussd",
		"customer": map[string]string{
			"email": req.Email,
		},
		"meta": map[string]interface{}{
			"user_id": req.UserID,
			"credits": req.Credits,
			"type":    "topup",
		},
	}
	if req.IsSubscription {
		payload["meta"].(map[string]interface{})["type"] = "subscription"
	}

	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	httpReq.Header.Set("Authorization", "Bearer "+p.secretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Link string `json:"link"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Status != "success" {
		return "", fmt.Errorf("flutterwave error: %s", result.Message)
	}

	return result.Data.Link, nil
}

func (p *FlutterwaveProvider) VerifyWebhook(c echo.Context) (*WebhookEvent, error) {
	// Verification logic for Flutterwave signature
	// usually checked via 'verif-hash' header
	return nil, fmt.Errorf("not implemented")
}

func (p *FlutterwaveProvider) ChargeSavedCard(ctx context.Context, token string, amount float64, currency string) (string, error) {
	// Flutterwave recurring charge via token
	url := "https://api.flutterwave.com/v3/tokenized-charges"
	payload := map[string]interface{}{
		"token":    token,
		"currency": currency,
		"amount":   amount,
		"tx_ref":   fmt.Sprintf("renew_%s_%d", token[:8], time.Now().Unix()),
	}
	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	httpReq.Header.Set("Authorization", "Bearer "+p.secretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("flutterwave charge failed: status %d", resp.StatusCode)
	}
	return "successful", nil
}

// Paystack Implementation
type PaystackProvider struct {
	secretKey     string
	webhookSecret string
}

func NewPaystackProvider() *PaystackProvider {
	return &PaystackProvider{
		secretKey:     config.AppConfig.PaystackSecretKey,
		webhookSecret: config.AppConfig.PaystackWebhookSecret,
	}
}

func (p *PaystackProvider) Name() string { return "paystack" }

func (p *PaystackProvider) CreateCheckoutSession(ctx context.Context, req CheckoutRequest) (string, error) {
	url := "https://api.paystack.co/transaction/initialize"
	payload := map[string]interface{}{
		"amount": int64(req.Amount * 100), // Kobo/Cents
		"email":  req.Email,
		"callback_url": config.AppConfig.FrontendURL + "/payment/paystack/callback",
		"metadata": map[string]interface{}{
			"user_id": req.UserID,
			"credits": req.Credits,
			"type":    "topup",
		},
	}
	if req.IsSubscription {
		payload["metadata"].(map[string]interface{})["type"] = "subscription"
	}

	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	httpReq.Header.Set("Authorization", "Bearer "+p.secretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			AuthorizationURL string `json:"authorization_url"`
			Reference        string `json:"reference"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if !result.Status {
		return "", fmt.Errorf("paystack error: %s", result.Message)
	}

	return result.Data.AuthorizationURL, nil
}

func (p *PaystackProvider) VerifyWebhook(c echo.Context) (*WebhookEvent, error) {
	// Paystack signature verification uses x-paystack-signature
	return nil, fmt.Errorf("not implemented")
}

func (p *PaystackProvider) ChargeSavedCard(ctx context.Context, token string, amount float64, currency string) (string, error) {
	url := "https://api.paystack.co/transaction/charge_authorization"
	payload := map[string]interface{}{
		"authorization_code": token,
		"email":             "user@example.com", // Should be retrieved from user profile
		"amount":            int64(amount * 100),
	}
	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	httpReq.Header.Set("Authorization", "Bearer "+p.secretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("paystack charge failed: status %d", resp.StatusCode)
	}
	return "successful", nil
}
