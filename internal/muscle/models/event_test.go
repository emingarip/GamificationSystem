package models

import (
	"encoding/json"
	"testing"
	"time"
)

// TestMatchEventSerialization tests MatchEvent JSON serialization
func TestMatchEventSerialization(t *testing.T) {
	tests := []struct {
		name    string
		event   MatchEvent
		wantErr bool
		checkFn func(*MatchEvent) bool
	}{
		{
			name: "basic goal event",
			event: MatchEvent{
				EventID:   "evt_001",
				EventType: EventTypeGoal,
				MatchID:   "match_123",
				TeamID:    "team_a",
				PlayerID:  "player_456",
				Minute:    45,
				Timestamp: time.Now(),
				Metadata:  nil,
			},
			wantErr: false,
			checkFn: func(e *MatchEvent) bool {
				return e.EventID == "evt_001" && e.EventType == EventTypeGoal && e.Minute == 45
			},
		},
		{
			name: "event with goal metadata",
			event: MatchEvent{
				EventID:   "evt_002",
				EventType: EventTypeGoal,
				MatchID:   "match_123",
				TeamID:    "team_a",
				PlayerID:  "player_456",
				Minute:    67,
				Timestamp: time.Now(),
				Metadata:  mustMarshal(GoalMetadata{ScorerID: "player_456", AssistPlayer: "player_789", GoalType: "penalty"}),
			},
			wantErr: false,
			checkFn: func(e *MatchEvent) bool {
				var meta GoalMetadata
				json.Unmarshal(e.Metadata, &meta)
				return meta.ScorerID == "player_456" && meta.AssistPlayer == "player_789" && meta.GoalType == "penalty"
			},
		},
		{
			name: "foul event with metadata",
			event: MatchEvent{
				EventID:   "evt_003",
				EventType: EventTypeFoul,
				MatchID:   "match_123",
				TeamID:    "team_b",
				PlayerID:  "player_999",
				Minute:    23,
				Timestamp: time.Now(),
				Metadata:  mustMarshal(FoulMetadata{FoulType: "tactical", Location: "midfield", Advantage: true}),
			},
			wantErr: false,
			checkFn: func(e *MatchEvent) bool {
				var meta FoulMetadata
				json.Unmarshal(e.Metadata, &meta)
				return meta.FoulType == "tactical" && meta.Advantage == true
			},
		},
		{
			name: "yellow card event with metadata",
			event: MatchEvent{
				EventID:   "evt_004",
				EventType: EventTypeYellowCard,
				MatchID:   "match_456",
				TeamID:    "team_c",
				PlayerID:  "player_111",
				Minute:    78,
				Timestamp: time.Now(),
				Metadata:  mustMarshal(CardMetadata{Reason: "unsporting_behavior", PreviousCard: false}),
			},
			wantErr: false,
			checkFn: func(e *MatchEvent) bool {
				var meta CardMetadata
				json.Unmarshal(e.Metadata, &meta)
				return meta.Reason == "unsporting_behavior" && !meta.PreviousCard
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				var decoded MatchEvent
				if err := json.Unmarshal(data, &decoded); err != nil {
					t.Errorf("Unmarshal() error = %v", err)
					return
				}

				if !tt.checkFn(&decoded) {
					t.Error("checkFn validation failed")
				}
			}
		})
	}
}

