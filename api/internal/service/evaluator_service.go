package service

import (
	"2026champs/internal/config"
	"2026champs/internal/model"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// EvaluatorService handles AI evaluation via Gemini API with multiple models
type EvaluatorService struct {
	config *config.AIConfig
	client *http.Client
}

// NewEvaluatorService creates a new evaluator service
func NewEvaluatorService() *EvaluatorService {
	cfg := config.DefaultAIConfig()
	return &EvaluatorService{
		config: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.TimeoutMS) * time.Millisecond,
		},
	}
}

// EvaluateAnswer evaluates an essay answer and extracts signals (L1)
func (s *EvaluatorService) EvaluateAnswer(ctx context.Context, question *model.Question, answer *model.Answer) (*model.EvaluationResult, error) {
	if !s.config.IsEnabled() {
		return s.mockEvaluate(question, answer), nil
	}

	prompt := s.buildEvaluationPrompt(question, answer)
	response, err := s.callGemini(ctx, s.config.Models.L1Eval, prompt)
	if err != nil {
		// Fallback to mock on error
		return s.mockEvaluate(question, answer), nil
	}

	var result model.EvaluationResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return s.mockEvaluate(question, answer), nil
	}

	return &result, nil
}

// GenerateFollowUp generates a personalized follow-up question (fast model)
func (s *EvaluatorService) GenerateFollowUp(ctx context.Context, question *model.Question, player *model.Player, evalResult *model.EvaluationResult, answerText string, qProfile *model.QuestionProfile, roomMemory *model.RoomMemory, history []model.Answer) (*model.Question, error) {
	if !s.config.IsEnabled() {
		return s.mockFollowUp(question), nil
	}

	prompt := s.buildFollowUpPrompt(question, player, evalResult, answerText, qProfile, roomMemory, history)
	response, err := s.callGemini(ctx, s.config.Models.FollowUp, prompt)
	if err != nil {
		return s.mockFollowUp(question), nil
	}

	var gen model.FollowUpGeneration
	if err := json.Unmarshal([]byte(response), &gen); err != nil {
		return s.mockFollowUp(question), nil
	}

	if len(gen.FollowUps) > 0 {
		fu := gen.FollowUps[0]
		return &model.Question{
			Key:       fu.QuestionKey,
			ParentKey: fu.ParentKey,
			Type:      fu.Type,
			Prompt:    fu.Prompt,
			Rubric:    fu.Rubric,
			PointsMax: fu.PointsMax,
			Threshold: fu.Threshold,
		}, nil
	}

	return s.mockFollowUp(question), nil
}

// GenerateFollowUpPool generates a pool of follow-up questions (quality model)
func (s *EvaluatorService) GenerateFollowUpPool(ctx context.Context, question *model.Question, surveyIntent string) (*model.FollowUpPool, error) {
	if !s.config.IsEnabled() {
		return s.mockPool(question), nil
	}

	prompt := s.buildPoolPrompt(question, surveyIntent)
	response, err := s.callGemini(ctx, s.config.Models.PoolGen, prompt)
	if err != nil {
		return s.mockPool(question), nil
	}

	var pool model.FollowUpPool
	if err := json.Unmarshal([]byte(response), &pool); err != nil {
		return s.mockPool(question), nil
	}

	return &pool, nil
}

