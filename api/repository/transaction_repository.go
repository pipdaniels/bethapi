package repository

import (
	"context"
	"time"

	"bethapi/api/database"
	"bethapi/api/models"
	"go.mongodb.org/mongo-driver/mongo"
)

type TransactionRepository struct {
	collection *mongo.Collection
}

func NewTransactionRepository() *TransactionRepository {
	return &TransactionRepository{
		collection: database.GetCollection("transactions"),
	}
}

func (r *TransactionRepository) Create(ctx context.Context, transaction *models.Transaction) error {
	if transaction.CreatedAt.IsZero() {
		transaction.CreatedAt = time.Now()
	}
	_, err := r.collection.InsertOne(ctx, transaction)
	return err
}
