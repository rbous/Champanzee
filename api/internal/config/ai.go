package config

import "os"

// GeminiModels defines which Gemini models to use for different tasks
type GeminiModels struct {
	// L1Eval is for per-answer evaluation (needs to be fast)
	L1Eval string `json:"l1Eval"`

	// FollowUp is for on-demand follow-up generation (needs to be fast)
	FollowUp string `json:"followUp"`

	// L3Refresh is for periodic misunderstandings refresh (can be slightly slower)
	L3Refresh string `json:"l3Refresh"`

	// PoolGen is for bulk follow-up pool generation (quality over speed)
	PoolGen string `json:"poolGen"`

	// ScopeAnchor is for building scope anchor from host context (one-time, quality matters)
	ScopeAnchor string `json:"scopeAnchor"`

	// Report is for post-room AI report generation (deep analysis, not blocking)
	Report string `json:"report"`
}

// AIConfig holds all AI-related configuration
type AIConfig struct {
	APIKey    string       `json:"-"` // Never serialize
	BaseURL   string       `json:"baseUrl"`
	Models    GeminiModels `json:"models"`
	TimeoutMS int          `json:"timeoutMs"`
}

// DefaultAIConfig returns the default AI configuration
func DefaultAIConfig() *AIConfig {
	return &AIConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		BaseURL: "https://generativelanguage.googleapis.com/v1beta/models",
		Models: GeminiModels{
			// Fast models for real-time operations
			L1Eval:    getEnvOrDefault("GEMINI_MODEL_L1", "gemini-2.5-flash-preview-05-20"),
			FollowUp:  getEnvOrDefault("GEMINI_MODEL_FOLLOWUP", "gemini-2.5-flash-preview-05-20"),
			L3Refresh: getEnvOrDefault("GEMINI_MODEL_L3", "gemini-2.0-flash"),

			// Quality models for background/bulk tasks
			PoolGen:     getEnvOrDefault("GEMINI_MODEL_POOL", "gemini-2.0-flash"),
			ScopeAnchor: getEnvOrDefault("GEMINI_MODEL_SCOPE", "gemini-2.0-flash"),
			Report:      getEnvOrDefault("GEMINI_MODEL_REPORT", "gemini-2.0-flash"),
		},
		TimeoutMS: 10000, // 10 second default timeout
	}
}

// IsEnabled returns true if the AI API is configured
func (c *AIConfig) IsEnabled() bool {
	return c.APIKey != ""
}

// ModelEndpoint returns the full endpoint for a given model
func (c *AIConfig) ModelEndpoint(model string) string {
	return c.BaseURL + "/" + model + ":generateContent"
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
