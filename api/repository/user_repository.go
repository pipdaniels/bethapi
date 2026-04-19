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

// ResetCreditBalance sets the user's credit_balance to the exact baseline for their plan
// and clears notification_count. Used on successful renewal (use-it-or-lose-it policy).
func (r *UserRepository) ResetCreditBalance(ctx context.Context, userID primitive.ObjectID, credits float64) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{
		"$set": bson.M{
			"credit_balance":     credits,
			"notification_count": 0,
			"updated_at":         time.Now(),
		},
	})
	return err
}

// SetGracePeriodStart records when the grace period began and resets the notification counter.
func (r *UserRepository) SetGracePeriodStart(ctx context.Context, userID primitive.ObjectID, at time.Time) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{
		"$set": bson.M{
			"grace_period_started": at,
			"notification_count":   0,
			"updated_at":           time.Now(),
		},
	})
	return err
}

// IncrementNotificationCount atomically increments the notification counter.
func (r *UserRepository) IncrementNotificationCount(ctx context.Context, userID primitive.ObjectID) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{
		"$inc": bson.M{"notification_count": 1},
		"$set": bson.M{"updated_at": time.Now()},
	})
	return err
}

// FindPastDueUsers returns all users currently in the past_due state.
func (r *UserRepository) FindPastDueUsers(ctx context.Context) ([]models.User, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"subscription_status": models.StatusPastDue})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// FindExpiringSubscriptions returns active subscriptions whose renewal date has passed.
func (r *UserRepository) FindExpiringSubscriptions(ctx context.Context) ([]models.User, error) {
	cursor, err := r.collection.Find(ctx, bson.M{
		"subscription_status": models.StatusActive,
		"plan":                bson.M{"$in": []models.UserPlan{models.PlanPro, models.PlanUltra}},
		"renews_at":           bson.M{"$lte": time.Now()},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) SavePaymentMethod(ctx context.Context, userID primitive.ObjectID, token, provider string) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{
		"$set": bson.M{
			"payment_token":    token,
			"payment_provider": provider,
			"updated_at":       time.Now(),
		},
	})
	return err
}

func (r *UserRepository) GetByStringID(ctx context.Context, id string) (*models.User, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, objID)
}
