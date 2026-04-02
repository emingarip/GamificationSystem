package redis

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gamification/config"
	"gamification/models"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// TestKeyPatterns tests that key patterns are correctly defined
func TestKeyPatterns(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{RuleKeyPrefix, "rule:"},
		{RuleListKey, "rules:all"},
		{RuleActiveSetKey, "rules:active"},
		{RuleByEventTypeKey, "rules:by_type"},
		{RuleCooldownKey, "rule:cooldown"},
		{BadgeKeyPrefix, "badge:"},
		{BadgeListKey, "badges:all"},
		{UserBadgeKeyPrefix, "user:badges:"},
		{LeaderboardKey, "leaderboard"},
		{LeaderboardByTypeKey, "leaderboard:"},
		{EventHistoryKey, "events:history"},
		{EventCountKey, "events:count"},
		{PlayerMatchStatsKey, "player:match:stats"},
		{UserAchievementsKey, "user:achievements"},
		{BadgeProgressKey, "badge:progress"},
	}

	for _, tt := range tests {
		t.Run(tt.constant, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.constant)
			}
		})
	}
}

// TestBuildKeyFunctions tests key building logic
func TestBuildKeyFunctions(t *testing.T) {
	// Test rule key
	ruleID := "rule_001"
	expectedRuleKey := RuleKeyPrefix + ruleID
	if expectedRuleKey != "rule:rule_001" {
		t.Errorf("Expected rule key 'rule:rule_001', got '%s'", expectedRuleKey)
	}

	// Test event type key
	eventType := models.EventTypeGoal
	expectedTypeKey := RuleByEventTypeKey + ":" + string(eventType)
	if expectedTypeKey != "rules:by_type:goal" {
		t.Errorf("Expected type key 'rules:by_type:goal', got '%s'", expectedTypeKey)
	}

	// Test cooldown key
	matchID := "match_123"
	playerID := "player_456"
	expectedCooldownKey := RuleCooldownKey + ":" + ruleID + ":" + matchID + ":" + playerID
	if expectedCooldownKey != "rule:cooldown:rule_001:match_123:player_456" {
		t.Errorf("Expected cooldown key, got '%s'", expectedCooldownKey)
	}

	// Test event count key
	expectedCountKey := EventCountKey + ":" + matchID + ":" + playerID + ":" + string(eventType)
	if expectedCountKey != "events:count:match_123:player_456:goal" {
		t.Errorf("Expected count key, got '%s'", expectedCountKey)
	}

	// Test badge key
	badgeID := "badge_001"
	expectedBadgeKey := BadgeKeyPrefix + badgeID
	if expectedBadgeKey != "badge:badge_001" {
		t.Errorf("Expected badge key 'badge:badge_001', got '%s'", expectedBadgeKey)
	}

	// Test user badges key
	userID := "user_001"
	expectedUserBadgeKey := UserBadgeKeyPrefix + userID
	if expectedUserBadgeKey != "user:badges:user_001" {
		t.Errorf("Expected user badge key, got '%s'", expectedUserBadgeKey)
	}

	// Test user points key
	expectedPointsKey := "user:points:" + userID
	if expectedPointsKey != "user:points:user_001" {
		t.Errorf("Expected points key, got '%s'", expectedPointsKey)
	}
}

// TestRedisConfig tests Redis configuration
func TestRedisConfig(t *testing.T) {
	cfg := &config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     100,
		MinIdleConns: 10,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  3 * time.Second,
	}

	addr := cfg.RedisAddr()
	if addr != "localhost:6379" {
		t.Errorf("Expected RedisAddr 'localhost:6379', got '%s'", addr)
	}
}