// TestRuleSerialization tests Rule JSON serialization
func TestRuleSerialization(t *testing.T) {
	tests := []struct {
		name    string
		rule    Rule
		wantErr bool
		checkFn func(*Rule) bool
	}{
		{
			name: "basic rule",
			rule: Rule{
				RuleID:      "rule_001",
				Name:        "First Goal Scorer",
				Description: "Award points for first goal",
				EventType:   EventTypeGoal,
				IsActive:    true,
				Priority:    10,
				Conditions: []RuleCondition{
					{Field: "minute", Operator: "==", Value: 1, EvaluationType: "simple"},
				},
				TargetUsers: TargetUsers{
					QueryPattern: "goal_scorer",
					Params:       map[string]string{"match_id": "$match_id"},
				},
				Actions: []RuleAction{
					{ActionType: "award_points", Params: map[string]any{"points": 100.0}},
				},
				CooldownSeconds: 0,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			wantErr: false,
			checkFn: func(r *Rule) bool {
				return r.RuleID == "rule_001" && r.IsActive == true && r.Priority == 10
			},
		},
		{
			name: "rule with multiple conditions",
			rule: Rule{
				RuleID:      "rule_002",
				Name:        "Hat Trick Bonus",
				Description: "Award bonus for scoring 3 goals",
				EventType:   EventTypeGoal,
				IsActive:    true,
				Priority:    20,
				Conditions: []RuleCondition{
					{Field: "consecutive_count", Operator: ">=", Value: float64(3), EvaluationType: "aggregation"},
					{Field: "minute", Operator: "<=", Value: float64(90), EvaluationType: "simple"},
				},
				TargetUsers: TargetUsers{
					QueryPattern: "scorer",
					Params:       map[string]string{"player_id": "$player_id"},
				},
				Actions: []RuleAction{
					{ActionType: "grant_badge", Params: map[string]any{"badge_id": "hat_trick", "badge_name": "Hat Trick", "points": 50.0}},
				},
				CooldownSeconds: 300,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			wantErr: false,
			checkFn: func(r *Rule) bool {
				return len(r.Conditions) == 2 && r.CooldownSeconds == 300
			},
		},
		{
			name: "rule with in operator",
			rule: Rule{
				RuleID:      "rule_003",
				Name:        "Set Piece Goal",
				Description: "Award points for set piece goals",
				EventType:   EventTypeGoal,
				IsActive:    true,
				Priority:    5,
				Conditions: []RuleCondition{
					{Field: "goal_type", Operator: "in", Value: []any{"penalty", "free_kick"}, EvaluationType: "simple"},
				},
				TargetUsers: TargetUsers{
					QueryPattern: "scorer",
					Params:       map[string]string{},
				},
				Actions: []RuleAction{
					{ActionType: "award_points", Params: map[string]any{"points": 150.0}},
				},
				CooldownSeconds: 0,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			wantErr: false,
			checkFn: func(r *Rule) bool {
				return r.Conditions[0].Operator == "in"
			},
		},
		{
			name: "inactive rule",
			rule: Rule{
				RuleID:      "rule_004",
				Name:        "Disabled Rule",
				Description: "This rule is disabled",
				EventType:   EventTypeFoul,
				IsActive:    false,
				Priority:    1,
				Conditions:  []RuleCondition{},
				TargetUsers: TargetUsers{QueryPattern: "", Params: nil},
				Actions:     []RuleAction{},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			wantErr: false,
			checkFn: func(r *Rule) bool {
				return r.IsActive == false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				var decoded Rule
				if err := json.Unmarshal(data, &decoded); err != nil {
					t.Errorf("Unmarshal() error = %v", err)
					return
				}

				if !tt.checkFn(&decoded) {
					t.Error("checkFn validation failed")
				}
			}
		})
	}
}

// TestRuleSetSerialization tests RuleSet serialization
func TestRuleSetSerialization(t *testing.T) {
	ruleSet := RuleSet{
		Rules: []Rule{
			{
				RuleID:   "rule_001",
				Name:     "Rule One",
				IsActive: true,
			},
			{
				RuleID:   "rule_002",
				Name:     "Rule Two",
				IsActive: false,
			},
		},
	}

	data, err := json.Marshal(ruleSet)
	if err != nil {
		t.Fatalf("Failed to marshal RuleSet: %v", err)
	}

	var decoded RuleSet
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal RuleSet: %v", err)
	}

	if len(decoded.Rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(decoded.Rules))
	}
}

// TestRuleEvaluationResult tests RuleEvaluationResult structure
func TestRuleEvaluationResult(t *testing.T) {
	result := &RuleEvaluationResult{
		Rule: &Rule{
			RuleID: "rule_001",
			Name:   "Test Rule",
		},
		Matched:    true,
		Users:      []string{"user_1", "user_2"},
		Actions:    []RuleAction{{ActionType: "award_points", Params: map[string]any{"points": 100.0}}},
		EvalTimeMs: 5.5,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal RuleEvaluationResult: %v", err)
	}

	var decoded RuleEvaluationResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal RuleEvaluationResult: %v", err)
	}

	if !decoded.Matched {
		t.Error("Expected Matched to be true")
	}
	if len(decoded.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(decoded.Users))
	}
	if decoded.EvalTimeMs != 5.5 {
		t.Errorf("Expected EvalTimeMs 5.5, got %f", decoded.EvalTimeMs)
	}
}

