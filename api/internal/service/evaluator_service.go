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
func (s *EvaluatorService) GenerateFollowUp(ctx context.Context, question *model.Question, player *model.Player, evalResult *model.EvaluationResult, answerText string, qProfile *model.QuestionProfile, roomMemory *model.RoomMemory, history []model.Answer, surveyIntent string, nextKey string, baseKey string) (*model.Question, error) {
	if !s.config.IsEnabled() {
		fmt.Println("[FollowUp] Config disabled, using mock")
		return s.mockFollowUp(question, nextKey, baseKey), nil
	}

	fmt.Printf("[FollowUp] Generating for Q: %s | Answer: %.50s...\n", question.Key, answerText)
	prompt := s.buildFollowUpPrompt(question, player, evalResult, answerText, qProfile, roomMemory, history, surveyIntent, baseKey)
	response, err := s.callGemini(ctx, s.config.Models.FollowUp, prompt)
	if err != nil {
		fmt.Printf("[FollowUp] Call Error: %v\n", err)
		return nil, err // Don't generate mock on error
	}
	fmt.Printf("[FollowUp] Raw Response: %s\n", response)

	var gen model.FollowUpGeneration
	if err := json.Unmarshal([]byte(response), &gen); err != nil {
		fmt.Printf("[FollowUp] JSON Error: %v\n", err)
		return nil, err // Don't generate mock on parse error
	}

	if len(gen.FollowUps) > 0 {
		fu := gen.FollowUps[0]
		return &model.Question{
			Key:       nextKey,
			ParentKey: baseKey,
			Type:      fu.Type,
			Prompt:    fu.Prompt,
			Rubric:    fu.Rubric,
			PointsMax: fu.PointsMax,
			Threshold: fu.Threshold,
			Options:   fu.Options, // Add options for MCQ
		}, nil
	}

	fmt.Printf("[FollowUp] No follow-up generated (AI decided not needed)\n")
	return nil, nil // No follow-up needed
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

	// Debug logging
	fmt.Printf("[Gemini] Requesting %s...\n", modelName)
	fmt.Printf("[Gemini] Prompt Preview: %.100s...\n", prompt) // Log start of prompt

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
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		fmt.Printf("[Gemini] JSON Unmarshal Error: %v | Body: %s\n", err, string(body))
		return "", err
	}

	if geminiResp.Error != nil {
		fmt.Printf("[Gemini] API Error: %s (Code: %d, Status: %s)\n", geminiResp.Error.Message, geminiResp.Error.Code, geminiResp.Error.Status)
		return "", fmt.Errorf("gemini api error: %s", geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}

	// Log details if empty
	finishReason := "UNKNOWN"
	if len(geminiResp.Candidates) > 0 {
		finishReason = geminiResp.Candidates[0].FinishReason
	}
	fmt.Printf("[Gemini] Empty Response! FinishReason: %s | Raw Body: %s\n", finishReason, string(body))

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
    "missing": ["list of truly missing required data based on rubric"],
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

Evaluate the answer.
- Narrow vs. Broad: If they named a narrow technical feature (e.g. "OLED", "4K"), do NOT mark "specifics" as missing. If they gave a broad/subjective reason (e.g. "price", "it's fast", "looks good"), you MAY mark "specifics" as missing to trigger one targeted drill-down.
- Leniency on Minimal Answers: If an answer is very short (3-8 words) but addresses the question, mark it as SAT but give it a low quality score (e.g. 0.3-0.5).
- Grading Scale: 
  - 0.9-1.0: Excellent, detailed, insights provided.
  - 0.6-0.9: Good, solid answer with a clear data point.
  - 0.3-0.6: Mid / Broad - technically answers but lacks depth (e.g. "the price is good").
  - 0.1-0.3: Minimalist / Horrible effort (e.g. "it is food").
  - 0.0: Irrelevant or gibberish.`,
		question.Prompt, question.Rubric, question.Threshold, answer.TextAnswer)
}

func (s *EvaluatorService) buildFollowUpPrompt(question *model.Question, player *model.Player, evalResult *model.EvaluationResult, answerText string, qProfile *model.QuestionProfile, roomMemory *model.RoomMemory, history []model.Answer, surveyIntent string, baseKey string) string {
	missingStr := strings.Join(evalResult.Signals.Missing, ", ")

	// Context construction
	peerCtx := ""
	if qProfile != nil {
		topThemes := ""
		themes := make([]string, 0)
		for t := range qProfile.ThemeCounts {
			themes = append(themes, t)
		}
		if len(themes) > 0 {
			if len(themes) > 4 {
				themes = themes[:4]
			}
			topThemes = strings.Join(themes, ", ")
		}
		if topThemes != "" {
			peerCtx += fmt.Sprintf("- What others are saying: %s\n", topThemes)
		}
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
			globalThemes = strings.Join(globals, ", ")
		}
		if globalThemes != "" {
			peerCtx += fmt.Sprintf("- Wider room themes: %s\n", globalThemes)
		}
	}

	if peerCtx == "" {
		peerCtx = "- No peer data yet (early in session)"
	}

	// History construction
	historyStr := ""
	if len(history) > 0 {
		var sb strings.Builder
		sb.WriteString("Player's Previous Context:\n")
		// Take last 3 answers
		start := 0
		if len(history) > 3 {
			start = len(history) - 3
		}
		for _, ans := range history[start:] {
			sb.WriteString(fmt.Sprintf("- Said: \"%s\"\n", ans.TextAnswer))
		}
		historyStr = sb.String()
	}

	return fmt.Sprintf(`You are an efficient and friendly data collector. Your goal is to gather high-value data without badgering the user.

SURVEY CONTEXT:
Intent: "%s"
Current Question: "%s"

PLAYER DATA:
Answer: "%s"
Initial Analysis: %s (Missing: %s)
%s

TASK:
1. DECIDE: Should you ask a follow-up?
   - YES if the answer is broad/mid-tier (e.g. "price", "quality", "design") and one targeted drill-down would add high value.
   - NO if they've already provided a narrow data point (e.g. "OLED", "under $500", "brushed aluminum").
   - NO if they've already provided a "Why" in their text.
   - DO NOT ASK "Why?". Instead, ask for a specific aspect or a comparison.
2. If (and ONLY if) you must ask a follow-up:
   - Acknowledge their response with energy (e.g. "Price is always a huge factor!", "Speed is king!").
   - Ask ONE short question to get a concrete detail (e.g. "What price range are you aiming for?" or "Which part of the design really caught your eye?").
   - KEEP IT CONCISE: 1-2 short sentences max.

Return ONLY valid JSON:
{
  "followUps": [{
    "type": "ESSAY" or "MCQ",
    "prompt": "Targeted, energetic question here",
    "options": ["A", "B"], // if MCQ
    "rubric": "Short rubric",
    "pointsMax": %d,
    "threshold": %.2f,
    "reason_in_scope": "Brief reason"
  }] // Return [] if the answer is already sufficiently narrow.
}`,
		surveyIntent, question.Prompt,
		answerText, evalResult.Resolution, missingStr, historyStr,
		question.PointsMax/2, question.Threshold)
}

func (s *EvaluatorService) buildPoolPrompt(question *model.Question, surveyIntent string) string {
	return fmt.Sprintf(`Generate follow-up question pools. Return ONLY valid JSON:
{
  "clarify": [{"key": "%s.c1", "parentKey": "%s", "prompt": "...", "type": "ESSAY", "pointsMax": 30, "threshold": 0.6, "rubric": "..."}],
  "deepen": [{"key": "%s.d1", "parentKey": "%s", "prompt": "...", "type": "MCQ", "options": ["Choice 1", "Choice 2", "Choice 3", "Choice 4"], "pointsMax": 30, "rubric": "..."}],
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
	// Leniency adjustment: Basic answer (5-10 words) gets ~0.5-0.7, elaboration gets higher
	quality := float64(wordCount) / 15.0
	if quality > 1.0 {
		quality = 1.0
	}

	resolution := "UNSAT"
	// Ensure minimal viable answer passes if threshold is reasonable
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

func (s *EvaluatorService) mockFollowUp(question *model.Question, nextKey string, baseKey string) *model.Question {
	return &model.Question{
		Key:       nextKey,
		ParentKey: baseKey,
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

// CondenseProbes takes a list of raw follow-up suggestions and selects the best ones for a new survey
func (s *EvaluatorService) CondenseProbes(ctx context.Context, probes []string, intent string) ([]model.BaseQuestion, error) {
	if !s.config.IsEnabled() {
		return s.mockCondenseProbes(), nil
	}

	prompt := s.buildCondenseProbesPrompt(probes, intent)
	response, err := s.callGemini(ctx, s.config.Models.PoolGen, prompt) // Use PoolGen model or similar
	if err != nil {
		return s.mockCondenseProbes(), nil
	}

	var result struct {
		Questions []model.BaseQuestion `json:"questions"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		fmt.Printf("[CondenseProbes] JSON Error: %v\n", err)
		return s.mockCondenseProbes(), nil
	}

	return result.Questions, nil
}

func (s *EvaluatorService) buildCondenseProbesPrompt(probes []string, intent string) string {
	probesStr := ""
	if len(probes) > 20 {
		probes = probes[:20] // Limit context window
	}
	probesStr = strings.Join(probes, "\n- ")

	return fmt.Sprintf(`You are an expert survey designer.
I have a list of "follow-up questions" and "probes" that were generated by AI during previous sessions of this survey (or similar ones).
I am creating a NEW version of the survey.
My Intent for this new survey is: "%s"

Here are the candidate follow-up questions from past sessions:
- %s

Task:
1. Analyze the candidates.
2. Select the top 3-5 most high-value questions that would be good PERMANENT additions to the main survey.
3. Reformulate them if necessary to be more general and high-quality (standardized).
4. Return them as a JSON array of question objects.

Return ONLY valid JSON:
{
  "questions": [
    {
      "key": "Q_Auto1",
      "type": "ESSAY" or "MCQ" or "DEGREE",
      "prompt": "...",
      "rubric": "...", // if ESSAY
      "pointsMax": 50,
      "threshold": 0.6,
      "scaleMin": 1, // if DEGREE
      "scaleMax": 5, // if DEGREE
      "options": ["A", "B"] // if MCQ
    }
  ]
}`, intent, probesStr)
}

func (s *EvaluatorService) mockCondenseProbes() []model.BaseQuestion {
	return []model.BaseQuestion{
		{
			Key:       "Q_Auto1",
			Type:      model.QuestionTypeEssay,
			Prompt:    "What specific improvements would you suggest based on your previous experience?",
			Rubric:    "Look for actionable suggestions.",
			PointsMax: 50,
			Threshold: 0.6,
		},
		{
			Key:       "Q_Auto2",
			Type:      model.QuestionTypeDegree,
			Prompt:    "How relevant were the topics discussed today to your personal goals?",
			ScaleMin:  1,
			ScaleMax:  5,
			PointsMax: 50,
		},
	}
}