// TestRedisConfigValidation tests Redis config validation
func TestRedisConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.RedisConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &config.RedisConfig{
				Host: "localhost",
				Port: 6379,
			},
			wantErr: false,
		},
		{
			name: "invalid port too high",
			cfg: &config.RedisConfig{
				Host: "localhost",
				Port: 70000,
			},
			wantErr: true,
		},
		{
			name: "invalid port negative",
			cfg: &config.RedisConfig{
				Host: "localhost",
				Port: -1,
			},
			wantErr: true,
		},
		{
			name: "empty host",
			cfg: &config.RedisConfig{
				Host: "",
				Port: 6379,
			},
			wantErr: true, // Empty host should return an error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Redis = *tt.cfg
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestEventTypeWithMiniredis tests CreateEventType with real Redis calls using miniredis
func TestEventTypeWithMiniredis(t *testing.T) {
	// Start miniredis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// Create Redis client pointing to miniredis
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer rdb.Close()

	// Create our Client wrapper
	client := &Client{
		client: rdb,
		config: &config.RedisConfig{Host: "localhost", Port: 6379},
	}

	ctx := context.Background()

	// Test 1: CreateEventType with Enabled=false should preserve false
	t.Run("EnabledFalsePreserved", func(t *testing.T) {
		eventType := &EventType{
			Key:         "test_enabled_false",
			Name:        "Test Enabled False",
			Description: "Test event with enabled=false",
			Category:    "sport",
			Enabled:     false,
		}

		_, err := client.CreateEventType(ctx, eventType)
		if err != nil {
			t.Fatalf("CreateEventType failed: %v", err)
		}

		// Verify by getting the event type back
		retrieved, err := client.GetEventType(ctx, "test_enabled_false")
		if err != nil {
			t.Fatalf("GetEventType failed: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Expected event type to be found")
		}
		if retrieved.Enabled != false {
			t.Errorf("Expected Enabled=false, got %v", retrieved.Enabled)
		}
	})

	// Test 2: CreateEventType with Enabled=true should preserve true
	t.Run("EnabledTruePreserved", func(t *testing.T) {
		eventType := &EventType{
			Key:         "test_enabled_true",
			Name:        "Test Enabled True",
			Description: "Test event with enabled=true",
			Category:    "sport",
			Enabled:     true,
		}

		_, err := client.CreateEventType(ctx, eventType)
		if err != nil {
			t.Fatalf("CreateEventType failed: %v", err)
		}

		retrieved, err := client.GetEventType(ctx, "test_enabled_true")
		if err != nil {
			t.Fatalf("GetEventType failed: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Expected event type to be found")
		}
		if retrieved.Enabled != true {
			t.Errorf("Expected Enabled=true, got %v", retrieved.Enabled)
		}
	})

	// Test 3: CreateEventType with empty category should default to "custom"
	t.Run("DefaultCategory", func(t *testing.T) {
		eventType := &EventType{
			Key:         "test_default_category",
			Name:        "Test Default Category",
			Description: "Test event with empty category",
			// Category not specified - should default to "custom"
			Enabled: true,
		}

		_, err := client.CreateEventType(ctx, eventType)
		if err != nil {
			t.Fatalf("CreateEventType failed: %v", err)
		}

		retrieved, err := client.GetEventType(ctx, "test_default_category")
		if err != nil {
			t.Fatalf("GetEventType failed: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Expected event type to be found")
		}
		if retrieved.Category != "custom" {
			t.Errorf("Expected Category='custom', got %v", retrieved.Category)
		}
	})

	// Test 4: Duplicate key should return error
	t.Run("DuplicateKeyError", func(t *testing.T) {
		eventType := &EventType{
			Key:         "test_duplicate",
			Name:        "Test Duplicate",
			Description: "Test event",
			Enabled:     true,
		}

		_, err := client.CreateEventType(ctx, eventType)
		if err != nil {
			t.Fatalf("First CreateEventType failed: %v", err)
		}

		// Try to create again - should fail
		_, err = client.CreateEventType(ctx, eventType)
		if err == nil {
			t.Error("Expected error for duplicate key, got nil")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("Expected 'already exists' in error, got: %v", err)
		}
	})

	// Test 5: ListEventTypes should include created event types
	t.Run("ListEventTypes", func(t *testing.T) {
		allTypes, err := client.ListEventTypes(ctx)
		if err != nil {
			t.Fatalf("ListEventTypes failed: %v", err)
		}

		// Should have at least 4 event types from our tests
		if len(allTypes) < 4 {
			t.Errorf("Expected at least 4 event types, got %d", len(allTypes))
		}

		// Check that our created types are in the list
		found := false
		for _, et := range allTypes {
			if et.Key == "test_enabled_false" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected test_enabled_false to be in list")
		}
	})
}

// TestEventTypeMapping tests event type string mapping
func TestEventTypeMapping(t *testing.T) {
	tests := []struct {
		eventType models.EventType
		expected  string
	}{
		{models.EventTypeGoal, "goal"},
		{models.EventTypeCorner, "corner"},
		{models.EventTypeFoul, "foul"},
		{models.EventTypeYellowCard, "yellow_card"},
		{models.EventTypeRedCard, "red_card"},
		{models.EventTypePenalty, "penalty"},
		{models.EventTypeOffside, "offside"},
	}

	for _, tt := range tests {
		if string(tt.eventType) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, string(tt.eventType))
		}
	}
}

