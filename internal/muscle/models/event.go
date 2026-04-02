package models

import (
	"encoding/json"
	"time"
)

// EventType represents the type of event - now dynamic, stored in Redis registry
// The old constants are kept for backward compatibility but rules should use any string
type EventType string

// Legacy event type constants - these are now just defaults in the seed data
const (
	EventTypeGoal       EventType = "goal"
	EventTypeCorner     EventType = "corner"
	EventTypeFoul       EventType = "foul"
	EventTypeYellowCard EventType = "yellow_card"
	EventTypeRedCard    EventType = "red_card"
	EventTypePenalty    EventType = "penalty"
	EventTypeOffside    EventType = "offside"
)

// MatchEvent represents an event that occurs during a football match
// Enhanced with generic fields for non-sport events
type MatchEvent struct {
	EventID   string          `json:"event_id"`
	EventType EventType       `json:"event_type"`
	MatchID   string          `json:"match_id,omitempty"` // Optional - for sport events
	TeamID    string          `json:"team_id,omitempty"`    // Optional - for sport events
	PlayerID  string          `json:"player_id,omitempty"`  // Optional - for sport events
	Minute    int             `json:"minute,omitempty"`      // Optional - for sport events
	Timestamp time.Time       `json:"timestamp"`
	Metadata  json.RawMessage `json:"metadata"` // Flexible JSON for event-specific data

	// Generic fields for non-sport events (e.g., daily_login, app_shared)
	SubjectID string            `json:"subject_id,omitempty"` // What the event is about
	ActorID   string            `json:"actor_id,omitempty"`  // Who triggered the event
	Source    string            `json:"source,omitempty"`    // Where the event came from
	Context   map[string]any    `json:"context,omitempty"`   // Additional context
}

// Metadata structures for different event types
type GoalMetadata struct {
	ScorerID     string `json:"scorer_id"`
	AssistPlayer string `json:"assist_player,omitempty"`
	GoalType     string `json:"goal_type,omitempty"` // open_play, penalty, free_kick, header
	XCoord       int    `json:"x_coord,omitempty"`
	YCoord       int    `json:"y_coord,omitempty"`
}

type CardMetadata struct {
	Reason       string `json:"reason"`
	PreviousCard bool   `json:"previous_card,omitempty"` // For second yellow
}

type FoulMetadata struct {
	FoulType  string `json:"foul_type"`
	Location  string `json:"location"`
	Advantage bool   `json:"advantage,omitempty"`
}

// RuleCondition defines a condition that must be met for a rule to trigger
type RuleCondition struct {
	Field          string `json:"field"`           // e.g., "consecutive_count", "total_goals"
	Operator       string `json:"operator"`        // e.g., ">=", "==", "<"
	Value          any    `json:"value"`           // e.g., 3, "home_team"
	EvaluationType string `json:"evaluation_type"` // "simple", "aggregation", "temporal"
}

// RuleAction defines an action to execute when a rule triggers
type RuleAction struct {
	ActionType string         `json:"action_type"` // "award_points", "grant_badge", "send_notification"
	Params     map[string]any `json:"params"`
}

// Rule represents a gamification rule stored in Redis
type Rule struct {
	RuleID          string          `json:"rule_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	EventType       EventType       `json:"event_type"` // Dynamic - any string, stored as EventType for compatibility
	IsActive        bool            `json:"is_active"`
	Priority        int             `json:"priority"` // Higher = more important
	Conditions      []RuleCondition `json:"conditions"`
	TargetUsers     TargetUsers     `json:"target_users"`
	Actions         []RuleAction    `json:"actions"`
	CooldownSeconds int             `json:"cooldown_seconds"` // Prevent duplicate triggers
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// TargetUsers defines how to find users affected by a rule
type TargetUsers struct {
	QueryPattern string            `json:"query_pattern"` // Neo4j query type
	Params       map[string]string `json:"params"`        // Query parameters
}

// RuleSet is a collection of rules for batch operations
type RuleSet struct {
	Rules []Rule `json:"rules"`
}

// RuleEvaluationResult holds the result of evaluating a rule
type RuleEvaluationResult struct {
	Rule       *Rule
	Matched    bool
	Users      []string // Affected user IDs
	Actions    []RuleAction
	EvalTimeMs float64
}

// RuleEngineResult holds the complete result of processing an event
type RuleEngineResult struct {
	Event          *MatchEvent
	TriggeredRules []RuleEvaluationResult
	TotalTimeMs    float64
	Success        bool
	Error          error
	Skipped        bool   // True if event was skipped (e.g., disabled event type)
	SkipReason     string // Reason for skipping (e.g., "event_type_disabled")
}

// Badge represents a gamification badge
type Badge struct {
	BadgeID     string    `json:"badge_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon,omitempty"`
	Points      int       `json:"points"`
	Criteria    string    `json:"criteria,omitempty"`
	Category    string    `json:"category,omitempty"`
	Metric      string    `json:"metric,omitempty"`
	Target      int       `json:"target,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserBadge represents a badge earned by a user
type UserBadge struct {
	UserID   string    `json:"user_id"`
	BadgeID  string    `json:"badge_id"`
	EarnedAt time.Time `json:"earned_at"`
	Reason   string    `json:"reason,omitempty"`
}

// UserPoints represents user points data
type UserPoints struct {
	UserID    string    `json:"user_id"`
	Points    int       `json:"points"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ==================== Swagger DTOs ====================

// SwaggerMatchEvent is a Swagger-friendly version of MatchEvent
// with explicit metadata field instead of json.RawMessage
// Supports both sport events (goal, corner) and generic events (daily_login, app_shared)
// @Description Match event for testing rules - supports sport and generic event types
type SwaggerMatchEvent struct {
	EventID   string         `json:"event_id" example:"evt_12345"`
	EventType EventType      `json:"event_type" example:"goal" description:"Event type - can be sport (goal, corner) or generic (daily_login, app_shared)"`
	MatchID   string         `json:"match_id,omitempty" example:"match_98765"`
	TeamID    string         `json:"team_id,omitempty" example:"team_abc"`
	PlayerID  string         `json:"player_id,omitempty" example:"player_xyz"`
	Minute    int            `json:"minute,omitempty" example:45`
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty" description:"Flexible metadata for event-specific data (e.g., goal_type, assist_player for goals)"`

	// Generic fields for non-sport events (e.g., daily_login, app_shared, purchase_completed)
	SubjectID string         `json:"subject_id,omitempty" example:"item_123" description:"What the event is about (e.g., item purchased, page visited)"`
	ActorID   string         `json:"actor_id,omitempty" example:"user_456" description:"Who triggered the event"`
	Source    string         `json:"source,omitempty" example:"mobile_app" description:"Where the event came from"`
	Context   map[string]any `json:"context,omitempty" description:"Additional context for the event"`
}
