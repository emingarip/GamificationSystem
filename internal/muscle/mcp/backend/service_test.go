package backend

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"gamification/config"
	redisclient "gamification/redis"

	"github.com/alicebob/miniredis/v2"
	redigo "github.com/redis/go-redis/v9"
)

// testRedisClient is a helper to create a Redis client for testing
func testRedisClient(mr *miniredis.Miniredis) *redisclient.Client {
	rdb := redigo.NewClient(&redigo.Options{
		Addr: mr.Addr(),
	})
	return redisclient.NewTestClient(rdb, &config.RedisConfig{Host: "localhost", Port: 6379})
}

// TestIsSportEventNilRedis tests that IsSportEvent handles nil Redis gracefully
func TestIsSportEventNilRedis(t *testing.T) {
	svc := &Service{
		redis: nil,
	}
	ctx := context.Background()

	t.Run("nil redis returns false", func(t *testing.T) {
		result := svc.IsSportEvent(ctx, "goal")
		if result != false {
			t.Error("Expected IsSportEvent to return false when Redis is nil")
		}
	})
}

// TestIsSportEventEmptyKey tests that IsSportEvent handles empty event type
func TestIsSportEventEmptyKey(t *testing.T) {
	svc := &Service{
		redis: nil,
	}
	ctx := context.Background()

	t.Run("empty event type returns false", func(t *testing.T) {
		result := svc.IsSportEvent(ctx, "")
		if result != false {
			t.Error("Expected IsSportEvent to return false for empty event type")
		}
	})
}

// TestIsSportEventWithRegistry tests IsSportEvent with real Redis registry
func TestIsSportEventWithRegistry(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rClient := testRedisClient(mr)
	ctx := context.Background()

	svc := &Service{
		redis: rClient,
	}

	// Test 1: Sport category event type
	t.Run("sport category returns true", func(t *testing.T) {
		sportEvent := &redisclient.EventType{
			Key:         "goal",
			Name:        "Goal",
			Description: "A goal scored",
			Category:    "sport",
			Enabled:     true,
		}
		_, err := rClient.CreateEventType(ctx, sportEvent)
		if err != nil {
			t.Fatalf("Failed to create sport event type: %v", err)
		}

		isSport := svc.IsSportEvent(ctx, "goal")
		if !isSport {
			t.Error("Expected IsSportEvent('goal') to return true for category=sport")
		}
	})

	// Test 2: Custom category event type
	t.Run("custom category returns false", func(t *testing.T) {
		customEvent := &redisclient.EventType{
			Key:         "purchase_completed",
			Name:        "Purchase Completed",
			Description: "A purchase was completed",
			Category:    "custom",
			Enabled:     true,
		}
		_, err := rClient.CreateEventType(ctx, customEvent)
		if err != nil {
			t.Fatalf("Failed to create custom event type: %v", err)
		}

		isSport := svc.IsSportEvent(ctx, "purchase_completed")
		if isSport {
			t.Error("Expected IsSportEvent('purchase_completed') to return false for category=custom")
		}
	})

	// Test 3: Engagement category event type
	t.Run("engagement category returns false", func(t *testing.T) {
		engagementEvent := &redisclient.EventType{
			Key:         "daily_login",
			Name:        "Daily Login",
			Description: "User logged in daily",
			Category:    "engagement",
			Enabled:     true,
		}
		_, err := rClient.CreateEventType(ctx, engagementEvent)
		if err != nil {
			t.Fatalf("Failed to create engagement event type: %v", err)
		}

		isSport := svc.IsSportEvent(ctx, "daily_login")
		if isSport {
			t.Error("Expected IsSportEvent('daily_login') to return false for category=engagement")
		}
	})

	// Test 4: Unknown event type returns false
	t.Run("unknown event type returns false", func(t *testing.T) {
		isSport := svc.IsSportEvent(ctx, "completely_unknown_event_type")
		if isSport {
			t.Error("Expected IsSportEvent('unknown') to return false (safe default)")
		}
	})
}