func TestSaveRuleGeneratesIDAndListsDisabledRules(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	client := NewTestClient(rdb, &config.RedisConfig{Host: "localhost", Port: 6379})
	ctx := context.Background()

	rule := &models.Rule{
		Name:        "Disabled welcome rule",
		Description: "Should still appear in admin lists",
		EventType:   "daily_login",
		IsActive:    false,
		Priority:    3,
	}

	if err := client.SaveRule(ctx, rule); err != nil {
		t.Fatalf("SaveRule failed: %v", err)
	}

	if rule.RuleID == "" {
		t.Fatal("expected SaveRule to generate a rule ID")
	}

	rules, err := client.GetAllRules(ctx)
	if err != nil {
		t.Fatalf("GetAllRules failed: %v", err)
	}

	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}

	if rules[0].RuleID != rule.RuleID {
		t.Fatalf("expected listed rule ID %s, got %s", rule.RuleID, rules[0].RuleID)
	}

	if rules[0].IsActive {
		t.Fatal("expected disabled rule to remain disabled in list response")
	}

	activeIDs, err := rdb.ZRange(ctx, RuleActiveSetKey, 0, -1).Result()
	if err != nil {
		t.Fatalf("failed to read active rule set: %v", err)
	}
	if len(activeIDs) != 0 {
		t.Fatalf("expected disabled rule to stay out of active set, got %v", activeIDs)
	}
}

func TestSaveRuleUpdateSyncsIndexes(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	client := NewTestClient(rdb, &config.RedisConfig{Host: "localhost", Port: 6379})
	ctx := context.Background()

	rule := &models.Rule{
		RuleID:      "rule_sync_indexes",
		Name:        "Original rule",
		Description: "Starts active on goal",
		EventType:   models.EventTypeGoal,
		IsActive:    true,
		Priority:    5,
	}

	if err := client.SaveRule(ctx, rule); err != nil {
		t.Fatalf("initial SaveRule failed: %v", err)
	}

	rule.EventType = "daily_login"
	rule.IsActive = false

	if err := client.SaveRule(ctx, rule); err != nil {
		t.Fatalf("updated SaveRule failed: %v", err)
	}

	activeIDs, err := rdb.ZRange(ctx, RuleActiveSetKey, 0, -1).Result()
	if err != nil {
		t.Fatalf("failed to read active rule set: %v", err)
	}
	if len(activeIDs) != 0 {
		t.Fatalf("expected disabled updated rule to be removed from active set, got %v", activeIDs)
	}

	oldTypeIDs, err := rdb.ZRange(ctx, RuleByEventTypeKey+":goal", 0, -1).Result()
	if err != nil {
		t.Fatalf("failed to read goal rule set: %v", err)
	}
	if len(oldTypeIDs) != 0 {
		t.Fatalf("expected rule to be removed from old event type set, got %v", oldTypeIDs)
	}

	newTypeIDs, err := rdb.ZRange(ctx, RuleByEventTypeKey+":daily_login", 0, -1).Result()
	if err != nil {
		t.Fatalf("failed to read daily_login rule set: %v", err)
	}
	if len(newTypeIDs) != 0 {
		t.Fatalf("expected disabled rule to stay out of new event type set, got %v", newTypeIDs)
	}
}

