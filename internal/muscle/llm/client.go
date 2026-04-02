package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/valyala/fastjson"
)

// Client represents an LLM client for vLLM
type Client struct {
	httpClient *http.Client
	baseURL    string
	modelName  string
	config     LLMClientConfig
}

// LLMClientConfig holds client configuration
type LLMClientConfig struct {
	Temperature float64
	TopP        float64
	MaxTokens   int
	Timeout     time.Duration
	MaxRetries  int
	RetryDelay  time.Duration
}

// LLMRequest represents a request to the LLM API
type LLMRequest struct {
	Model          string                 `json:"model"`
	Messages       []ChatMessage          `json:"messages"`
	Temperature    float64                `json:"temperature,omitempty"`
	TopP           float64                `json:"top_p,omitempty"`
	MaxTokens      int                    `json:"max_tokens,omitempty"`
	Stream         bool                   `json:"stream"`
	ResponseFormat map[string]interface{} `json:"response_format,omitempty"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMResponse represents a response from the LLM API
type LLMResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a choice in the response
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Rule represents a gamification rule
type Rule struct {
	RuleID      string      `json:"rule_id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	EventType   string      `json:"event_type"`
	IsActive    bool        `json:"is_active"`
	Priority    int         `json:"priority"`
	Conditions  []Condition `json:"conditions"`
	TargetUsers TargetUsers `json:"target_users"`
	Actions     []Action    `json:"actions"`
	CooldownSec int         `json:"cooldown_seconds"`
}

// Condition represents a rule condition
type Condition struct {
	Field          string      `json:"field"`
	Operator       string      `json:"operator"`
	Value          interface{} `json:"value"`
	EvaluationType string      `json:"evaluation_type"`
}

// TargetUsers represents target users for the rule
type TargetUsers struct {
	QueryPattern string                 `json:"query_pattern"`
	Params       map[string]interface{} `json:"params"`
}

// Action represents an action to be performed
type Action struct {
	ActionType string                 `json:"action_type"`
	Params     map[string]interface{} `json:"params"`
}

// NewClient creates a new LLM client
func NewClient(host string, port int, modelName string, config LLMClientConfig) *Client {
	baseURL := fmt.Sprintf("http://%s:%d", host, port)

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}
	if config.Temperature == 0 {
		config.Temperature = 0.1
	}
	if config.TopP == 0 {
		config.TopP = 0.9
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 2048
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		baseURL:   baseURL,
		modelName: modelName,
		config:    config,
	}
}

// TransformRule transforms a natural language rule to JSON
func (c *Client) TransformRule(ctx context.Context, naturalLanguageRule string) (*Rule, error) {
	// Build the request
	userPrompt := BuildUserPrompt(naturalLanguageRule)

	request := LLMRequest{
		Model: c.modelName,
		Messages: []ChatMessage{
			{Role: "system", Content: SystemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: c.config.Temperature,
		TopP:        c.config.TopP,
		MaxTokens:   c.config.MaxTokens,
		Stream:      false,
		ResponseFormat: map[string]interface{}{
			"type": "json_object",
		},
	}

	// Send request with retry logic
	var lastErr error
	for attempt := 0; attempt < c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.config.RetryDelay):
			}
		}

		rule, err := c.sendRequest(ctx, request)
		if err != nil {
			lastErr = err
			// Check if it's a retryable error
			if !isRetryable(err) {
				return nil, err
			}
			continue
		}
		return rule, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// sendRequest sends a request to the LLM API
func (c *Client) sendRequest(ctx context.Context, request LLMRequest) (*Rule, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var response LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := response.Choices[0].Message.Content

	// Parse JSON from response
	rule, err := parseRuleJSON(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rule JSON: %w", err)
	}

	return rule, nil
}

// parseRuleJSON parses the JSON content into a Rule struct
// Includes recovery for common JSON issues
func parseRuleJSON(content string) (*Rule, error) {
	// Try to find JSON in the response (in case model added markdown or extra text)
	jsonStr := extractJSON(content)

	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	// Try to parse directly first
	var rule Rule
	err := json.Unmarshal([]byte(jsonStr), &rule)
	if err != nil {
		// Try to fix common JSON issues
		fixedJSON, fixErr := fixJSON(jsonStr)
		if fixErr != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w, also failed to fix: %v", err, fixErr)
		}
		err = json.Unmarshal([]byte(fixedJSON), &rule)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fixed JSON: %w", err)
		}
	}

	// Validate required fields
	if rule.RuleID == "" {
		return nil, fmt.Errorf("rule_id is required")
	}
	if rule.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if rule.EventType == "" {
		return nil, fmt.Errorf("event_type is required")
	}
	if len(rule.Actions) == 0 {
		return nil, fmt.Errorf("at least one action is required")
	}

	return &rule, nil
}

// extractJSON extracts JSON from a string that might contain markdown or extra text
func extractJSON(content string) string {
	// Try to find JSON block in markdown
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json")
		content = content[start+7:]
		if end := strings.Index(content, "```"); end > 0 {
			content = content[:end]
		}
	} else if strings.Contains(content, "```") {
		start := strings.Index(content, "```")
		content = content[start+3:]
		if end := strings.Index(content, "```"); end > 0 {
			content = content[:end]
		}
	}

	// Trim whitespace
	content = strings.TrimSpace(content)

	// Check if it's wrapped in braces
	if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
		return content
	}

	// Try to find the first { and last }
	start := strings.Index(content, "{")
	if start == -1 {
		return content
	}
	end := strings.LastIndex(content, "}")
	if end == -1 {
		return content
	}

	return content[start : end+1]
}

// fixJSON attempts to fix common JSON issues
func fixJSON(jsonStr string) (string, error) {
	// Use fastjson to parse and get corrected JSON
	parser := &fastjson.Parser{}
	_, err := parser.Parse(jsonStr)
	if err != nil {
		// Try to fix common issues
		// Remove trailing commas
		jsonStr = strings.ReplaceAll(jsonStr, ",}", "}")
		jsonStr = strings.ReplaceAll(jsonStr, ",]", "]")

		// Try parsing again
		parser := &fastjson.Parser{}
		_, err = parser.Parse(jsonStr)
		if err != nil {
			return "", fmt.Errorf("could not fix JSON: %w", err)
		}
	}

	return jsonStr, nil
}

// isRetryable checks if an error is retryable
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Retry on connection errors, timeouts, and rate limiting
	retryablePatterns := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"i/o timeout",
		"server misbehaving",
		"429",
		"503",
		"502",
	}
	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}
	return false
}

// CheckHealth checks if the LLM service is healthy
func (c *Client) CheckHealth(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status: %d", resp.StatusCode)
	}

	return nil
}

// GetModelName returns the configured model name
func (c *Client) GetModelName() string {
	return c.modelName
}
