package repository

import (
	"2026champs/internal/model"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// SurveyRepo handles MongoDB operations for surveys
type SurveyRepo interface {
	Create(ctx context.Context, survey *model.Survey) (string, error)
	GetByID(ctx context.Context, id string) (*model.Survey, error)
	GetByHostID(ctx context.Context, hostID string) ([]*model.Survey, error)
	Update(ctx context.Context, survey *model.Survey) error
	Delete(ctx context.Context, id string) error
}

type surveyRepo struct {
	collection *mongo.Collection
}

// NewSurveyRepo creates a new survey repository
func NewSurveyRepo(db *mongo.Database) SurveyRepo {
	return &surveyRepo{
		collection: db.Collection("surveys"),
	}
}

func (r *surveyRepo) Create(ctx context.Context, survey *model.Survey) (string, error) {
	survey.CreatedAt = time.Now()
	survey.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, survey)
	if err != nil {
		return "", err
	}

	oid, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", nil
	}
	return oid.Hex(), nil
}

func (r *surveyRepo) GetByID(ctx context.Context, id string) (*model.Survey, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var survey model.Survey
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&survey)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	survey.ID = id
	return &survey, nil
}

func (r *surveyRepo) GetByHostID(ctx context.Context, hostID string) ([]*model.Survey, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"hostId": hostID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var surveys []*model.Survey
	if err := cursor.All(ctx, &surveys); err != nil {
		return nil, err
	}
	return surveys, nil
}

func (r *surveyRepo) Update(ctx context.Context, survey *model.Survey) error {
	oid, err := primitive.ObjectIDFromHex(survey.ID)
	if err != nil {
		return err
	}

	survey.UpdatedAt = time.Now()
	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": oid}, survey)
	return err
}

func (r *surveyRepo) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}
