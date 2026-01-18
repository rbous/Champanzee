package repository

import (
	"2026champs/internal/model"
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ReportRepo handles MongoDB operations for reports
type ReportRepo interface {
	SaveSnapshot(ctx context.Context, snapshot *model.RoomSnapshot) error
	GetSnapshot(ctx context.Context, roomCode string) (*model.RoomSnapshot, error)
	SaveAIReport(ctx context.Context, report *model.AIReport) error
	GetAIReport(ctx context.Context, roomCode string) (*model.AIReport, error)
}

type reportRepo struct {
	snapshots *mongo.Collection
	aiReports *mongo.Collection
}

// NewReportRepo creates a new report repository
func NewReportRepo(db *mongo.Database) ReportRepo {
	return &reportRepo{
		snapshots: db.Collection("room_snapshots"),
		aiReports: db.Collection("ai_reports"),
	}
}

func (r *reportRepo) SaveSnapshot(ctx context.Context, snapshot *model.RoomSnapshot) error {
	opts := options.Replace().SetUpsert(true)
	_, err := r.snapshots.ReplaceOne(ctx, bson.M{"roomCode": snapshot.RoomCode}, snapshot, opts)
	return err
}

func (r *reportRepo) GetSnapshot(ctx context.Context, roomCode string) (*model.RoomSnapshot, error) {
	var snapshot model.RoomSnapshot
	err := r.snapshots.FindOne(ctx, bson.M{"roomCode": roomCode}).Decode(&snapshot)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (r *reportRepo) SaveAIReport(ctx context.Context, report *model.AIReport) error {
	opts := options.Replace().SetUpsert(true)
	_, err := r.aiReports.ReplaceOne(ctx, bson.M{"roomCode": report.RoomCode}, report, opts)
	return err
}

func (r *reportRepo) GetAIReport(ctx context.Context, roomCode string) (*model.AIReport, error) {
	var report model.AIReport
	err := r.aiReports.FindOne(ctx, bson.M{"roomCode": roomCode}).Decode(&report)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &report, nil
}
