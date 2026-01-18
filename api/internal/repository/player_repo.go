package repository

import (
	"2026champs/internal/model"
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"
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
	// Generate ObjectID if not provided
	if player.ID == "" {
		player.ID = primitive.NewObjectID().Hex()
	}

	// Insert the player into MongoDB
	_, err := r.collection.InsertOne(ctx, player)
	if err != nil {
		return err
	}

	return nil
}

func (r *playerRepo) GetByID(ctx context.Context, id string) (*model.Player, error) {
	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	// Find the player by ID
	var player model.Player
	err = r.collection.FindOne(ctx, map[string]interface{}{"_id": objectID}).Decode(&player)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Player not found
		}
		return nil, err
	}

	return &player, nil
}

func (r *playerRepo) Update(ctx context.Context, player *model.Player) error {
	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(player.ID)
	if err != nil {
		return err
	}

	// Update the player
	_, err = r.collection.ReplaceOne(ctx, map[string]interface{}{"_id": objectID}, player)
	return err
}

func (r *playerRepo) Delete(ctx context.Context, id string) error {
	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	// Delete the player
	_, err = r.collection.DeleteOne(ctx, map[string]interface{}{"_id": objectID})
	return err
}