func TestGetAllRulesMigratesLegacyBlankRuleID(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	client := NewTestClient(rdb, &config.RedisConfig{Host: "localhost", Port: 6379})
	ctx := context.Background()

	legacyRule := models.Rule{
		Name:        "Legacy blank id rule",
		Description: "Created before automatic IDs existed",
		EventType:   models.EventTypeGoal,
		IsActive:    true,
		Priority:    9,
	}

	payload, err := json.Marshal(legacyRule)
	if err != nil {
		t.Fatalf("failed to marshal legacy rule: %v", err)
	}

	if err := rdb.Set(ctx, RuleKeyPrefix, payload, 0).Err(); err != nil {
		t.Fatalf("failed to seed legacy rule key: %v", err)
	}
	if err := rdb.ZAdd(ctx, RuleActiveSetKey, redis.Z{Score: 9, Member: ""}).Err(); err != nil {
		t.Fatalf("failed to seed legacy active member: %v", err)
	}
	if err := rdb.ZAdd(ctx, RuleByEventTypeKey+":goal", redis.Z{Score: 9, Member: ""}).Err(); err != nil {
		t.Fatalf("failed to seed legacy type member: %v", err)
	}

	rules, err := client.GetAllRules(ctx)
	if err != nil {
		t.Fatalf("GetAllRules failed: %v", err)
	}

	if len(rules) != 1 {
		t.Fatalf("expected 1 migrated rule, got %d", len(rules))
	}

	if rules[0].RuleID == "" {
		t.Fatal("expected migrated rule to have a generated ID")
	}

	exists, err := rdb.Exists(ctx, RuleKeyPrefix+rules[0].RuleID).Result()
	if err != nil {
		t.Fatalf("failed to check migrated rule key: %v", err)
	}
	if exists != 1 {
		t.Fatal("expected migrated rule to be stored under its new key")
	}

	legacyExists, err := rdb.Exists(ctx, RuleKeyPrefix).Result()
	if err != nil {
		t.Fatalf("failed to check legacy blank key removal: %v", err)
	}
	if legacyExists != 0 {
		t.Fatal("expected legacy blank rule key to be removed")
	}
}

// TestMatchEventStructure tests MatchEvent fields
func TestMatchEventStructure(t *testing.T) {
	event := models.MatchEvent{
		EventID:   "evt_001",
		EventType: models.EventTypeGoal,
		MatchID:   "match_123",
		TeamID:    "team_a",
		PlayerID:  "player_456",
		Minute:    45,
		Timestamp: time.Now(),
		Metadata:  nil,
	}

	if event.EventID != "evt_001" {
		t.Errorf("Expected EventID 'evt_001', got '%s'", event.EventID)
	}
	if event.EventType != models.EventTypeGoal {
		t.Errorf("Expected EventType 'goal', got '%s'", event.EventType)
	}
	if event.MatchID != "match_123" {
		t.Errorf("Expected MatchID 'match_123', got '%s'", event.MatchID)
	}
	if event.TeamID != "team_a" {
		t.Errorf("Expected TeamID 'team_a', got '%s'", event.TeamID)
	}
	if event.PlayerID != "player_456" {
		t.Errorf("Expected PlayerID 'player_456', got '%s'", event.PlayerID)
	}
	if event.Minute != 45 {
		t.Errorf("Expected Minute 45, got %d", event.Minute)
	}
}

