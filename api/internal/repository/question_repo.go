package repository

import (
	"2026champs/internal/model"
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type QuestionRepo interface {
	// Basic CRUD Operations
	Create(ctx context.Context, question *model.Question) error
	GetByID(ctx context.Context, id string) (*model.Question, error)
	Update(ctx context.Context, question *model.Question) error
	Delete(ctx context.Context, id string) error

	// AI-Driven Selection (Core Intelligence)
	GetByCategory(ctx context.Context, category string) ([]*model.Question, error)
	GetByPriority(ctx context.Context, minPriority, maxPriority int) ([]*model.Question, error)
	GetByType(ctx context.Context, questionType model.QuestionType) ([]*model.Question, error)

	// Analysis Support Methods
	GetByIDs(ctx context.Context, ids []string) ([]*model.Question, error)

	// Management Methods
	GetAll(ctx context.Context) ([]*model.Question, error)
	GetActive(ctx context.Context) ([]*model.Question, error)
}

type questionRepo struct {
	collection *mongo.Collection
}

func NewQuestionRepo(client *mongo.Client) QuestionRepo {
	db := client.Database("2026champs")
	return &questionRepo{
		collection: db.Collection("questions"),
	}
}

func (r *questionRepo) Create(ctx context.Context, question *model.Question) error {
	// Generate ObjectID if not provided
	if question.ID == "" {
		question.ID = primitive.NewObjectID().Hex()
	}

	// Insert the question into MongoDB
	_, err := r.collection.InsertOne(ctx, question)
	if err != nil {
		return err
	}

	return nil
}

func (r *questionRepo) GetByID(ctx context.Context, id string) (*model.Question, error) {
	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	// Find the question by ID
	var question model.Question
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&question)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Question not found
		}
		return nil, err
	}

	return &question, nil
}

func (r *questionRepo) Update(ctx context.Context, question *model.Question) error {
	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(question.ID)
	if err != nil {
		return err
	}

	// Update the question
	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, question)
	return err
}

func (r *questionRepo) Delete(ctx context.Context, id string) error {
	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	// Delete the question
	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *questionRepo) GetByCategory(ctx context.Context, category string) ([]*model.Question, error) {
	// Find all questions for the category
	cursor, err := r.collection.Find(ctx, bson.M{"category": category})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Decode all results into slice
	var questions []*model.Question
	if err = cursor.All(ctx, &questions); err != nil {
		return nil, err
	}

	return questions, nil
}

func (r *questionRepo) GetByPriority(ctx context.Context, minPriority, maxPriority int) ([]*model.Question, error) {
	// Find questions within priority range
	cursor, err := r.collection.Find(ctx, bson.M{
		"priority": bson.M{
			"$gte": minPriority,
			"$lte": maxPriority,
		},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Decode all results into slice
	var questions []*model.Question
	if err = cursor.All(ctx, &questions); err != nil {
		return nil, err
	}

	return questions, nil
}

func (r *questionRepo) GetByType(ctx context.Context, questionType model.QuestionType) ([]*model.Question, error) {
	// Find questions by type
	cursor, err := r.collection.Find(ctx, bson.M{"type": questionType})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Decode all results into slice
	var questions []*model.Question
	if err = cursor.All(ctx, &questions); err != nil {
		return nil, err
	}

	return questions, nil
}

func (r *questionRepo) GetByIDs(ctx context.Context, ids []string) ([]*model.Question, error) {
	// Convert string IDs to ObjectIDs
	var objectIDs []primitive.ObjectID
	for _, id := range ids {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return nil, err
		}
		objectIDs = append(objectIDs, objectID)
	}

	// Find questions by IDs
	cursor, err := r.collection.Find(ctx, bson.M{"_id": bson.M{"$in": objectIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Decode all results into slice
	var questions []*model.Question
	if err = cursor.All(ctx, &questions); err != nil {
		return nil, err
	}

	return questions, nil
}

func (r *questionRepo) GetAll(ctx context.Context) ([]*model.Question, error) {
	// Find all questions
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Decode all results into slice
	var questions []*model.Question
	if err = cursor.All(ctx, &questions); err != nil {
		return nil, err
	}

	return questions, nil
}

func (r *questionRepo) GetActive(ctx context.Context) ([]*model.Question, error) {
	// For now, GetActive is the same as GetAll
	// In the future, you might add an "active" field to filter by
	return r.GetAll(ctx)
}
