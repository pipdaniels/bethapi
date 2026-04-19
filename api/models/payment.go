package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
)

type PaymentType string

const (
	PaymentTypeTopup        PaymentType = "topup"
	PaymentTypeSubscription PaymentType = "subscription"
)

type PaymentTransaction struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID        primitive.ObjectID `bson:"user_id" json:"user_id"`
	ExternalRef   string             `bson:"external_ref" json:"external_ref"` // Provider session/ref ID
	Amount        float64            `bson:"amount" json:"amount"`
	Currency      string             `bson:"currency" json:"currency"`
	Credits       float64            `bson:"credits" json:"credits"`
	Status        PaymentStatus      `bson:"status" json:"status"`
	Type          PaymentType        `bson:"type" json:"type"`
	Provider      string             `bson:"provider" json:"provider"`
	PaymentMethod string             `bson:"payment_method,omitempty" json:"payment_method,omitempty"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}