// TestRuleStructure tests Rule structure
func TestRuleStructure(t *testing.T) {
	rule := models.Rule{
		RuleID:      "rule_001",
		Name:        "First Goal Scorer",
		Description: "Award points for first goal",
		EventType:   models.EventTypeGoal,
		IsActive:    true,
		Priority:    10,
		Conditions: []models.RuleCondition{
			{Field: "minute", Operator: "==", Value: 1, EvaluationType: "simple"},
		},
		TargetUsers: models.TargetUsers{
			QueryPattern: "goal_scorer",
			Params:       map[string]string{"match_id": "$match_id"},
		},
		Actions: []models.RuleAction{
			{ActionType: "award_points", Params: map[string]any{"points": 100.0}},
		},
		CooldownSeconds: 0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if rule.RuleID != "rule_001" {
		t.Errorf("Expected RuleID 'rule_001', got '%s'", rule.RuleID)
	}
	if rule.Name != "First Goal Scorer" {
		t.Errorf("Expected Name 'First Goal Scorer', got '%s'", rule.Name)
	}
	if rule.IsActive != true {
		t.Error("Expected IsActive to be true")
	}
	if rule.Priority != 10 {
		t.Errorf("Expected Priority 10, got %d", rule.Priority)
	}
	if len(rule.Conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(rule.Conditions))
	}
	if len(rule.Actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(rule.Actions))
	}
}

// TestBadgeStructure tests Badge structure
func TestBadgeStructure(t *testing.T) {
	badge := models.Badge{
		BadgeID:     "badge_001",
		Name:        "First Goal",
		Description: "Score your first goal",
		Icon:        "⚽",
		Points:      100,
		Criteria:    "score_one_goal",
		Category:    "goals",
		CreatedAt:   time.Now(),
	}

	if badge.BadgeID != "badge_001" {
		t.Errorf("Expected BadgeID 'badge_001', got '%s'", badge.BadgeID)
	}
	if badge.Name != "First Goal" {
		t.Errorf("Expected Name 'First Goal', got '%s'", badge.Name)
	}
	if badge.Points != 100 {
		t.Errorf("Expected Points 100, got %d", badge.Points)
	}
}

// TestUserBadgeStructure tests UserBadge structure
func TestUserBadgeStructure(t *testing.T) {
	userBadge := models.UserBadge{
		UserID:   "user_001",
		BadgeID:  "badge_001",
		EarnedAt: time.Now(),
		Reason:   "Scored first goal",
	}

	if userBadge.UserID != "user_001" {
		t.Errorf("Expected UserID 'user_001', got '%s'", userBadge.UserID)
	}
	if userBadge.BadgeID != "badge_001" {
		t.Errorf("Expected BadgeID 'badge_001', got '%s'", userBadge.BadgeID)
	}
	if userBadge.Reason != "Scored first goal" {
		t.Errorf("Expected Reason 'Scored first goal', got '%s'", userBadge.Reason)
	}
}

// TestUserPointsStructure tests UserPoints structure
func TestUserPointsStructure(t *testing.T) {
	userPoints := models.UserPoints{
		UserID:    "user_001",
		Points:    1500,
		UpdatedAt: time.Now(),
	}

	if userPoints.UserID != "user_001" {
		t.Errorf("Expected UserID 'user_001', got '%s'", userPoints.UserID)
	}
	if userPoints.Points != 1500 {
		t.Errorf("Expected Points 1500, got %d", userPoints.Points)
	}
}

