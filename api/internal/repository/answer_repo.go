package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"2026champs/internal/model"
)

type AnswerRepository interface {
	Create(ctx context.Context, answer *model.Answer) error
	GetByID(ctx context.Context, id string) (*model.Answer, error)
	GetBySessionID(ctx context.Context, sessionID string) ([]*model.Answer, error)
	GetByPlayerID(ctx context.Context, playerID string) ([]*model.Answer, error)
	Update(ctx context.Context, answer *model.Answer) error
	Delete(ctx context.Context, id string) error
}

type answerRepository struct {
	collection *mongo.Collection
}

func NewAnswerRepository(client *mongo.Client) AnswerRepository {
	db := client.Database("2026champs")
	return &answerRepository{
		collection: db.Collection("answers"),
	}
}

func (r *answerRepository) Create(ctx context.Context, answer *model.Answer) error {
	// Set creation timestamp if not set
	if answer.AnsweredAt.IsZero() {
		answer.AnsweredAt = time.Now()
	}

	result, err := r.collection.InsertOne(ctx, answer)
	if err != nil {
		return err
	}

	// Set the ID from MongoDB
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		answer.ID = oid.Hex()
	}

	return nil
}

func (r *answerRepository) GetByID(ctx context.Context, id string) (*model.Answer, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var answer model.Answer
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&answer)
	if err != nil {
		return nil, err
	}

	return &answer, nil
}

func (r *answerRepository) GetBySessionID(ctx context.Context, sessionID string) ([]*model.Answer, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"sessionId": sessionID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var answers []*model.Answer
	if err = cursor.All(ctx, &answers); err != nil {
		return nil, err
	}

	return answers, nil
}

func (r *answerRepository) GetByPlayerID(ctx context.Context, playerID string) ([]*model.Answer, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"playerId": playerID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var answers []*model.Answer
	if err = cursor.All(ctx, &answers); err != nil {
		return nil, err
	}

	return answers, nil
}

func (r *answerRepository) Update(ctx context.Context, answer *model.Answer) error {
	oid, err := primitive.ObjectIDFromHex(answer.ID)
	if err != nil {
		return err
	}

	update := bson.M{"$set": answer}
	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": oid}, update)
	return err
}

func (r *answerRepository) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}
