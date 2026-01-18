package repository

import (
	"2026champs/internal/model"
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SMRepo handles MongoDB operations for SurveyMonkey data
type SMRepo interface {
	// Collectors
	UpsertCollector(ctx context.Context, collector *model.SMCollector) error
	GetCollectorByID(ctx context.Context, collectorID string) (*model.SMCollector, error)
	GetCollectorsBySurvey(ctx context.Context, surveyID string) ([]*model.SMCollector, error)

	// Raw responses (Layer 1)
	UpsertRawResponse(ctx context.Context, response *model.SMResponseRaw) error
	GetRawResponse(ctx context.Context, responseID string) (*model.SMResponseRaw, error)
	GetRawResponsesBySurvey(ctx context.Context, surveyID string, limit int) ([]*model.SMResponseRaw, error)

	// Answers (Layer 2)
	DeleteAnswersByResponseID(ctx context.Context, responseID string) error
	InsertAnswers(ctx context.Context, answers []*model.SMAnswer) error
	GetAnswersBySurveyQuestion(ctx context.Context, surveyID, questionID string) ([]*model.SMAnswer, error)

	// Features (Layer 3)
	UpsertFeatures(ctx context.Context, features *model.SMResponseFeatures) error
	GetFeaturesByResponseID(ctx context.Context, responseID string) (*model.SMResponseFeatures, error)
	GetFeaturesBySurvey(ctx context.Context, surveyID string) ([]*model.SMResponseFeatures, error)

	// Analytics aggregations
	GetSurveySummary(ctx context.Context, surveyID string) (*model.SMSurveySummary, error)
	GetDistribution(ctx context.Context, surveyID, metric string) (*model.SMDistribution, error)
}

type smRepo struct {
	rawResponses *mongo.Collection
	answers      *mongo.Collection
	features     *mongo.Collection
	collectors   *mongo.Collection
}

// NewSMRepo creates a new SurveyMonkey repository with indexes
func NewSMRepo(db *mongo.Database) SMRepo {
	repo := &smRepo{
		rawResponses: db.Collection("sm_responses_raw"),
		answers:      db.Collection("sm_answers"),
		features:     db.Collection("sm_response_features"),
		collectors:   db.Collection("sm_collectors"),
	}

	// Create indexes
	repo.ensureIndexes(context.Background())

	return repo
}

func (r *smRepo) ensureIndexes(ctx context.Context) {
	// sm_responses_raw indexes
	r.createIndex(ctx, r.rawResponses, bson.D{{Key: "response_id", Value: 1}}, true)
	r.createIndex(ctx, r.rawResponses, bson.D{
		{Key: "survey_id", Value: 1},
		{Key: "date_modified", Value: -1},
	}, false)
	r.createIndex(ctx, r.rawResponses, bson.D{
		{Key: "collector_id", Value: 1},
		{Key: "date_modified", Value: -1},
	}, false)

	// sm_answers indexes
	r.createIndex(ctx, r.answers, bson.D{{Key: "response_id", Value: 1}}, false)
	r.createIndex(ctx, r.answers, bson.D{
		{Key: "survey_id", Value: 1},
		{Key: "question_id", Value: 1},
	}, false)
	r.createIndex(ctx, r.answers, bson.D{
		{Key: "question_id", Value: 1},
		{Key: "choice_id", Value: 1},
	}, false)

	// sm_response_features indexes
	r.createIndex(ctx, r.features, bson.D{{Key: "response_id", Value: 1}}, true)
	r.createIndex(ctx, r.features, bson.D{
		{Key: "survey_id", Value: 1},
		{Key: "submitted_at", Value: -1},
	}, false)
	r.createIndex(ctx, r.features, bson.D{
		{Key: "survey_id", Value: 1},
		{Key: "overall_satisfaction", Value: 1},
	}, false)

	// sm_collectors indexes
	r.createIndex(ctx, r.collectors, bson.D{{Key: "collector_id", Value: 1}}, true)
	r.createIndex(ctx, r.collectors, bson.D{{Key: "survey_id", Value: 1}}, false)

	log.Println("SM indexes ensured")
}

func (r *smRepo) createIndex(ctx context.Context, coll *mongo.Collection, keys bson.D, unique bool) {
	opts := options.Index().SetUnique(unique)
	_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: keys, Options: opts})
	if err != nil {
		log.Printf("Warning: failed to create index on %s: %v", coll.Name(), err)
	}
}

// Collector methods

func (r *smRepo) UpsertCollector(ctx context.Context, collector *model.SMCollector) error {
	opts := options.Replace().SetUpsert(true)
	_, err := r.collectors.ReplaceOne(ctx,
		bson.M{"collector_id": collector.CollectorID},
		collector,
		opts,
	)
	return err
}

func (r *smRepo) GetCollectorByID(ctx context.Context, collectorID string) (*model.SMCollector, error) {
	var collector model.SMCollector
	err := r.collectors.FindOne(ctx, bson.M{"collector_id": collectorID}).Decode(&collector)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &collector, nil
}

