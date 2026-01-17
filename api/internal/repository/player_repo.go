package repository

import (
	"2026champs/internal/model"
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type PlayerRepo interface {
	Create(ctx context.Context, player *model.Player) error
	GetByID(ctx context.Context, id string) (*model.Player, error)
	Update(ctx context.Context, player *model.Player) error
	Delete(ctx context.Context, id string) error
}

type playerRepo struct {
	collection *mongo.Collection
}

func NewPlayerRepo(client *mongo.Client) PlayerRepo {
	db := client.Database("2026champs")
	return &playerRepo{
		collection: db.Collection("players"),
	}
}

func (r *playerRepo) Create(ctx context.Context, player *model.Player) error {
	// TODO: implement
	return nil
}

func (r *playerRepo) GetByID(ctx context.Context, id string) (*model.Player, error) {
	// TODO: implement
	return nil, nil
}

func (r *playerRepo) Update(ctx context.Context, player *model.Player) error {
	// TODO: implement
	return nil
}

func (r *playerRepo) Delete(ctx context.Context, id string) error {
	// TODO: implement
	return nil
}