// TestRegistryBackedValidation tests that validation runs based on event type category
func TestRegistryBackedValidation(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rClient := testRedisClient(mr)
	ctx := context.Background()

	svc := &Service{
		redis: rClient,
		// ruleEngine is nil - we'll get "Rule engine not available" after validation passes
	}

	// Test 1: Sport event WITHOUT match_id should FAIL validation
	t.Run("sport event requires match_id validation", func(t *testing.T) {
		sportEvent := &redisclient.EventType{
			Key:         "goal",
			Name:        "Goal",
			Description: "A goal scored",
			Category:    "sport",
			Enabled:     true,
		}
		_, err := rClient.CreateEventType(ctx, sportEvent)
		if err != nil {
			t.Fatalf("Failed to create sport event type: %v", err)
		}

		eventJSON := map[string]any{
			"event_type": "goal",
			"player_id":  "player-123",
			// missing match_id - should fail validation
		}

		_, err = svc.TestEvent(ctx, eventJSON, true)
		if err == nil {
			t.Error("Expected validation error for sport event without match_id")
		}
		if err != nil && !strings.Contains(err.Error(), "match_id is required") {
			t.Errorf("Expected 'match_id is required' error, got: %v", err)
		}
	})

	// Test 2: Sport event WITHOUT player_id should FAIL validation
	t.Run("sport event requires player_id validation", func(t *testing.T) {
		eventJSON := map[string]any{
			"event_type": "goal",
			"match_id":   "match-123",
			// missing player_id - should fail validation
		}

		_, err = svc.TestEvent(ctx, eventJSON, true)
		if err == nil {
			t.Error("Expected validation error for sport event without player_id")
		}
		if err != nil && !strings.Contains(err.Error(), "player_id is required") {
			t.Errorf("Expected 'player_id is required' error, got: %v", err)
		}
	})

	// Test 3: Custom event should NOT require match_id/player_id
	t.Run("custom event doesn't require match_id", func(t *testing.T) {
		eventJSON := map[string]any{
			"event_type": "purchase_completed",
			"subject_id": "user-123",
		}

		_, err = svc.TestEvent(ctx, eventJSON, true)
		// Should NOT get validation error - should get "Rule engine not available"
		if err != nil && strings.Contains(err.Error(), "validation error") {
			t.Errorf("Custom event should NOT require validation, got: %v", err)
		}
	})

	// Test 4: Engagement event should NOT require match_id/player_id
	t.Run("engagement event doesn't require match_id", func(t *testing.T) {
		eventJSON := map[string]any{
			"event_type": "daily_login",
			"subject_id": "user-456",
		}

		_, err = svc.TestEvent(ctx, eventJSON, true)
		if err != nil && strings.Contains(err.Error(), "validation error") {
			t.Errorf("Engagement event should NOT require validation, got: %v", err)
		}
	})

	// Test 5: Unknown event type should NOT require match_id/player_id
	t.Run("unknown event type defaults to no validation", func(t *testing.T) {
		eventJSON := map[string]any{
			"event_type": "new_event_type_not_in_registry",
		}

		_, err = svc.TestEvent(ctx, eventJSON, true)
		if err != nil && strings.Contains(err.Error(), "validation error") {
			t.Error("Unknown event type should pass validation (default to non-sport)")
		}
	})

	// Test 6: Sport event WITH both match_id and player_id should PASS validation
	t.Run("sport event with all required fields passes validation", func(t *testing.T) {
		eventJSON := map[string]any{
			"event_type": "goal",
			"match_id":   "match-123",
			"player_id":  "player-456",
		}

		_, err = svc.TestEvent(ctx, eventJSON, true)
		if err != nil && strings.Contains(err.Error(), "validation error") {
			t.Error("Sport event with match_id and player_id should pass validation")
		}
	})
}