func (r *smRepo) GetCollectorsBySurvey(ctx context.Context, surveyID string) ([]*model.SMCollector, error) {
	cursor, err := r.collectors.Find(ctx, bson.M{"survey_id": surveyID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var collectors []*model.SMCollector
	if err := cursor.All(ctx, &collectors); err != nil {
		return nil, err
	}
	return collectors, nil
}

// Raw response methods (Layer 1)

func (r *smRepo) UpsertRawResponse(ctx context.Context, response *model.SMResponseRaw) error {
	response.LastSeenAt = time.Now()
	opts := options.Replace().SetUpsert(true)
	_, err := r.rawResponses.ReplaceOne(ctx,
		bson.M{"response_id": response.ResponseID},
		response,
		opts,
	)
	return err
}

func (r *smRepo) GetRawResponse(ctx context.Context, responseID string) (*model.SMResponseRaw, error) {
	var response model.SMResponseRaw
	err := r.rawResponses.FindOne(ctx, bson.M{"response_id": responseID}).Decode(&response)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (r *smRepo) GetRawResponsesBySurvey(ctx context.Context, surveyID string, limit int) ([]*model.SMResponseRaw, error) {
	opts := options.Find().SetSort(bson.D{{Key: "date_modified", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.rawResponses.Find(ctx, bson.M{"survey_id": surveyID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var responses []*model.SMResponseRaw
	if err := cursor.All(ctx, &responses); err != nil {
		return nil, err
	}
	return responses, nil
}

// Answer methods (Layer 2)

func (r *smRepo) DeleteAnswersByResponseID(ctx context.Context, responseID string) error {
	_, err := r.answers.DeleteMany(ctx, bson.M{"response_id": responseID})
	return err
}

func (r *smRepo) InsertAnswers(ctx context.Context, answers []*model.SMAnswer) error {
	if len(answers) == 0 {
		return nil
	}

	docs := make([]interface{}, len(answers))
	for i, a := range answers {
		docs[i] = a
	}

	_, err := r.answers.InsertMany(ctx, docs)
	return err
}

func (r *smRepo) GetAnswersBySurveyQuestion(ctx context.Context, surveyID, questionID string) ([]*model.SMAnswer, error) {
	cursor, err := r.answers.Find(ctx, bson.M{
		"survey_id":   surveyID,
		"question_id": questionID,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var answers []*model.SMAnswer
	if err := cursor.All(ctx, &answers); err != nil {
		return nil, err
	}
	return answers, nil
}

// Feature methods (Layer 3)

func (r *smRepo) UpsertFeatures(ctx context.Context, features *model.SMResponseFeatures) error {
	opts := options.Replace().SetUpsert(true)
	_, err := r.features.ReplaceOne(ctx,
		bson.M{"response_id": features.ResponseID},
		features,
		opts,
	)
	return err
}

func (r *smRepo) GetFeaturesByResponseID(ctx context.Context, responseID string) (*model.SMResponseFeatures, error) {
	var features model.SMResponseFeatures
	err := r.features.FindOne(ctx, bson.M{"response_id": responseID}).Decode(&features)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &features, nil
}

func (r *smRepo) GetFeaturesBySurvey(ctx context.Context, surveyID string) ([]*model.SMResponseFeatures, error) {
	opts := options.Find().SetSort(bson.D{{Key: "submitted_at", Value: -1}})
	cursor, err := r.features.Find(ctx, bson.M{"survey_id": surveyID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var features []*model.SMResponseFeatures
	if err := cursor.All(ctx, &features); err != nil {
		return nil, err
	}
	return features, nil
}

// Analytics aggregations

func (r *smRepo) GetSurveySummary(ctx context.Context, surveyID string) (*model.SMSurveySummary, error) {
	// Get all features for survey
	features, err := r.GetFeaturesBySurvey(ctx, surveyID)
	if err != nil {
		return nil, err
	}

	summary := &model.SMSurveySummary{
		TotalResponses:   len(features),
		TopFeatureCounts: []model.SMFeatureCount{},
	}

	if len(features) == 0 {
		return summary, nil
	}

	// Calculate avg satisfaction and top features
	var satSum float64
	var satCount int
	featureCounts := make(map[string]int)

	for _, f := range features {
		if f.OverallSatisfaction != nil {
			satSum += float64(*f.OverallSatisfaction)
			satCount++
		}
		if f.TopFeature != nil && *f.TopFeature != "" {
			featureCounts[*f.TopFeature]++
		}
		if summary.LatestSubmittedAt == nil || f.SubmittedAt.After(*summary.LatestSubmittedAt) {
			t := f.SubmittedAt
			summary.LatestSubmittedAt = &t
		}
	}

	if satCount > 0 {
		summary.AvgOverallSatisfaction = satSum / float64(satCount)
	}

	// Get top 5 features
	type kv struct {
		k string
		v int
	}
	var sorted []kv
	for k, v := range featureCounts {
		sorted = append(sorted, kv{k, v})
	}
	// Sort by count descending
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].v > sorted[i].v {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	for i := 0; i < len(sorted) && i < 5; i++ {
		summary.TopFeatureCounts = append(summary.TopFeatureCounts, model.SMFeatureCount{
			Feature: sorted[i].k,
			Count:   sorted[i].v,
		})
	}

	return summary, nil
}

func (r *smRepo) GetDistribution(ctx context.Context, surveyID, metric string) (*model.SMDistribution, error) {
	features, err := r.GetFeaturesBySurvey(ctx, surveyID)
	if err != nil {
		return nil, err
	}

	dist := &model.SMDistribution{
		Metric:    metric,
		Histogram: make(map[int]int),
	}

	for _, f := range features {
		var val *int
		switch metric {
		case "overall_satisfaction":
			val = f.OverallSatisfaction
		case "battery_rating":
			val = f.BatteryRating
		case "camera_rating":
			val = f.CameraRating
		}

		if val != nil {
			dist.Histogram[*val]++
			dist.TotalCount++
		}
	}

	return dist, nil
}