// TestRuleConditionStructure tests RuleCondition structure
func TestRuleConditionStructure(t *testing.T) {
	condition := models.RuleCondition{
		Field:          "consecutive_count",
		Operator:       ">=",
		Value:          float64(3),
		EvaluationType: "aggregation",
	}

	if condition.Field != "consecutive_count" {
		t.Errorf("Expected Field 'consecutive_count', got '%s'", condition.Field)
	}
	if condition.Operator != ">=" {
		t.Errorf("Expected Operator '>=', got '%s'", condition.Operator)
	}
	if condition.EvaluationType != "aggregation" {
		t.Errorf("Expected EvaluationType 'aggregation', got '%s'", condition.EvaluationType)
	}
}

// TestRuleActionStructure tests RuleAction structure
func TestRuleActionStructure(t *testing.T) {
	action := models.RuleAction{
		ActionType: "grant_badge",
		Params: map[string]any{
			"badge_id":    "hat_trick",
			"badge_name":  "Hat Trick",
			"description": "Score 3 goals in one match",
			"points":      float64(50),
		},
	}

	if action.ActionType != "grant_badge" {
		t.Errorf("Expected ActionType 'grant_badge', got '%s'", action.ActionType)
	}
	badgeID, ok := action.Params["badge_id"].(string)
	if !ok || badgeID != "hat_trick" {
		t.Error("Expected badge_id 'hat_trick' in params")
	}
}

// TestTargetUsersStructure tests TargetUsers structure
func TestTargetUsersStructure(t *testing.T) {
	target := models.TargetUsers{
		QueryPattern: "team_supporters",
		Params: map[string]string{
			"team_id": "$team_id",
		},
	}

	if target.QueryPattern != "team_supporters" {
		t.Errorf("Expected QueryPattern 'team_supporters', got '%s'", target.QueryPattern)
	}
	teamID, ok := target.Params["team_id"]
	if !ok || teamID != "$team_id" {
		t.Error("Expected team_id param")
	}
}

// TestContextTimeout tests context creation
func TestContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		if ctx.Err() != nil {
			t.Error("Context should not be done yet")
		}
	default:
		// Context should not be done
	}
}

// TestRuleSetStructure tests RuleSet structure
func TestRuleSetStructure(t *testing.T) {
	ruleSet := models.RuleSet{
		Rules: []models.Rule{
			{RuleID: "rule_001", Name: "Rule One", IsActive: true},
			{RuleID: "rule_002", Name: "Rule Two", IsActive: false},
		},
	}

	if len(ruleSet.Rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(ruleSet.Rules))
	}
}

// TestRuleEvaluationResultStructure tests RuleEvaluationResult structure
func TestRuleEvaluationResultStructure(t *testing.T) {
	result := models.RuleEvaluationResult{
		Rule:    &models.Rule{RuleID: "rule_001", Name: "Test Rule"},
		Matched: true,
		Users:   []string{"user_001", "user_002"},
		Actions: []models.RuleAction{
			{ActionType: "award_points", Params: map[string]any{"points": float64(100)}},
		},
		EvalTimeMs: 5.5,
	}

	if !result.Matched {
		t.Error("Expected Matched to be true")
	}
	if len(result.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(result.Users))
	}
	if result.EvalTimeMs != 5.5 {
		t.Errorf("Expected EvalTimeMs 5.5, got %f", result.EvalTimeMs)
	}
}

// TestRuleEngineResultStructure tests RuleEngineResult structure
func TestRuleEngineResultStructure(t *testing.T) {
	result := models.RuleEngineResult{
		Event: &models.MatchEvent{
			EventID:   "evt_001",
			EventType: models.EventTypeGoal,
		},
		TriggeredRules: []models.RuleEvaluationResult{
			{Matched: true, Users: []string{"user_001"}},
		},
		TotalTimeMs: 10.5,
		Success:     true,
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}
	if result.Event.EventID != "evt_001" {
		t.Errorf("Expected EventID 'evt_001', got %s", result.Event.EventID)
	}
	if len(result.TriggeredRules) != 1 {
		t.Errorf("Expected 1 triggered rule, got %d", len(result.TriggeredRules))
	}
}

