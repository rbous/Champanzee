package main

import (
	"2026champs/internal/model"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("2026champs")
	surveyColl := db.Collection("surveys")

	// Host ID observed in logs
	hostID := "host_8263b93c"

	survey := model.Survey{
		ID:     primitive.NewObjectID().Hex(),
		HostID: hostID,
		Title:  "Smartphone Launch Feedback",
		Intent: "Understand user perception, satisfaction, and improvement areas for the new device.",
		Settings: model.SurveySettings{
			SatisfactoryThreshold: 0.7,
			MaxFollowUps:          2,
			DefaultPointsMax:      100,
			AllowSkipAfter:        1,
		},
		Questions: []model.BaseQuestion{
			{
				Key:       "Q1",
				Type:      model.QuestionTypeDegree,
				Prompt:    "On a scale from 1 to 5, how satisfied are you with this smartphone overall?",
				PointsMax: 50,
				ScaleMin:  1,
				ScaleMax:  5,
			},
			{
				Key:       "Q2",
				Type:      model.QuestionTypeMCQ,
				Prompt:    "Which model did you purchase?",
				PointsMax: 50,
				Options: []string{
					"Standard Model",
					"Pro / Plus Model",
					"Ultra / Max Model",
				},
			},
			{
				Key:       "Q3",
				Type:      model.QuestionTypeEssay,
				Prompt:    "Which feature do you find the most impressive? (Display, Battery, Camera, Speed, Design)",
				Rubric:    "Look for specific mention of one of the listed features and why they like it.",
				PointsMax: 100,
				Threshold: 0.6,
			},
			{
				Key:       "Q4",
				Type:      model.QuestionTypeDegree,
				Prompt:    "How would you rate the phone’s performance during everyday tasks?",
				PointsMax: 50,
				ScaleMin:  1,
				ScaleMax:  5,
			},
			{
				Key:       "Q5",
				Type:      model.QuestionTypeEssay,
				Prompt:    "What was the main reason you chose this phone? (Price, Features, Brand, Design, Reviews)",
				Rubric:    "Identify the primary motivation factor.",
				PointsMax: 100,
				Threshold: 0.6,
			},
			{
				Key:       "Q6",
				Type:      model.QuestionTypeEssay,
				Prompt:    "What is one thing you would improve or change about this smartphone?",
				Rubric:    "Constructive criticism or specific feature requests.",
				PointsMax: 100,
				Threshold: 0.6,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = surveyColl.InsertOne(ctx, survey)
	if err != nil {
		log.Fatalf("Failed to insert survey: %v", err)
	}

	fmt.Printf("Successfully created default survey '%s' for host '%s'\n", survey.Title, hostID)
}