// RefreshQuestionProfile refreshes misunderstandings for a question (L3)
func (s *EvaluatorService) RefreshQuestionProfile(ctx context.Context, profile *model.QuestionProfile, recentSummaries []string) (*model.QuestionProfile, error) {
	if !s.config.IsEnabled() {
		return profile, nil
	}

	prompt := s.buildL3RefreshPrompt(profile, recentSummaries)
	response, err := s.callGemini(ctx, s.config.Models.L3Refresh, prompt)
	if err != nil {
		return profile, nil
	}

	var result struct {
		Misunderstandings []string `json:"misunderstandings"`
		BestProbes        []string `json:"bestProbes"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return profile, nil
	}

	profile.Misunderstandings = result.Misunderstandings
	profile.BestProbes = result.BestProbes
	return profile, nil
}

// GenerateAIReport generates the full AI insight report (deep model)
func (s *EvaluatorService) GenerateAIReport(ctx context.Context, snapshot *model.RoomSnapshot, evidenceSamples map[string][]string) (*model.AIReport, error) {
	if !s.config.IsEnabled() {
		return s.mockReport(snapshot), nil
	}

	prompt := s.buildReportPrompt(snapshot, evidenceSamples)
	response, err := s.callGemini(ctx, s.config.Models.Report, prompt)
	if err != nil {
		return s.mockReport(snapshot), nil
	}

	var report model.AIReport
	if err := json.Unmarshal([]byte(response), &report); err != nil {
		return s.mockReport(snapshot), nil
	}

	report.RoomCode = snapshot.RoomCode
	report.Status = "ready"
	now := time.Now()
	report.ReadyAt = &now

	return &report, nil
}

// callGemini makes a request to the Gemini API
func (s *EvaluatorService) callGemini(ctx context.Context, modelName, prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"responseMimeType": "application/json",
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s?key=%s", s.config.ModelEndpoint(modelName), s.config.APIKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse Gemini response structure
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", fmt.Errorf("empty response from Gemini")
}

// Prompt builders
func (s *EvaluatorService) buildEvaluationPrompt(question *model.Question, answer *model.Answer) string {
	return fmt.Sprintf(`You are evaluating a survey response. Return ONLY valid JSON matching this schema:
{
  "resolution": "SAT" or "UNSAT",
  "qualityScore": 0.0 to 1.0,
  "signals": {
    "themes": ["theme1", "theme2"],
    "missing": ["example", "context", "specifics"],
    "specificity": 0.0 to 1.0,
    "clarity": 0.0 to 1.0,
    "sentiment": -1.0 to 1.0,
    "confidence_language": 0.0 to 1.0,
    "summary": "one sentence summary",
    "cluster_hint": "optional grouping hint",
    "risk_flags": []
  },
  "followup_hint": "clarify" or "deepen" or "branch" or "challenge",
  "notes_for_host": "optional private note"
}

Question: %s
Rubric: %s
Threshold for SAT: %.2f
Player's Answer: %s

Evaluate the answer. If qualityScore >= threshold, resolution is SAT. Otherwise UNSAT.
Extract themes, identify what's missing, and suggest a follow-up mode.`,
		question.Prompt, question.Rubric, question.Threshold, answer.TextAnswer)
}