// TestParseMatchEventFields tests that JSON fields are correctly parsed into MatchEvent
// This test uses the production parseMatchEvent helper to ensure the parser is tested
func TestParseMatchEventFields(t *testing.T) {
	// Test 1: Generic event with generic fields
	t.Run("generic event with all generic fields", func(t *testing.T) {
		eventJSON := map[string]any{
			"event_type": "app_shared",
			"event_id":   "evt_test_123",
			"subject_id": "article_456",
			"actor_id":   "user_789",
			"source":     "mobile_app",
			"context": map[string]any{
				"app_version": "2.0.0",
				"platform":    "ios",
			},
		}

		event := parseMatchEvent(eventJSON)

		// Assert all generic fields
		if event.EventID != "evt_test_123" {
			t.Errorf("Expected EventID='evt_test_123', got '%s'", event.EventID)
		}
		if event.EventType != "app_shared" {
			t.Errorf("Expected EventType='app_shared', got '%s'", event.EventType)
		}
		if event.SubjectID != "article_456" {
			t.Errorf("Expected SubjectID='article_456', got '%s'", event.SubjectID)
		}
		if event.ActorID != "user_789" {
			t.Errorf("Expected ActorID='user_789', got '%s'", event.ActorID)
		}
		if event.Source != "mobile_app" {
			t.Errorf("Expected Source='mobile_app', got '%s'", event.Source)
		}
		if event.Context == nil {
			t.Error("Expected Context to be set")
		} else {
			platform, ok := event.Context["platform"].(string)
			if !ok || platform != "ios" {
				t.Errorf("Expected Context['platform']='ios', got '%v'", platform)
			}
			appVersion, ok := event.Context["app_version"].(string)
			if !ok || appVersion != "2.0.0" {
				t.Errorf("Expected Context['app_version']='2.0.0', got '%v'", appVersion)
			}
		}
		// Sport fields should be empty
		if event.MatchID != "" {
			t.Errorf("Expected empty MatchID, got '%s'", event.MatchID)
		}
		if event.PlayerID != "" {
			t.Errorf("Expected empty PlayerID, got '%s'", event.PlayerID)
		}
	})

	// Test 2: Sport event with all sport fields
	t.Run("sport event with all sport fields", func(t *testing.T) {
		eventJSON := map[string]any{
			"event_type": "goal",
			"event_id":   "evt_goal_123",
			"match_id":   "match_456",
			"team_id":    "team_a",
			"player_id":  "player_789",
			"minute":     45.0,
		}

		event := parseMatchEvent(eventJSON)

		// Assert all sport fields
		if event.EventID != "evt_goal_123" {
			t.Errorf("Expected EventID='evt_goal_123', got '%s'", event.EventID)
		}
		if event.EventType != "goal" {
			t.Errorf("Expected EventType='goal', got '%s'", event.EventType)
		}
		if event.MatchID != "match_456" {
			t.Errorf("Expected MatchID='match_456', got '%s'", event.MatchID)
		}
		if event.TeamID != "team_a" {
			t.Errorf("Expected TeamID='team_a', got '%s'", event.TeamID)
		}
		if event.PlayerID != "player_789" {
			t.Errorf("Expected PlayerID='player_789', got '%s'", event.PlayerID)
		}
		if event.Minute != 45 {
			t.Errorf("Expected Minute=45, got %d", event.Minute)
		}
		// Generic fields should be empty
		if event.SubjectID != "" {
			t.Errorf("Expected empty SubjectID, got '%s'", event.SubjectID)
		}
		if event.Source != "" {
			t.Errorf("Expected empty Source, got '%s'", event.Source)
		}
	})

	// Test 3: Mixed event - both sport and generic fields
	t.Run("mixed event with both sport and generic fields", func(t *testing.T) {
		eventJSON := map[string]any{
			"event_type": "goal",
			"event_id":   "evt_goal_123",
			"match_id":   "match_456",
			"player_id":  "player_789",
			"team_id":    "team_a",
			"minute":     67.0,
			"source":     "match_feed",
			"context": map[string]any{
				"goal_type":      "open_play",
				"deflection":    true,
			},
			"metadata": map[string]any{
				"goal_type": "penalty",
			},
		}

		event := parseMatchEvent(eventJSON)

		// Assert sport fields
		if event.MatchID != "match_456" {
			t.Errorf("Expected MatchID='match_456', got '%s'", event.MatchID)
		}
		if event.PlayerID != "player_789" {
			t.Errorf("Expected PlayerID='player_789', got '%s'", event.PlayerID)
		}
		if event.TeamID != "team_a" {
			t.Errorf("Expected TeamID='team_a', got '%s'", event.TeamID)
		}
		if event.Minute != 67 {
			t.Errorf("Expected Minute=67, got %d", event.Minute)
		}

		// Assert generic fields
		if event.Source != "match_feed" {
			t.Errorf("Expected Source='match_feed', got '%s'", event.Source)
		}
		if event.Context == nil {
			t.Error("Expected Context to be set")
		} else {
			goalType, ok := event.Context["goal_type"].(string)
			if !ok || goalType != "open_play" {
				t.Errorf("Expected Context['goal_type']='open_play', got '%v'", goalType)
			}
		}

		// Assert metadata is converted to bytes
		if event.Metadata == nil {
			t.Error("Expected Metadata to be set")
		}
		// Verify metadata JSON content
		var metadataMap map[string]any
		if err := json.Unmarshal(event.Metadata, &metadataMap); err != nil {
			t.Errorf("Failed to parse metadata: %v", err)
		} else {
			if metadataMap["goal_type"] != "penalty" {
				t.Errorf("Expected metadata['goal_type']='penalty', got '%v'", metadataMap["goal_type"])
			}
		}
	})

	// Test 4: Empty event type - should still parse
	t.Run("empty event type generates auto ID", func(t *testing.T) {
		eventJSON := map[string]any{
			"subject_id": "user_123",
		}

		event := parseMatchEvent(eventJSON)

		// Event ID should be auto-generated
		if event.EventID == "" {
			t.Error("Expected auto-generated event ID")
		}
		if !strings.HasPrefix(event.EventID, "test_") {
			t.Errorf("Expected event ID to start with 'test_', got '%s'", event.EventID)
		}
		// Subject ID should be set
		if event.SubjectID != "user_123" {
			t.Errorf("Expected SubjectID='user_123', got '%s'", event.SubjectID)
		}
	})

	// Test 5: Missing optional fields - should not panic
	t.Run("missing optional fields does not panic", func(t *testing.T) {
		eventJSON := map[string]any{
			"event_type": "daily_login",
		}

		// Should not panic
		event := parseMatchEvent(eventJSON)

		// Basic fields should be set
		if event.EventType != "daily_login" {
			t.Errorf("Expected EventType='daily_login', got '%s'", event.EventType)
		}
		// Other fields should be empty
		if event.EventID == "" {
			t.Error("Expected auto-generated event ID")
		}
	})
}

