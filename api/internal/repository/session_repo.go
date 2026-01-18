package repository

import (
	"2026champs/internal/model"
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	// Generate ObjectID if not provided
	if session.ID == "" {
		session.ID = primitive.NewObjectID().Hex()
	}

	// Insert the session into MongoDB
	_, err := r.collection.InsertOne(ctx, session)
	if err != nil {
		return err
	}

	return nil
}

func (r *sessionRepo) GetByID(ctx context.Context, id string) (*model.Session, error) {
	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	// Find the session by ID
	var session model.Session
	err = r.collection.FindOne(ctx, map[string]interface{}{"_id": objectID}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Session not found
		}
		return nil, err
	}

	return &session, nil
}

func (r *sessionRepo) Update(ctx context.Context, session *model.Session) error {
	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(session.ID)
	if err != nil {
		return err
	}

	// Update the session
	_, err = r.collection.ReplaceOne(ctx, map[string]interface{}{"_id": objectID}, session)
	return err
}

func (r *sessionRepo) Delete(ctx context.Context, id string) error {
	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	// Delete the session
	_, err = r.collection.DeleteOne(ctx, map[string]interface{}{"_id": objectID})
	return err
}

func (r *sessionRepo) GetByRoomCode(ctx context.Context, roomCode string) ([]*model.Session, error) {
	// Find all sessions for the room
	cursor, err := r.collection.Find(ctx, bson.M{"roomCode": roomCode})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Decode all results into slice
	var sessions []*model.Session
	if err = cursor.All(ctx, &sessions); err != nil {
		return nil, err
	}

	return sessions, nil
}
