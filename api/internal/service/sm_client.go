package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
)

// SMClient wraps SurveyMonkey API calls
type SMClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
	maxRetries int
}

// NewSMClient creates a new SurveyMonkey API client
func NewSMClient() *SMClient {
	token := os.Getenv("SM_ACCESS_TOKEN")
	if token == "" {
		log.Println("Warning: SM_ACCESS_TOKEN not set")
	} else {
		log.Printf("SM_ACCESS_TOKEN loaded: %s", token)
	}

	return &SMClient{
		baseURL: "https://api.surveymonkey.com/v3",
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxRetries: 5,
	}
}

// SMCollectorResponse is the API response for collector creation
type SMCollectorResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	Status string `json:"status"`
}

// SMBulkResponseList is the API response for listing responses
type SMBulkResponseList struct {
	Data    []SMBulkResponse `json:"data"`
	Page    int              `json:"page"`
	PerPage int              `json:"per_page"`
	Total   int              `json:"total"`
	Links   struct {
		Self string `json:"self"`
		Next string `json:"next,omitempty"`
	} `json:"links"`
}

// SMBulkResponse is a single response in bulk list
type SMBulkResponse struct {
	ID             string `json:"id"`
	ResponseStatus string `json:"response_status"`
	DateCreated    string `json:"date_created"`
	DateModified   string `json:"date_modified"`
	CollectorID    string `json:"collector_id"`
	SurveyID       string `json:"survey_id"`
}

// SMResponseDetails is the full response details
type SMResponseDetails struct {
	ID             string   `json:"id"`
	ResponseStatus string   `json:"response_status"`
	DateCreated    string   `json:"date_created"`
	DateModified   string   `json:"date_modified"`
	CollectorID    string   `json:"collector_id"`
	SurveyID       string   `json:"survey_id"`
	Pages          []SMPage `json:"pages"`
}

// SMPage is a page in response details
type SMPage struct {
	ID        string       `json:"id"`
	Questions []SMQuestion `json:"questions"`
}

// SMQuestion is a question with answers
type SMQuestion struct {
	ID      string     `json:"id"`
	Answers []SMAnswer `json:"answers"`
}

// SMAnswer is an answer cell from SurveyMonkey
type SMAnswer struct {
	ChoiceID string `json:"choice_id,omitempty"`
	RowID    string `json:"row_id,omitempty"`
	Text     string `json:"text,omitempty"`
}

// doRequest performs HTTP request with retry logic
func (c *SMClient) doRequest(method, path string, body io.Reader) ([]byte, error) {
	url := c.baseURL + path
	log.Printf("[SM Client] %s %s", method, path)

	var lastErr error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("[SM Client] Retry attempt %d/%d for %s %s", attempt, c.maxRetries, method, path)
		}

		req, err := http.NewRequest(method, url, body)
		if err != nil {
			log.Printf("[SM Client] ERROR: Failed to create request: %v", err)
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			log.Printf("[SM Client] ERROR: HTTP request failed (attempt %d): %v", attempt+1, err)
			lastErr = err
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("[SM Client] ERROR: Failed to read response body: %v", err)
			lastErr = err
			continue
		}

		log.Printf("[SM Client] Response status: %d, body length: %d bytes", resp.StatusCode, len(respBody))

		// Handle rate limiting (429)
		if resp.StatusCode == 429 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			log.Printf("[SM Client] RATE LIMITED: Retry %d/%d in %v", attempt+1, c.maxRetries, backoff)
			time.Sleep(backoff)
			lastErr = fmt.Errorf("rate limited")
			continue
		}

		// Handle other errors
		if resp.StatusCode >= 400 {
			log.Printf("[SM Client] ERROR: API returned %d: %s", resp.StatusCode, string(respBody))
			return nil, fmt.Errorf("SM API error %d: %s", resp.StatusCode, string(respBody))
		}

		log.Printf("[SM Client] SUCCESS: %s %s completed", method, path)
		return respBody, nil
	}

	log.Printf("[SM Client] ERROR: Max retries (%d) exceeded for %s %s: %v", c.maxRetries, method, path, lastErr)
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// CreateCollector creates a weblink collector for a survey
func (c *SMClient) CreateCollector(surveyID, name string) (*SMCollectorResponse, error) {
	payload := fmt.Sprintf(`{"type":"weblink","name":"%s"}`, name)
	path := fmt.Sprintf("/surveys/%s/collectors", surveyID)

	respBody, err := c.doRequest("POST", path, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}

	var collector SMCollectorResponse
	if err := json.Unmarshal(respBody, &collector); err != nil {
		return nil, fmt.Errorf("failed to parse collector response: %w", err)
	}

	return &collector, nil
}

