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
	// TODO: implement
	return nil
}

func (r *roomRepo) GetByCode(ctx context.Context, code string) (*model.Room, error) {
	// TODO: implement
	return nil, nil
}

func (r *roomRepo) Update(ctx context.Context, room *model.Room) error {
	// TODO: implement
	return nil
}

func (r *roomRepo) Delete(ctx context.Context, code string) error {
	// TODO: implement
	return nil
}
