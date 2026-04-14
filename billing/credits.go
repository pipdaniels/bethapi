package billing

import (
	"context"
	"time"

	"bethapi/api/models"
	"bethapi/api/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CreditService struct {
	userRepo  *repository.UserRepository
	transRepo *repository.TransactionRepository
}

func NewCreditService(userRepo *repository.UserRepository, transRepo *repository.TransactionRepository) *CreditService {
	return &CreditService{
		userRepo:  userRepo,
		transRepo: transRepo,
	}
}

func (s *CreditService) AddCredits(ctx context.Context, userID primitive.ObjectID, amount float64, description string) error {
	err := s.userRepo.UpdateCredits(ctx, userID, amount)
	if err != nil {
		return err
	}

	// Log transaction
	transaction := &models.Transaction{
		UserID:      userID,
		Type:        models.TypeCredit,
		Amount:      amount,
		Description: description,
		CreatedAt:   time.Now(),
	}
	return s.transRepo.Create(ctx, transaction)
}

func (s *CreditService) DeductCredits(ctx context.Context, userID primitive.ObjectID, amount float64, jobID string, description string) error {
	err := s.userRepo.DeductCredits(ctx, userID, amount)
	if err != nil {
		return err
	}

	// Log transaction
	transaction := &models.Transaction{
		UserID:      userID,
		Type:        models.TypeDebit,
		Amount:      amount,
		JobID:       jobID,
		Description: description,
		CreatedAt:   time.Now(),
	}
	return s.transRepo.Create(ctx, transaction)
}
