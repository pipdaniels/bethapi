package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	MongoURI       string
	MongoDBName    string
	RedisAddr      string
	JWTSecret      string
	ResendAPIKey   string
	GoogleAIKey    string
	GoogleProjectID string
	GoogleLocation  string
	R2AccessKey    string
	R2SecretKey    string
	R2Endpoint     string
	R2Bucket       string
	R2PublicDomain string
	AllowedOrigins []string

	// Pricing (Credits)
	PricingLLMPrompt1K  float64
	PricingLLMOutput1K  float64
	PricingVideoSec     float64
	PricingImagen       float64

	// Subscriptions
	ProPriceID   string
	UltraPriceID string

	// Renewal Policy
	GracePeriodHours int
	RenewalNotifyCount int

	// OAuth & Security
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	FrontendURL        string
	OTPExpiryMinutes   int
	CookieDomain       string
	Env                string

	// Payments (Secrets)
	StripeSecretKey     string
	StripeWebhookSecret  string
	FlutterwaveSecretKey string
	FlutterwaveWebhookSecret string
	PaystackSecretKey    string
	PaystackWebhookSecret string

	// Conversion Rates (Fixed)
	RateNGN float64
	RateKES float64
	RateGHS float64

	// Tiered Pricing
	PricePro   float64
	PriceUltra float64
	CreditsPro   float64
	CreditsUltra float64
	RatePAYG     float64

	PricingVideoSecFast float64
	PricingVideoSecStd  float64
}

var AppConfig *Config

func LoadConfig() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, using environment variables")
	}

	AppConfig = &Config{
		Port:           GetEnv("PORT", "8080"),
		MongoURI:       GetEnv("MONGO_URI", ""),
		MongoDBName:    GetEnv("MONGO_DB_NAME", "bethapi"),
		RedisAddr:      GetEnv("REDIS_ADDR", "localhost:6379"),
		JWTSecret:      GetEnv("JWT_SECRET", ""),
		ResendAPIKey:   GetEnv("RESEND_API_KEY", ""),
		GoogleAIKey:    GetEnv("GOOGLE_AI_API_KEY", ""),
		GoogleProjectID: GetEnv("GOOGLE_PROJECT_ID", ""),
		GoogleLocation:  GetEnv("GOOGLE_LOCATION", "us-central1"),
		R2AccessKey:    GetEnv("R2_ACCESS_KEY", ""),
		R2SecretKey:    GetEnv("R2_SECRET_KEY", ""),
		R2Endpoint:     GetEnv("R2_ENDPOINT", ""),
		R2Bucket:       GetEnv("R2_BUCKET", ""),
		R2PublicDomain: GetEnv("R2_PUBLIC_DOMAIN", ""),
		AllowedOrigins: strings.Fields(GetEnv("ALLOWED_ORIGINS", "")),

		PricingLLMPrompt1K:  GetEnvFloat("PRICING_LLM_PROMPT_1K", 1),
		PricingLLMOutput1K:  GetEnvFloat("PRICING_LLM_OUTPUT_1K", 2),
		PricingVideoSec:     GetEnvFloat("PRICING_VIDEO_SEC", 60),
		PricingImagen:       GetEnvFloat("PRICING_IMAGEN", 50),

		ProPriceID:   GetEnv("SUBSCRIPTION_PRO_PRICE_ID", ""),
		UltraPriceID: GetEnv("SUBSCRIPTION_ULTRA_PRICE_ID", ""),

		GracePeriodHours:   GetEnvInt("RENEWAL_GRACE_PERIOD_HOURS", 48),
		RenewalNotifyCount: GetEnvInt("RENEWAL_NOTIFY_COUNT", 4),

		GoogleClientID:     GetEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: GetEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  GetEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/google/callback"),
		FrontendURL:        GetEnv("FRONTEND_URL", "http://localhost:3000"),
		OTPExpiryMinutes:   GetEnvInt("OTP_EXPIRY_MINUTES", 5),
		CookieDomain:       GetEnv("COOKIE_DOMAIN", "localhost"),
		Env:                GetEnv("ENV", "development"),

		StripeSecretKey:     GetEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret:  GetEnv("STRIPE_WEBHOOK_SECRET", ""),
		FlutterwaveSecretKey: GetEnv("FLUTTERWAVE_SECRET_KEY", ""),
		FlutterwaveWebhookSecret: GetEnv("FLUTTERWAVE_WEBHOOK_SECRET", ""),
		PaystackSecretKey:    GetEnv("PAYSTACK_SECRET_KEY", ""),
		PaystackWebhookSecret: GetEnv("PAYSTACK_WEBHOOK_SECRET", ""),

		RateNGN: GetEnvFloat("CONVERSION_RATE_NGN", 1500),
		RateKES: GetEnvFloat("CONVERSION_RATE_KES", 130),
		RateGHS: GetEnvFloat("CONVERSION_RATE_GHS", 12),

		PricePro:     GetEnvFloat("PRO_PRICE", 29.0),
		PriceUltra:   GetEnvFloat("ULTRA_PRICE", 199.0),
		CreditsPro:   GetEnvFloat("PRO_CREDITS", 3500.0),
		CreditsUltra: GetEnvFloat("ULTRA_CREDITS", 30000.0),
		RatePAYG:     GetEnvFloat("PAYG_RATE", 0.015),

		PricingVideoSecFast: GetEnvFloat("PRICING_VIDEO_SEC_FAST", 1.0),
		PricingVideoSecStd:  GetEnvFloat("PRICING_VIDEO_SEC_STD", 0.5),
	}
}

func GetEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GetEnvFloat(key string, fallback float64) float64 {
	valStr := GetEnv(key, "")
	if valStr == "" {
		return fallback
	}
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return fallback
	}
	return val
}

func GetEnvInt(key string, fallback int) int {
	valStr := GetEnv(key, "")
	if valStr == "" {
		return fallback
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return fallback
	}
	return val
}