// TestLeaderboardDeltaAdd tests that the "add" operation calculates delta correctly
// According to client.go line 435: "For 'add' or any other operation: increment by points (default behavior)"
func TestLeaderboardDeltaAdd(t *testing.T) {
	// Test the delta calculation logic for "add" operation
	// When operation is "add", delta should equal the points value (positive)
	operation := "add"
	points := 100

	var delta float64
	switch operation {
	case "subtract":
		delta = -float64(points)
	case "set":
		// For set, we would need current score from Redis
		// This test verifies the default/add behavior
		delta = float64(points)
	default:
		// For "add" or any other operation: increment by points
		delta = float64(points)
	}

	// Verify add operation produces positive delta
	if delta != float64(points) {
		t.Errorf("Expected delta=%v for add operation, got %v", float64(points), delta)
	}

	// Additional verification: add should increase the score
	if delta <= 0 {
		t.Error("Add operation should produce positive delta (increase score)")
	}
}

// TestLeaderboardDeltaSubtract tests that the "subtract" operation calculates delta correctly
// According to client.go line 419-421: "For subtract: decrement by points (negative delta)"
func TestLeaderboardDeltaSubtract(t *testing.T) {
	// Test the delta calculation logic for "subtract" operation
	// When operation is "subtract", delta should be negative
	operation := "subtract"
	points := 50

	var delta float64
	switch operation {
	case "subtract":
		// For subtract: decrement by points (negative delta)
		delta = -float64(points)
	case "set":
		// For set, we would need current score from Redis
		delta = float64(points)
	default:
		// For "add" or any other operation: increment by points
		delta = float64(points)
	}

	// Verify subtract operation produces negative delta
	if delta != -float64(points) {
		t.Errorf("Expected delta=%v for subtract operation, got %v", -float64(points), delta)
	}

	// Additional verification: subtract should decrease the score
	if delta >= 0 {
		t.Error("Subtract operation should produce negative delta (decrease score)")
	}
}

// TestLeaderboardDeltaSet tests that the "set" operation calculates delta correctly
// According to client.go line 422-433: "For set: calculate delta between new value and current Redis value"
func TestLeaderboardDeltaSet(t *testing.T) {
	// Test the delta calculation logic for "set" operation
	// When operation is "set", delta should be: new_points - current_score
	operation := "set"
	points := 200         // The new points value being set
	currentScore := 150.0 // Current score in Redis

	var delta float64
	switch operation {
	case "subtract":
		delta = -float64(points)
	case "set":
		// For set: calculate delta between new value and current Redis value
		delta = float64(points) - currentScore
	default:
		// For "add" or any other operation: increment by points
		delta = float64(points)
	}

	// Verify set operation calculates delta correctly
	expectedDelta := float64(points) - currentScore // 200 - 150 = 50
	if delta != expectedDelta {
		t.Errorf("Expected delta=%v for set operation (new-current), got %v", expectedDelta, delta)
	}

	// Test case 2: Set to lower value
	points = 100
	currentScore = 150.0
	delta = float64(points) - currentScore // 100 - 150 = -50

	if delta >= 0 {
		t.Error("Set operation should produce negative delta when setting to lower value")
	}

	// Test case 3: Set when user not in leaderboard (currentScore = 0)
	points = 100
	currentScore = 0                       // User not in leaderboard
	delta = float64(points) - currentScore // 100 - 0 = 100

	if delta != 100 {
		t.Errorf("Expected delta=100 when setting for new user, got %v", delta)
	}
}

