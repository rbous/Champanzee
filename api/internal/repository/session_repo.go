package repository

import (
	"2026champs/internal/model"
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type SessionRepo interface {
	Create(ctx context.Context, session *model.Session) error
	GetByID(ctx context.Context, id string) (*model.Session, error)
	Update(ctx context.Context, session *model.Session) error
	Delete(ctx context.Context, id string) error
	GetByRoomCode(ctx context.Context, roomCode string) ([]*model.Session, error)
}

type sessionRepo struct {
	collection *mongo.Collection
}

func NewSessionRepo(client *mongo.Client) SessionRepo {
	db := client.Database("2026champs")
	return &sessionRepo{
		collection: db.Collection("sessions"),
	}
}

func (r *sessionRepo) Create(ctx context.Context, session *model.Session) error {
	// TODO: implement
	return nil
}

func (r *sessionRepo) GetByID(ctx context.Context, id string) (*model.Session, error) {
	// TODO: implement
	return nil, nil
}

func (r *sessionRepo) Update(ctx context.Context, session *model.Session) error {
	// TODO: implement
	return nil
}

func (r *sessionRepo) Delete(ctx context.Context, id string) error {
	// TODO: implement
	return nil
}

func (r *sessionRepo) GetByRoomCode(ctx context.Context, roomCode string) ([]*model.Session, error) {
	// TODO: implement
	return nil, nil
}
