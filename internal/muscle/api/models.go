package api

import (
	"time"

	"gamification/models"
)

// ==================== Auth Types ====================

// BadgeSummary represents a badge summary for user responses
// @Description Summary of a badge earned by a user
type BadgeSummary struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon,omitempty"`
	Points      int       `json:"points"`
	EarnedAt    time.Time `json:"earned_at,omitempty"`
	Reason      string    `json:"reason,omitempty"`
}

// CurrentUserResponse represents the current authenticated user
// @Description Current user information response
type CurrentUserResponse struct {
	ID        string         `json:"id"`
	Email     string         `json:"email"`
	Name      string         `json:"name"`
	Points    int            `json:"points"`
	Level     int            `json:"level"`
	Badges    []BadgeSummary `json:"badges"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

// LoginResponse represents the response after successful login
// @Description Login response with JWT tokens and user info
type LoginResponse struct {
	Token        string              `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string              `json:"refreshToken" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	User         CurrentUserResponse `json:"user"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string                   `json:"status"`
	Timestamp time.Time                `json:"timestamp"`
	Services  map[string]ServiceStatus `json:"services"`
}

// ServiceStatus represents the status of a single service
type ServiceStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// ==================== Rules Types ====================

// CreateRuleRequest represents a request to create a rule
type CreateRuleRequest struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	EventType   string                 `json:"event_type"`
	Points      int                    `json:"points"`
	Multiplier  float64                `json:"multiplier"`
	Cooldown    int                    `json:"cooldown"`
	Enabled     bool                   `json:"enabled"`
	Conditions  []models.RuleCondition `json:"conditions"`
	Rewards     map[string]any         `json:"rewards"`
	Actions     []models.RuleAction    `json:"actions,omitempty"`
}

// UpdateRuleRequest represents a request to update a rule
type UpdateRuleRequest struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	EventType   string                 `json:"event_type,omitempty"`
	Points      int                    `json:"points,omitempty"`
	Multiplier  float64                `json:"multiplier,omitempty"`
	Cooldown    int                    `json:"cooldown,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Conditions  []models.RuleCondition `json:"conditions,omitempty"`
	Rewards     map[string]any         `json:"rewards,omitempty"`
	Actions     []models.RuleAction    `json:"actions,omitempty"`
}

// RuleInfo represents rule information in API responses
type RuleInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	EventType   string                 `json:"event_type"`
	Points      int                    `json:"points"`
	Multiplier  float64                `json:"multiplier"`
	Cooldown    int                    `json:"cooldown"`
	Enabled     bool                   `json:"enabled"`
	Conditions  []models.RuleCondition `json:"conditions"`
	Rewards     map[string]any         `json:"rewards"`
	Actions     []models.RuleAction    `json:"actions"`
}

// RuleResponse represents the response after creating/updating a rule
type RuleResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// RulesListResponse represents a list of rules
type RulesListResponse struct {
	Rules []RuleInfo `json:"rules"`
	Count int        `json:"count"`
}

// ==================== Users Types ====================

// RichBadgeInfo represents detailed badge info for user profile
type RichBadgeInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Metric      string    `json:"metric"`
	Target      int       `json:"target"`
	Icon        string    `json:"icon"`
	Points      int       `json:"points"`
	EarnedAt    time.Time `json:"earned_at"`
	Reason      string    `json:"reason"`
}

// RecentActivityEntry represents a recent reward action
type RecentActivityEntry struct {
	ActionType string    `json:"action_type"`
	Points     int       `json:"points"`
	Reason     string    `json:"reason"`
	Timestamp  time.Time `json:"timestamp"`
}

// UserProfileResponse represents a user's profile
type UserProfileResponse struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	Email          string                `json:"email"`
	Points         int                   `json:"points"`
	Level          int                   `json:"level"`
	CreatedAt      time.Time             `json:"created_at"`
	Stats          map[string]int        `json:"stats"`
	RichBadgeInfo  []RichBadgeInfo       `json:"rich_badge_info"`
	RecentActivity []RecentActivityEntry `json:"recent_activity,omitempty"`
}

// UsersListResponse represents a list of users
type UsersListResponse struct {
	Users  []UserProfileResponse `json:"users"`
	Count  int                   `json:"count"`
	Limit  int                   `json:"limit"`
	Offset int                   `json:"offset"`
}

// UpdatePointsRequest represents a request to update user points
// @Description Request to add, subtract, or set user points
type UpdatePointsRequest struct {
	Points    int    `json:"points" example:"100"`
	Operation string `json:"operation" example:"add"` // "add", "subtract", "set"
}

// UserPointsResponse represents the response after updating points
type UserPointsResponse struct {
	UserID  string `json:"user_id"`
	Points  int    `json:"points"`
	Message string `json:"message"`
}

// AssignBadgeRequest represents a request to assign a badge to a user
type AssignBadgeRequest struct {
	BadgeID string `json:"badge_id"`
}

// BadgeAssignResponse represents the response after assigning a badge
type BadgeAssignResponse struct {
	UserID  string `json:"user_id"`
	BadgeID string `json:"badge_id"`
	Message string `json:"message"`
}

// ==================== Badges Types ====================

// CreateBadgeRequest represents a request to create a badge
type CreateBadgeRequest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Points      int    `json:"points"`
	Criteria    string `json:"criteria"`
	Rarity      string `json:"rarity"` // common, rare, epic, legendary
}

// BadgeInfo represents badge information in API responses
type BadgeInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon,omitempty"`
	Points      int    `json:"points"`
	Category    string `json:"category,omitempty"`
	Metric      string `json:"metric,omitempty"`
	Target      int    `json:"target,omitempty"`
}