// ListResponses lists all responses for a survey (bulk, no answer data)
func (c *SMClient) ListResponses(surveyID string, modifiedSince *time.Time) (*SMBulkResponseList, error) {
	path := fmt.Sprintf("/surveys/%s/responses/bulk", surveyID)

	if modifiedSince != nil {
		path += "?start_modified_at=" + modifiedSince.Format(time.RFC3339)
	}

	respBody, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var list SMBulkResponseList
	if err := json.Unmarshal(respBody, &list); err != nil {
		return nil, fmt.Errorf("failed to parse response list: %w", err)
	}

	return &list, nil
}

// GetResponseDetails gets full response details including answers
func (c *SMClient) GetResponseDetails(surveyID, responseID string) (*SMResponseDetails, map[string]interface{}, error) {
	path := fmt.Sprintf("/surveys/%s/responses/%s/details", surveyID, responseID)

	respBody, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	// Parse raw JSON for storage
	var raw map[string]interface{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, nil, fmt.Errorf("failed to parse raw response: %w", err)
	}

	// Parse typed response
	var details SMResponseDetails
	if err := json.Unmarshal(respBody, &details); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response details: %w", err)
	}

	return &details, raw, nil
}

// IsConfigured returns true if access token is set
func (c *SMClient) IsConfigured() bool {
	return c.token != ""
}

// SMSurveyCreateRequest is the request to create a survey
type SMSurveyCreateRequest struct {
	Title string `json:"title"`
}

// SMSurveyResponse is the response from survey creation
type SMSurveyResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Href  string `json:"href"`
}

// SMPageResponse is a page in a survey
type SMPageResponse struct {
	ID       string `json:"id"`
	Title    string `json:"title,omitempty"`
	Position int    `json:"position"`
}

// SMQuestionCreateRequest is the request to create a question
type SMQuestionCreateRequest struct {
	Headings []struct {
		Heading string `json:"heading"`
	} `json:"headings"`
	Family   string                 `json:"family"`
	Subtype  string                 `json:"subtype"`
	Answers  map[string]interface{} `json:"answers,omitempty"`
	Position int                    `json:"position"`
}

// SMQuestionResponse is the response from question creation
type SMQuestionResponse struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	Family   string `json:"family"`
	Subtype  string `json:"subtype"`
}

// CreateSurvey creates a new survey in SurveyMonkey
func (c *SMClient) CreateSurvey(title string) (*SMSurveyResponse, error) {
	log.Printf("[SM Client] Creating survey: %s", title)
	payload, _ := json.Marshal(SMSurveyCreateRequest{Title: title})
	log.Printf("[SM Client] Survey payload: %s", string(payload))

	respBody, err := c.doRequest("POST", "/surveys", strings.NewReader(string(payload)))
	if err != nil {
		log.Printf("[SM Client] ERROR: Failed to create survey: %v", err)
		return nil, err
	}

	var survey SMSurveyResponse
	if err := json.Unmarshal(respBody, &survey); err != nil {
		log.Printf("[SM Client] ERROR: Failed to parse survey response: %v", err)
		return nil, fmt.Errorf("failed to parse survey response: %w", err)
	}

	log.Printf("[SM Client] Survey created successfully: ID=%s, Title=%s", survey.ID, survey.Title)
	return &survey, nil
}

// GetSurveyPages gets all pages in a survey
func (c *SMClient) GetSurveyPages(surveyID string) ([]SMPageResponse, error) {
	path := fmt.Sprintf("/surveys/%s/pages", surveyID)

	respBody, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []SMPageResponse `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse pages response: %w", err)
	}

	return result.Data, nil
}

// CreateQuestion creates a question in a survey page
func (c *SMClient) CreateQuestion(surveyID, pageID string, question SMQuestionCreateRequest) (*SMQuestionResponse, error) {
	path := fmt.Sprintf("/surveys/%s/pages/%s/questions", surveyID, pageID)
	log.Printf("[SM Client] Creating question in survey %s, page %s: family=%s, subtype=%s", surveyID, pageID, question.Family, question.Subtype)

	payload, _ := json.Marshal(question)
	log.Printf("[SM Client] Question payload: %s", string(payload))

	respBody, err := c.doRequest("POST", path, strings.NewReader(string(payload)))
	if err != nil {
		log.Printf("[SM Client] ERROR: Failed to create question: %v", err)
		return nil, err
	}

	var q SMQuestionResponse
	if err := json.Unmarshal(respBody, &q); err != nil {
		log.Printf("[SM Client] ERROR: Failed to parse question response: %v", err)
		return nil, fmt.Errorf("failed to parse question response: %w", err)
	}

	log.Printf("[SM Client] Question created successfully: ID=%s, Position=%d", q.ID, q.Position)
	return &q, nil
}