func (s *EvaluatorService) buildFollowUpPrompt(question *model.Question, player *model.Player, evalResult *model.EvaluationResult, answerText string, qProfile *model.QuestionProfile, roomMemory *model.RoomMemory, history []model.Answer) string {
	missingStr := strings.Join(evalResult.Signals.Missing, ", ")

	// Context construction
	peerCtx := ""
	if qProfile != nil {
		topMis := ""
		if len(qProfile.Misunderstandings) > 0 {
			topMis = "- Common misunderstandings: " + strings.Join(qProfile.Misunderstandings, "; ")
		}
		topThemes := ""
		themes := make([]string, 0)
		for t := range qProfile.ThemeCounts {
			themes = append(themes, t)
		}
		if len(themes) > 0 {
			// Just take first 3 for brevity
			if len(themes) > 3 {
				themes = themes[:3]
			}
			topThemes = "- Other players discussed: " + strings.Join(themes, ", ")
		}
		peerCtx += fmt.Sprintf("\nPeer Context:\n%s\n%s", topThemes, topMis)
	}

	if roomMemory != nil {
		globalThemes := ""
		if len(roomMemory.GlobalThemesTop) > 0 {
			globals := make([]string, 0)
			for _, t := range roomMemory.GlobalThemesTop {
				globals = append(globals, t.Theme)
			}
			if len(globals) > 3 {
				globals = globals[:3]
			}
			globalThemes = "- Room-wide themes: " + strings.Join(globals, ", ")
		}
		peerCtx += fmt.Sprintf("\nRoom Context:\n%s", globalThemes)
	}

	// History construction
	historyStr := ""
	if len(history) > 0 {
		var sb strings.Builder
		sb.WriteString("\nConversation History:\n")
		for _, ans := range history {
			// We ideally need the question text too, but answer model doesn't store it directly.
			// We can infer it's a previous turn.
			// For now, let's just list the player's previous answers to give context on what they've already said.
			sb.WriteString(fmt.Sprintf("- Player previously said: \"%s\"\n", ans.TextAnswer))
		}
		historyStr = sb.String()
	}

	return fmt.Sprintf(`You are a charismatic game show host. Generate a personalized follow-up question for a player.
Return ONLY valid JSON:
{
  "followUps": [{
    "questionKey": "%s.1",
    "parentKey": "%s",
    "type": "ESSAY",
    "prompt": "follow-up text",
    "rubric": "grading guidance",
    "pointsMax": %d,
    "threshold": %.2f,
    "reason_in_scope": "why this is relevant to the player's answer and room context"
  }]
}

Original Question: %s
Player's Answer: "%s"
Evaluation: %s (Missing: %s)
Suggested Mode: %s
%s
%s

Instructions:
1. Address the player's specific answer.
2. If "Peer Context" is available, COMPARE their view to others (e.g., "Others mentioned ... how do you feel?").
3. If "Comparative" mode is suggested, explicitly ask for a tradeoff.
4. Do NOT repeat questions or ask about things the player has already mentioned in the "Conversation History".
5. Keep it short, engaging, and conversational.`,
		question.Key, question.Key, question.PointsMax/2, question.Threshold,
		question.Prompt, answerText, evalResult.Resolution, missingStr, evalResult.FollowUpHint, peerCtx, historyStr)
}

func (s *EvaluatorService) buildPoolPrompt(question *model.Question, surveyIntent string) string {
	return fmt.Sprintf(`Generate follow-up question pools. Return ONLY valid JSON:
{
  "clarify": [{"key": "%s.c1", "parentKey": "%s", "prompt": "...", "type": "ESSAY", "pointsMax": 30, "threshold": 0.6, "rubric": "..."}],
  "deepen": [{"key": "%s.d1", "parentKey": "%s", "prompt": "...", "type": "ESSAY", "pointsMax": 30, "threshold": 0.6, "rubric": "..."}],
  "branch": [],
  "challenge": [],
  "compare": []
}

Survey Intent: %s
Base Question: %s

Generate 2-3 follow-ups per category that stay within the survey's scope.`,
		question.Key, question.Key, question.Key, question.Key, surveyIntent, question.Prompt)
}

func (s *EvaluatorService) buildL3RefreshPrompt(profile *model.QuestionProfile, recentSummaries []string) string {
	summariesStr := strings.Join(recentSummaries, "\n- ")
	themes := make([]string, 0)
	for theme := range profile.ThemeCounts {
		themes = append(themes, theme)
	}
	themesStr := strings.Join(themes, ", ")

	return fmt.Sprintf(`Analyze these survey responses and identify patterns. Return ONLY valid JSON:
{
  "misunderstandings": ["bullet 1", "bullet 2", "bullet 3"],
  "bestProbes": ["suggested follow-up angle 1", "suggested follow-up angle 2"]
}

Question received %d answers. Top themes: %s

Recent response summaries:
- %s

Identify the top 3 misunderstandings and suggest 2 best follow-up angles.`,
		profile.AnswerCount, themesStr, summariesStr)
}

