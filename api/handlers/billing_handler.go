package handlers

import (
	"log"
	"net/http"

	"bethapi/api/dto"
	"bethapi/api/middleware"
	"bethapi/api/models"
	"bethapi/api/repository"
	"bethapi/billing"
	"bethapi/config"
	"github.com/labstack/echo/v4"
)

type BillingHandler struct {
	paymentService *billing.PaymentService
	userRepo       *repository.UserRepository
}

func NewBillingHandler(paymentService *billing.PaymentService, userRepo *repository.UserRepository) *BillingHandler {
	return &BillingHandler{
		paymentService: paymentService,
		userRepo:       userRepo,
	}
}

func (h *BillingHandler) CreateTopupLink(c echo.Context) error {
	var req dto.TopupRequest
	if err := middleware.BindAndValidate(c, &req); err != nil {
		return err
	}

	user := c.Get("user").(models.User)
	provider, err := h.paymentService.GetActiveProvider()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{Message: err.Error()})
	}

	// Calculate credits based on amount and PAYG rate ($1 = 1/0.015 credits)
	// We assume input amount is in the chosen currency
	rate := h.paymentService.GetConversionRate(req.Currency)
	usdAmount := req.Amount / rate
	credits := usdAmount / config.AppConfig.RatePAYG

	checkoutReq := billing.CheckoutRequest{
		UserID:         user.ID.Hex(),
		Email:          user.Email,
		Amount:         req.Amount,
		Currency:       req.Currency,
		IsSubscription: false,
		Credits:        credits,
	}

	url, err := provider.CreateCheckoutSession(c.Request().Context(), checkoutReq)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: "Failed to create payment session"})
	}

	return c.JSON(http.StatusOK, dto.CheckoutResponse{URL: url})
}

func (h *BillingHandler) Subscribe(c echo.Context) error {
	var req dto.SubscribeRequest
	if err := middleware.BindAndValidate(c, &req); err != nil {
		return err
	}

	user := c.Get("user").(models.User)
	provider, err := h.paymentService.GetActiveProvider()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{Message: err.Error()})
	}

	// Base prices in USD
	var amount float64
	var credits float64
	if req.Plan == "pro" {
		amount = config.AppConfig.PricePro
		credits = config.AppConfig.CreditsPro
	} else {
		amount = config.AppConfig.PriceUltra
		credits = config.AppConfig.CreditsUltra
	}

	// Convert to desired currency
	rate := h.paymentService.GetConversionRate(req.Currency)
	convertedAmount := amount * rate

	checkoutReq := billing.CheckoutRequest{
		UserID:         user.ID.Hex(),
		Email:          user.Email,
		Amount:         convertedAmount,
		Currency:       req.Currency,
		IsSubscription: true,
		Credits:        credits,
	}

	url, err := provider.CreateCheckoutSession(c.Request().Context(), checkoutReq)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: "Failed to create subscription session"})
	}

	return c.JSON(http.StatusOK, dto.CheckoutResponse{URL: url})
}

func (h *BillingHandler) HandleWebhook(c echo.Context) error {
	provider := c.Param("provider")
	
	err := h.paymentService.ProcessWebhook(c, provider)
	if err != nil {
		log.Printf("Webhook error (%s): %v", provider, err)
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "Webhook verification failed"})
	}

	return c.NoContent(http.StatusOK)
}
