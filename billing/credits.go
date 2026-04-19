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
	jobRepo   *repository.JobRepository
}

func NewCreditService(userRepo *repository.UserRepository, transRepo *repository.TransactionRepository, jobRepo *repository.JobRepository) *CreditService {
	return &CreditService{
		userRepo:  userRepo,
		transRepo: transRepo,
		jobRepo:   jobRepo,
	}
}

func (s *CreditService) RefundCredits(ctx context.Context, jobID primitive.ObjectID) error {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return err
	}

	if job.CreditsDeducted <= 0 {
		return nil // Nothing to refund
	}

	description := "Refund for failed job: " + job.ID.Hex()
	return s.AddCredits(ctx, job.UserID, job.CreditsDeducted, description)
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
