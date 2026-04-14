package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserPlan string

const (
	PlanFree  UserPlan = "free"
	PlanPro   UserPlan = "pro"
	PlanUltra UserPlan = "ultra"
)

type SubscriptionStatus string

const (
	StatusActive   SubscriptionStatus = "active"
	StatusPastDue  SubscriptionStatus = "past_due"
	StatusCanceled SubscriptionStatus = "canceled"
)

type User struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email              string             `bson:"email" json:"email"`
	Password           string             `bson:"password,omitempty" json:"-"`
	Name               string             `bson:"name" json:"name"`
	CreditBalance      float64            `bson:"credit_balance" json:"credit_balance"`
	TotalCreditsUsed  float64            `bson:"total_credits_used" json:"total_credits_used"`
	Plan               UserPlan           `bson:"plan" json:"plan"`
	SubscriptionStatus SubscriptionStatus `bson:"subscription_status" json:"subscription_status"`
	RenewsAt           *time.Time         `bson:"renews_at,omitempty" json:"renews_at,omitempty"`
	GracePeriodStarted *time.Time         `bson:"grace_period_started,omitempty" json:"grace_period_started,omitempty"`
	NotificationCount  int                `bson:"notification_count" json:"notification_count"`
	APIKey             string             `bson:"api_key,omitempty" json:"api_key,omitempty"`
	CreatedAt          time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt          time.Time          `bson:"updated_at" json:"updated_at"`
}

type TransactionType string

const (
	TypeCredit TransactionType = "credit"
	TypeDebit  TransactionType = "debit"
)

type Transaction struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID `bson:"user_id" json:"user_id"`
	Type        TransactionType    `bson:"type" json:"type"`
	Amount      float64            `bson:"amount" json:"amount"`
	TokensUsed  int64              `bson:"tokens_used,omitempty" json:"tokens_used,omitempty"`
	JobID       string             `bson:"job_id,omitempty" json:"job_id,omitempty"`
	Description string             `bson:"description" json:"description"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}