// BadgeResponse represents the response after creating a badge
type BadgeResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// BadgesListResponse represents a list of badges
type BadgesListResponse struct {
	Badges []BadgeInfo `json:"badges"`
	Count  int         `json:"count"`
}

// ==================== Test Event Types ====================

// TestEventRequest represents a request to test an event
type TestEventRequest struct {
	Event  models.MatchEvent `json:"event"`
	DryRun *bool             `json:"dry_run"` // defaults to true, pointer to allow nil detection
}

// SwaggerTestEventRequest is a Swagger-friendly version of TestEventRequest
// with explicit metadata field for better documentation
// @Description Request body for testing an event
type SwaggerTestEventRequest struct {
	// Event to test (use MatchEvent or SwaggerMatchEvent format)
	Event models.SwaggerMatchEvent `json:"event"`
	// Whether to execute actions or just evaluate (default: true)
	DryRun *bool `json:"dry_run" example:"true"`
}

// TestEventResponse represents the response for testing an event
type TestEventResponse struct {
	Matches       []RuleMatchInfo `json:"matches"`
	AffectedUsers []string        `json:"affected_users"`
	Actions       []ActionInfo    `json:"actions"`
	Executed      bool            `json:"executed"`
}

// RuleMatchInfo represents a matched rule in test response
type RuleMatchInfo struct {
	RuleID  string `json:"rule_id"`
	Name    string `json:"name"`
	Matched bool   `json:"matched"`
}

// ActionInfo represents an action to execute
type ActionInfo struct {
	ActionType string         `json:"action_type"`
	Params     map[string]any `json:"params"`
}

// ==================== Analytics Types ====================

// AnalyticsSummaryResponse represents analytics summary
type AnalyticsSummaryResponse struct {
	TotalUsers        int `json:"total_users"`
	TotalBadges       int `json:"total_badges"`
	BadgeCatalogCount int `json:"badge_catalog_count"`
	ActiveUsers       int `json:"active_users"`
	ActiveRules       int `json:"active_rules"`
	PointsDistributed int `json:"points_distributed"`
	EventsProcessed   int `json:"events_processed"`
}

