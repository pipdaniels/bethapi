package config

import (
	"log"
	"os"
	"strconv"

	"github.com/spf13/viper"
)

type Config struct {
	Port           string `mapstructure:"PORT"`
	MongoURI       string `mapstructure:"MONGO_URI"`
	MongoDBName    string `mapstructure:"MONGO_DB_NAME"`
	RedisAddr      string `mapstructure:"REDIS_ADDR"`
	JWTSecret      string `mapstructure:"JWT_SECRET"`
	ResendAPIKey   string `mapstructure:"RESEND_API_KEY"`
	GoogleAIKey    string `mapstructure:"GOOGLE_AI_API_KEY"`
	R2AccessKey    string `mapstructure:"R2_ACCESS_KEY"`
	R2SecretKey    string `mapstructure:"R2_SECRET_KEY"`
	R2Endpoint     string `mapstructure:"R2_ENDPOINT"`
	R2Bucket       string `mapstructure:"R2_BUCKET"`
	R2PublicDomain string `mapstructure:"R2_PUBLIC_DOMAIN"`
	AllowedOrigins []string `mapstructure:"ALLOWED_ORIGINS"`

	// Pricing (Credits)
	PricingLLMPrompt1K  float64 `mapstructure:"PRICING_LLM_PROMPT_1K"`
	PricingLLMOutput1K  float64 `mapstructure:"PRICING_LLM_OUTPUT_1K"`
	PricingVideoSec     float64 `mapstructure:"PRICING_VIDEO_SEC"`
	PricingImagen       float64 `mapstructure:"PRICING_IMAGEN"`

	// Subscriptions
	ProPriceID   string `mapstructure:"SUBSCRIPTION_PRO_PRICE_ID"`
	UltraPriceID string `mapstructure:"SUBSCRIPTION_ULTRA_PRICE_ID"`

	// Renewal Policy
	GracePeriodHours int `mapstructure:"RENEWAL_GRACE_PERIOD_HOURS"`
	RenewalNotifyCount int `mapstructure:"RENEWAL_NOTIFY_COUNT"`
}

var AppConfig *Config

func LoadConfig() {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("No .env file found, using environment variables")
	}

	AppConfig = &Config{}
	if err := viper.Unmarshal(AppConfig); err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}

	// Set defaults if missing
	if AppConfig.Port == "" {
		AppConfig.Port = "8080"
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