func (s *EvaluatorService) buildReportPrompt(snapshot *model.RoomSnapshot, evidenceSamples map[string][]string) string {
	evidenceStr := ""
	for qKey, samples := range evidenceSamples {
		evidenceStr += fmt.Sprintf("\n%s:\n- %s", qKey, strings.Join(samples, "\n- "))
	}

	return fmt.Sprintf(`Generate an AI insight report for this survey room. Return ONLY valid JSON:
{
  "executiveSummary": ["finding 1", "finding 2", "finding 3", "finding 4", "finding 5"],
  "keyThemes": [{"name": "theme", "meaning": "explanation", "percentage": 0.0, "evidenceSnippets": ["snippet"]}],
  "contrasts": [{"axis": "axis name", "sideA": "view A", "sideB": "view B", "predictor": "what predicts each"}],
  "perQuestionInsights": [{"questionKey": "Q1", "whatWorked": [], "misunderstandings": [], "missingDetails": [], "bestFollowUps": []}],
  "frictionAnalysis": [{"questionKey": "Q1", "issueDescription": "...", "hypothesizedReason": "..."}],
  "recommendedQuestions": ["new question 1", "new question 2"],
  "recommendedEdits": [{"questionKey": "Q1", "currentText": "...", "suggestedText": "...", "reason": "..."}]
}

Room Stats:
- Total players: %d
- Completion rate: %.1f%%
- Skip rate: %.1f%%

Evidence samples:%s

Generate a comprehensive but concise insight report.`,
		snapshot.TotalPlayers, snapshot.CompletionRate*100, snapshot.OverallSkipRate*100, evidenceStr)
}

// Mock implementations
func (s *EvaluatorService) mockEvaluate(question *model.Question, answer *model.Answer) *model.EvaluationResult {
	wordCount := len(strings.Fields(answer.TextAnswer))
	quality := float64(wordCount) / 50.0
	if quality > 1.0 {
		quality = 1.0
	}

	resolution := "UNSAT"
	if quality >= question.Threshold {
		resolution = "SAT"
	}

	return &model.EvaluationResult{
		Resolution:   resolution,
		QualityScore: quality,
		Signals: model.Signals{
			Themes:             []string{"general response"},
			Missing:            []string{"specifics", "examples"},
			Specificity:        quality,
			Clarity:            quality,
			Sentiment:          0.0,
			ConfidenceLanguage: quality,
			Summary:            "Mock evaluation based on response length.",
		},
		FollowUpHint: "clarify",
	}
}

func (s *EvaluatorService) mockFollowUp(question *model.Question) *model.Question {
	return &model.Question{
		Key:       question.Key + ".1",
		ParentKey: question.Key,
		Type:      model.QuestionTypeEssay,
		Prompt:    "Could you please elaborate with more specific details?",
		Rubric:    "Looking for concrete examples.",
		PointsMax: question.PointsMax / 2,
		Threshold: question.Threshold,
	}
}

func (s *EvaluatorService) mockPool(question *model.Question) *model.FollowUpPool {
	return &model.FollowUpPool{
		Clarify: []model.Question{
			{Key: question.Key + ".c1", ParentKey: question.Key, Type: model.QuestionTypeEssay,
				Prompt: "What specific details were you referring to?", PointsMax: 20, Threshold: 0.5},
		},
		Deepen: []model.Question{
			{Key: question.Key + ".d1", ParentKey: question.Key, Type: model.QuestionTypeEssay,
				Prompt: "Can you give a concrete example?", PointsMax: 20, Threshold: 0.5},
		},
	}
}

func (s *EvaluatorService) mockReport(snapshot *model.RoomSnapshot) *model.AIReport {
	now := time.Now()
	return &model.AIReport{
		RoomCode: snapshot.RoomCode,
		Status:   "ready",
		ExecutiveSummary: []string{
			"Survey completed with " + fmt.Sprintf("%d", snapshot.TotalPlayers) + " participants",
			"Mock report - enable Gemini for real insights",
		},
		ReadyAt: &now,
	}
}
