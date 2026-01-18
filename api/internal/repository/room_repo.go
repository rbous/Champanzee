package repository

import (
	"2026champs/internal/model"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// RoomRepo handles MongoDB operations for rooms (historical persistence)
type RoomRepo interface {
	Create(ctx context.Context, room *model.Room) error
	GetByCode(ctx context.Context, code string) (*model.Room, error)
	Update(ctx context.Context, room *model.Room) error
	Delete(ctx context.Context, code string) error
	GetBySurveyID(ctx context.Context, surveyID string) ([]*model.Room, error)
}

type roomRepo struct {
	collection *mongo.Collection
}

// NewRoomRepo creates a new room repository
func NewRoomRepo(db *mongo.Database) RoomRepo {
	return &roomRepo{
		collection: db.Collection("rooms"),
	}
}

func (r *roomRepo) Create(ctx context.Context, room *model.Room) error {
	room.CreatedAt = time.Now()
	_, err := r.collection.InsertOne(ctx, room)
	return err
}

func (r *roomRepo) GetByCode(ctx context.Context, code string) (*model.Room, error) {
	var room model.Room
	err := r.collection.FindOne(ctx, bson.M{"code": code}).Decode(&room)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *roomRepo) Update(ctx context.Context, room *model.Room) error {
	_, err := r.collection.ReplaceOne(ctx, bson.M{"code": room.Code}, room)
	return err
}

func (r *roomRepo) Delete(ctx context.Context, code string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"code": code})
	return err
}

func (r *roomRepo) GetBySurveyID(ctx context.Context, surveyID string) ([]*model.Room, error) {
	oid, err := primitive.ObjectIDFromHex(surveyID)
	if err != nil {
		return nil, err
	}

	cursor, err := r.collection.Find(ctx, bson.M{"surveyId": oid})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	rooms := []*model.Room{}
	if err := cursor.All(ctx, &rooms); err != nil {
		return nil, err
	}
	return rooms, nil
}