// ActivityEntry represents a single reward action
type ActivityEntry struct {
	UserID     string    `json:"user_id"`
	ActionType string    `json:"action_type"`
	Points     int       `json:"points"`
	Reason     string    `json:"reason"`
	Timestamp  time.Time `json:"timestamp"`
}

// ActivityResponse represents recent activity response
type ActivityResponse struct {
	Activities []ActivityEntry `json:"activities"`
	Count      int             `json:"count"`
}

// PointsHistoryEntry represents points distribution over time
type PointsHistoryEntry struct {
	Date   string `json:"date"`
	Points int    `json:"points"`
}

// PointsHistoryResponse represents points history response
type PointsHistoryResponse struct {
	Period  string               `json:"period"`
	History []PointsHistoryEntry `json:"history"`
}

// LeaderboardEntry represents a single entry in the leaderboard
type LeaderboardEntry struct {
	Rank   int    `json:"rank"`
	UserID string `json:"user_id"`
	Score  int    `json:"score"`
}

// LeaderboardResponse represents the leaderboard response
type LeaderboardResponse struct {
	Entries []LeaderboardEntry `json:"entries"`
	Count   int                `json:"count"`
}

// MatchStatsResponse represents match statistics
type MatchStatsResponse struct {
	MatchID      string           `json:"match_id"`
	Participants []map[string]any `json:"participants"`
	Count        int              `json:"count"`
}

// ==================== Event Types API Types ====================

// EventTypeInfo represents event type information in API responses
type EventTypeInfo struct {
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// EventTypesListResponse represents a list of event types
type EventTypesListResponse struct {
	EventTypes []EventTypeInfo `json:"event_types"`
	Count      int             `json:"count"`
}

// CreateEventTypeRequest represents a request to create an event type
type CreateEventTypeRequest struct {
	Key           string         `json:"key"`
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	Category      string         `json:"category,omitempty"`
	Enabled       *bool          `json:"enabled"` // Pointer to allow distinguishing between not set (default true) and explicitly false
	SamplePayload map[string]any `json:"sample_payload,omitempty"`
}

// UpdateEventTypeRequest represents a request to update an event type
type UpdateEventTypeRequest struct {
	Name          string         `json:"name,omitempty"`
	Description   string         `json:"description,omitempty"`
	Category      string         `json:"category,omitempty"`
	Enabled       *bool          `json:"enabled,omitempty"` // Pointer - only update if explicitly provided
	SamplePayload map[string]any `json:"sample_payload,omitempty"`
}

// EventTypeResponse represents the response after creating/updating an event type
type EventTypeResponse struct {
	Key     string `json:"key"`
	Message string `json:"message"`
}

// ==================== MCP Event Types API Types ====================

// MCPEventTypeInfo represents event type information for MCP API responses
type MCPEventTypeInfo struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Enabled     bool   `json:"enabled"`
}

// MCPEventTypesListResponse represents a list of event types for MCP
type MCPEventTypesListResponse struct {
	EventTypes []MCPEventTypeInfo `json:"event_types"`
	Count      int                  `json:"count"`
}

// MCPEventTypeResponse represents the response for MCP event type operations
type MCPEventTypeResponse struct {
	Success bool   `json:"success"`
	Key     string `json:"key,omitempty"`
	Message string `json:"message,omitempty"`
}

// ==================== Client Event Types ====================

// ProcessEventRequest represents a request from a client to process a user event
// @Description Request body for processing a user event
type ProcessEventRequest struct {
	UserID    string         `json:"user_id"`
	EventType string         `json:"event_type"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// ProcessEventResponse represents the response after processing a client event
type ProcessEventResponse struct {
	Message       string   `json:"message"`
	MatchedRules  []string `json:"matched_rules,omitempty"`
	PointsAwarded int      `json:"points_awarded,omitempty"`
	BadgesAwarded []string `json:"badges_awarded,omitempty"`
}

