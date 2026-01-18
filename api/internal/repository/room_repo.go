package repository

import (
	"2026champs/internal/model"
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type RoomRepo interface {
	Create(ctx context.Context, room *model.Room) error
	GetByCode(ctx context.Context, code string) (*model.Room, error)
	Update(ctx context.Context, room *model.Room) error
	Delete(ctx context.Context, code string) error
}

type roomRepo struct {
	collection *mongo.Collection
}

func NewRoomRepo(client *mongo.Client) RoomRepo {
	db := client.Database("2026champs")
	return &roomRepo{
		collection: db.Collection("rooms"),
	}
}

func (r *roomRepo) Create(ctx context.Context, room *model.Room) error {
	// Insert the room into MongoDB
	_, err := r.collection.InsertOne(ctx, room)
	if err != nil {
		return err
	}

	return nil
}

func (r *roomRepo) GetByCode(ctx context.Context, code string) (*model.Room, error) {
	// Find the room by code
	var room model.Room
	err := r.collection.FindOne(ctx, map[string]interface{}{"code": code}).Decode(&room)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Room not found
		}
		return nil, err
	}

	return &room, nil
}

func (r *roomRepo) Update(ctx context.Context, room *model.Room) error {
	// Update the room by code
	_, err := r.collection.ReplaceOne(ctx, map[string]interface{}{"code": room.Code}, room)
	return err
}

func (r *roomRepo) Delete(ctx context.Context, code string) error {
	// Delete the room by code
	_, err := r.collection.DeleteOne(ctx, map[string]interface{}{"code": code})
	return err
}
