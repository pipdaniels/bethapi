package repository

import (
	"context"
	"errors"
	"time"

	"bethapi/api/database"
	"bethapi/api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepository struct {
	collection *mongo.Collection
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		collection: database.GetCollection("users"),
	}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	if user.Plan == "" {
		user.Plan = models.PlanFree
	}
	if user.SubscriptionStatus == "" {
		user.SubscriptionStatus = models.StatusActive
	}
	
	res, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		return err
	}
	user.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) UpdateCredits(ctx context.Context, userID primitive.ObjectID, amount float64) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{
		"$inc": bson.M{"credit_balance": amount},
		"$set": bson.M{"updated_at": time.Now()},
	})
	return err
}

func (r *UserRepository) DeductCredits(ctx context.Context, userID primitive.ObjectID, amount float64) error {
	// Atomic check and deduct
	res, err := r.collection.UpdateOne(ctx, 
		bson.M{"_id": userID, "credit_balance": bson.M{"$gte": amount}}, 
		bson.M{
			"$inc": bson.M{"credit_balance": -amount, "total_credits_used": amount},
			"$set": bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		return err
	}
	if res.ModifiedCount == 0 {
		return errors.New("insufficient credits")
	}
	return nil
}

func (r *UserRepository) UpdateSubscription(ctx context.Context, userID primitive.ObjectID, plan models.UserPlan, status models.SubscriptionStatus, renewsAt *time.Time) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{
		"$set": bson.M{
			"plan":                plan,
			"subscription_status": status,
			"renews_at":           renewsAt,
			"updated_at":          time.Now(),
		},
	})
	return err
}
