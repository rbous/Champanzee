package repository

import (
	"2026champs/internal/model"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// AnswerRepo handles MongoDB operations for answers (historical persistence)
type AnswerRepo interface {
	Create(ctx context.Context, answer *model.Answer) (string, error)
	GetByID(ctx context.Context, id string) (*model.Answer, error)
	GetByRoomCode(ctx context.Context, roomCode string) ([]*model.Answer, error)
	GetByRoomAndPlayer(ctx context.Context, roomCode, playerID string) ([]*model.Answer, error)
	GetByRoomAndQuestion(ctx context.Context, roomCode, questionKey string) ([]*model.Answer, error)
	Update(ctx context.Context, answer *model.Answer) error
	CheckIdempotency(ctx context.Context, roomCode, playerID, questionKey, clientAttemptID string) (bool, error)
}

type answerRepo struct {
	collection *mongo.Collection
}

// NewAnswerRepo creates a new answer repository
func NewAnswerRepo(db *mongo.Database) AnswerRepo {
	return &answerRepo{
		collection: db.Collection("answers"),
	}
}

func (r *answerRepo) Create(ctx context.Context, answer *model.Answer) (string, error) {
	answer.CreatedAt = time.Now()
	answer.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, answer)
	if err != nil {
		return "", err
	}

	oid, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", nil
	}
	return oid.Hex(), nil
}

func (r *answerRepo) GetByID(ctx context.Context, id string) (*model.Answer, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var answer model.Answer
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&answer)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	answer.ID = id
	return &answer, nil
}

func (r *answerRepo) GetByRoomCode(ctx context.Context, roomCode string) ([]*model.Answer, error) {
	cursor, err := r.collection.Find(ctx, bson.M{
		"roomCode": roomCode,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	answers := []*model.Answer{}
	if err := cursor.All(ctx, &answers); err != nil {
		return nil, err
	}
	return answers, nil
}

func (r *answerRepo) GetByRoomAndPlayer(ctx context.Context, roomCode, playerID string) ([]*model.Answer, error) {
	cursor, err := r.collection.Find(ctx, bson.M{
		"roomCode": roomCode,
		"playerId": playerID,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	answers := []*model.Answer{}
	if err := cursor.All(ctx, &answers); err != nil {
		return nil, err
	}
	return answers, nil
}

func (r *answerRepo) GetByRoomAndQuestion(ctx context.Context, roomCode, questionKey string) ([]*model.Answer, error) {
	cursor, err := r.collection.Find(ctx, bson.M{
		"roomCode":    roomCode,
		"questionKey": questionKey,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	answers := []*model.Answer{}
	if err := cursor.All(ctx, &answers); err != nil {
		return nil, err
	}
	return answers, nil
}

func (r *answerRepo) Update(ctx context.Context, answer *model.Answer) error {
	oid, err := primitive.ObjectIDFromHex(answer.ID)
	if err != nil {
		return err
	}

	answer.UpdatedAt = time.Now()
	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": oid}, answer)
	return err
}

func (r *answerRepo) CheckIdempotency(ctx context.Context, roomCode, playerID, questionKey, clientAttemptID string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{
		"roomCode":        roomCode,
		"playerId":        playerID,
		"questionKey":     questionKey,
		"clientAttemptId": clientAttemptID,
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