// TestServiceMethods tests that all service methods have proper signatures
func TestServiceMethods(t *testing.T) {
	_ = []interface{}{
		func() {
			svc := &Service{}
			var ctx context.Context
			_, _ = svc.ListRules(ctx, "")
		},
		func() {
			svc := &Service{}
			var ctx context.Context
			_, _ = svc.GetRule(ctx, "rule-1")
		},
		func() {
			svc := &Service{}
			var ctx context.Context
			_, _ = svc.TestEvent(ctx, map[string]any{}, true)
		},
		func() {
			svc := &Service{}
			var ctx context.Context
			_ = svc.AssignBadgeToUser(ctx, "user-1", "badge-1", "test")
		},
		func() {
			svc := &Service{}
			var ctx context.Context
			_, _ = svc.UpdateUserPoints(ctx, "user-1", 100, "add")
		},
		func() {
			svc := &Service{}
			var ctx context.Context
			_, _ = svc.ListUsers(ctx, 10, 0)
		},
		func() {
			svc := &Service{}
			var ctx context.Context
			_, _ = svc.GetUserProfile(ctx, "user-1")
		},
		func() {
			svc := &Service{}
			var ctx context.Context
			_, _ = svc.GetAnalyticsSummary(ctx)
		},
		func() {
			svc := &Service{}
			var ctx context.Context
			_, _ = svc.ListEventTypes(ctx)
		},
		func() {
			svc := &Service{}
			var ctx context.Context
			_ = svc.IsSportEvent(ctx, "goal")
		},
	}
}