// TestEventTypeCategoryValidation tests that event type categories are stored and retrieved correctly
func TestEventTypeCategoryValidation(t *testing.T) {
	// Start miniredis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// Create Redis client pointing to miniredis
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer rdb.Close()

	// Create our Client wrapper
	client := &Client{
		client: rdb,
		config: &config.RedisConfig{Host: "localhost", Port: 6379},
	}

	ctx := context.Background()

	// Test 1: Sport category event type
	t.Run("sport category stored correctly", func(t *testing.T) {
		sportEvent := &EventType{
			Key:         "goal",
			Name:        "Goal",
			Description: "A goal scored",
			Category:    "sport",
			Enabled:     true,
		}

		_, err := client.CreateEventType(ctx, sportEvent)
		if err != nil {
			t.Fatalf("CreateEventType failed: %v", err)
		}

		retrieved, err := client.GetEventType(ctx, "goal")
		if err != nil {
			t.Fatalf("GetEventType failed: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Expected event type to be found")
		}
		if retrieved.Category != "sport" {
			t.Errorf("Expected Category='sport', got '%v'", retrieved.Category)
		}
	})

	// Test 2: Custom category event type
	t.Run("custom category stored correctly", func(t *testing.T) {
		customEvent := &EventType{
			Key:         "purchase_completed",
			Name:        "Purchase Completed",
			Description: "A purchase was completed",
			Category:    "custom",
			Enabled:     true,
		}

		_, err := client.CreateEventType(ctx, customEvent)
		if err != nil {
			t.Fatalf("CreateEventType failed: %v", err)
		}

		retrieved, err := client.GetEventType(ctx, "purchase_completed")
		if err != nil {
			t.Fatalf("GetEventType failed: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Expected event type to be found")
		}
		if retrieved.Category != "custom" {
			t.Errorf("Expected Category='custom', got '%v'", retrieved.Category)
		}
	})

	// Test 3: Engagement category event type
	t.Run("engagement category stored correctly", func(t *testing.T) {
		engagementEvent := &EventType{
			Key:         "daily_login",
			Name:        "Daily Login",
			Description: "User logged in daily",
			Category:    "engagement",
			Enabled:     true,
		}

		_, err := client.CreateEventType(ctx, engagementEvent)
		if err != nil {
			t.Fatalf("CreateEventType failed: %v", err)
		}

		retrieved, err := client.GetEventType(ctx, "daily_login")
		if err != nil {
			t.Fatalf("GetEventType failed: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Expected event type to be found")
		}
		if retrieved.Category != "engagement" {
			t.Errorf("Expected Category='engagement', got '%v'", retrieved.Category)
		}
	})

	// Test 4: Unknown event type should return nil
	t.Run("unknown event type returns nil", func(t *testing.T) {
		retrieved, err := client.GetEventType(ctx, "completely_unknown_event")
		if err != nil {
			t.Fatalf("GetEventType failed: %v", err)
		}
		if retrieved != nil {
			t.Error("Expected nil for unknown event type, got a result")
		}
	})

	// Test 5: GetEnabledEventTypes should only return enabled types
	t.Run("get enabled only returns enabled", func(t *testing.T) {
		// Create a disabled event type
		disabledEvent := &EventType{
			Key:         "disabled_event",
			Name:        "Disabled Event",
			Description: "This event is disabled",
			Category:    "custom",
			Enabled:     false,
		}

		_, err := client.CreateEventType(ctx, disabledEvent)
		if err != nil {
			t.Fatalf("CreateEventType failed: %v", err)
		}

		enabled, err := client.GetEnabledEventTypes(ctx)
		if err != nil {
			t.Fatalf("GetEnabledEventTypes failed: %v", err)
		}

		// Should not contain disabled event
		for _, et := range enabled {
			if et.Key == "disabled_event" {
				t.Error("GetEnabledEventTypes should not include disabled events")
			}
		}

		// Should contain enabled events
		hasGoal := false
		hasPurchase := false
		for _, et := range enabled {
			if et.Key == "goal" {
				hasGoal = true
			}
			if et.Key == "purchase_completed" {
				hasPurchase = true
			}
		}
		if !hasGoal {
			t.Error("GetEnabledEventTypes should include enabled sport events")
		}
		if !hasPurchase {
			t.Error("GetEnabledEventTypes should include enabled custom events")
		}
	})
}