// TestRuleEngineResult tests RuleEngineResult structure
func TestRuleEngineResult(t *testing.T) {
	event := &MatchEvent{
		EventID:   "evt_001",
		EventType: EventTypeGoal,
		MatchID:   "match_123",
	}

	result := &RuleEngineResult{
		Event: event,
		TriggeredRules: []RuleEvaluationResult{
			{Matched: true, Users: []string{"user_1"}},
			{Matched: false, Users: []string{}},
		},
		TotalTimeMs: 10.5,
		Success:     true,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal RuleEngineResult: %v", err)
	}

	var decoded RuleEngineResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal RuleEngineResult: %v", err)
	}

	if !decoded.Success {
		t.Error("Expected Success to be true")
	}
	if len(decoded.TriggeredRules) != 2 {
		t.Errorf("Expected 2 triggered rules, got %d", len(decoded.TriggeredRules))
	}
}

// TestEventTypes tests all event types are correctly defined
func TestEventTypes(t *testing.T) {
	tests := []struct {
		eventType EventType
		expected  string
	}{
		{EventTypeGoal, "goal"},
		{EventTypeCorner, "corner"},
		{EventTypeFoul, "foul"},
		{EventTypeYellowCard, "yellow_card"},
		{EventTypeRedCard, "red_card"},
		{EventTypePenalty, "penalty"},
		{EventTypeOffside, "offside"},
	}

	for _, tt := range tests {
		if string(tt.eventType) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, string(tt.eventType))
		}
	}
}

// TestBadgeSerialization tests Badge serialization
func TestBadgeSerialization(t *testing.T) {
	badge := Badge{
		BadgeID:     "badge_001",
		Name:        "First Goal",
		Description: "Score your first goal",
		Icon:        "⚽",
		Points:      100,
		Criteria:    "score_one_goal",
		Category:    "goals",
		CreatedAt:   time.Now(),
	}

	data, err := json.Marshal(badge)
	if err != nil {
		t.Fatalf("Failed to marshal Badge: %v", err)
	}

	var decoded Badge
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Badge: %v", err)
	}

	if decoded.BadgeID != badge.BadgeID {
		t.Errorf("Expected BadgeID %s, got %s", badge.BadgeID, decoded.BadgeID)
	}
	if decoded.Points != badge.Points {
		t.Errorf("Expected Points %d, got %d", badge.Points, decoded.Points)
	}
}

// TestUserBadgeSerialization tests UserBadge serialization
func TestUserBadgeSerialization(t *testing.T) {
	userBadge := UserBadge{
		UserID:   "user_001",
		BadgeID:  "badge_001",
		EarnedAt: time.Now(),
		Reason:   "Scored first goal of the match",
	}

	data, err := json.Marshal(userBadge)
	if err != nil {
		t.Fatalf("Failed to marshal UserBadge: %v", err)
	}

	var decoded UserBadge
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal UserBadge: %v", err)
	}

	if decoded.UserID != userBadge.UserID {
		t.Errorf("Expected UserID %s, got %s", userBadge.UserID, decoded.UserID)
	}
	if decoded.Reason != userBadge.Reason {
		t.Errorf("Expected Reason %s, got %s", userBadge.Reason, decoded.Reason)
	}
}

// TestUserPointsSerialization tests UserPoints serialization
func TestUserPointsSerialization(t *testing.T) {
	userPoints := UserPoints{
		UserID:    "user_001",
		Points:    1500,
		UpdatedAt: time.Now(),
	}

	data, err := json.Marshal(userPoints)
	if err != nil {
		t.Fatalf("Failed to marshal UserPoints: %v", err)
	}

	var decoded UserPoints
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal UserPoints: %v", err)
	}

	if decoded.UserID != userPoints.UserID {
		t.Errorf("Expected UserID %s, got %s", userPoints.UserID, decoded.UserID)
	}
	if decoded.Points != userPoints.Points {
		t.Errorf("Expected Points %d, got %d", userPoints.Points, decoded.Points)
	}
}

// Helper function to marshal JSON or fail
func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
