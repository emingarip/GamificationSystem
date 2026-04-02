package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gamification/models"
	"gamification/redis"
)

// RuleMatcher handles the matching of events against active rules.
// It provides high-performance rule matching by caching rules in memory
// and using efficient condition evaluation.
//
// The matcher supports three types of rule conditions:
// 1. Simple: Direct comparisons (event.type == rule.eventType AND event.value >= rule.threshold)
// 2. Aggregation: Count-based conditions (e.g., "3 corners in last 5 minutes")
// 3. Temporal: Time-window based conditions (e.g., "3 goals in first half")
type RuleMatcher struct {
	redisClient *redis.Client
	// In-memory cache of rules indexed by event type
	// Key: event type (goal, corner, foul, etc.)
	// Value: slice of active rules for that event type
	rulesCache  map[models.EventType][]models.Rule
	rulesMu     sync.RWMutex
	cacheTTL    time.Duration
	lastRefresh time.Time
}

// NewRuleMatcher creates a new rule matcher with Redis backend
func NewRuleMatcher(redisClient *redis.Client) *RuleMatcher {
	return &RuleMatcher{
		redisClient: redisClient,
		rulesCache:  make(map[models.EventType][]models.Rule),
		cacheTTL:    5 * time.Minute, // Refresh cache every 5 minutes
	}
}

// FindMatchingRules finds all rules that match the given event.
// It first checks the in-memory cache, then evaluates each rule's conditions.
func (rm *RuleMatcher) FindMatchingRules(ctx context.Context, event models.MatchEvent) ([]models.Rule, error) {
	// Refresh cache if needed
	if rm.needsRefresh() {
		if err := rm.refreshCache(ctx); err != nil {
			log.Printf("Warning: failed to refresh rule cache: %v", err)
			// Continue with stale cache rather than failing
		}
	}

	// Get rules for this event type
	rm.rulesMu.RLock()
	rules, ok := rm.rulesCache[event.EventType]
	rm.rulesMu.RUnlock()

	if !ok || len(rules) == 0 {
		log.Printf("No rules found for event type: %s", event.EventType)
		return nil, nil
	}

	// Evaluate each rule against the event
	var matchedRules []models.Rule
	for _, rule := range rules {
		if rm.EvaluateRule(ctx, rule, event) {
			matchedRules = append(matchedRules, rule)
		}
	}

	// Sort matched rules by priority (higher priority first)
	if len(matchedRules) > 1 {
		sortRulesByPriority(matchedRules)
	}

	return matchedRules, nil
}

// needsRefresh checks if the cache needs to be refreshed
func (rm *RuleMatcher) needsRefresh() bool {
	rm.rulesMu.RLock()
	defer rm.rulesMu.RUnlock()
	return time.Since(rm.lastRefresh) > rm.cacheTTL || len(rm.rulesCache) == 0
}

// refreshCache reloads all active rules from Redis into memory
func (rm *RuleMatcher) refreshCache(ctx context.Context) error {
	log.Println("Refreshing rule cache from Redis...")

	// Get event types from registry - this is the primary source
	// Registry is seeded at startup, so it should always have values
	eventTypes, err := rm.getEventTypesForCache(ctx)
	if err != nil {
		// This should not happen if startup seed worked correctly
		// But if it does, we log an error and return - no fallback
		log.Printf("ERROR: No event types found in registry. This indicates startup seed may have failed: %v", err)
		return fmt.Errorf("event type registry empty: %w", err)
	}

	rm.rulesMu.Lock()
	defer rm.rulesMu.Unlock()

	// Clear the entire cache before refreshing to ensure clean state
	// This removes rules that are no longer in Redis
	rm.rulesCache = make(map[models.EventType][]models.Rule)

	for _, eventType := range eventTypes {
		rules, err := rm.redisClient.GetRulesByEventType(ctx, models.EventType(eventType))
		if err != nil {
			log.Printf("Warning: failed to get rules for event type %s: %v", eventType, err)
			continue
		}
		rm.rulesCache[models.EventType(eventType)] = rules
	}

	rm.lastRefresh = time.Now()
	log.Printf("Rule cache refreshed: %d event types loaded from registry", len(rm.rulesCache))

	return nil
}

// getEventTypesForCache gets event types from the registry
func (rm *RuleMatcher) getEventTypesForCache(ctx context.Context) ([]string, error) {
	// Try to get from event type registry
	eventTypes, err := rm.redisClient.GetEnabledEventTypes(ctx)
	if err != nil {
		return nil, err
	}

	if len(eventTypes) == 0 {
		return nil, fmt.Errorf("no event types in registry")
	}

	keys := make([]string, len(eventTypes))
	for i, et := range eventTypes {
		keys[i] = et.Key
	}
	return keys, nil
}

// EvaluateRule evaluates all conditions in a rule against the event.
// Returns true if all conditions match.
func (rm *RuleMatcher) EvaluateRule(ctx context.Context, rule models.Rule, event models.MatchEvent) bool {
	if len(rule.Conditions) == 0 {
		// No conditions means the rule matches any event of the correct type
		return event.EventType == rule.EventType
	}

	// Evaluate each condition
	for _, condition := range rule.Conditions {
		var matched bool
		var err error

		switch condition.EvaluationType {
		case "simple":
			matched = rm.MatchSimpleCondition(rule, event, condition)
		case "aggregation":
			matched, err = rm.MatchAggregationCondition(ctx, rule, event, condition)
		case "temporal":
			matched, err = rm.MatchTemporalCondition(ctx, rule, event, condition)
		default:
			log.Printf("Unknown evaluation type: %s", condition.EvaluationType)
			return false
		}

		if err != nil {
			log.Printf("Error evaluating condition: %v", err)
			return false
		}

		if !matched {
			return false
		}
	}

	return true
}

// LoadActiveRulesFromRedis loads all active rules from Redis and organizes them by event type.
// This is called during initialization and periodically refreshed.
//
// Returns a map of event type to list of active rules.
// NOTE: This function is deprecated - use FindMatchingRules which uses the registry.
func (rm *RuleMatcher) LoadActiveRulesFromRedis(ctx context.Context) (map[models.EventType][]models.Rule, error) {
	// Get event types from registry (the proper source)
	eventTypes, err := rm.redisClient.GetEnabledEventTypes(ctx)
	if err != nil || len(eventTypes) == 0 {
		// Fallback: try to list all event types
		eventTypes, err = rm.redisClient.ListEventTypes(ctx)
		if err != nil || len(eventTypes) == 0 {
			return nil, fmt.Errorf("no event types in registry - startup seed may have failed")
		}
	}

	result := make(map[models.EventType][]models.Rule)

	for _, et := range eventTypes {
		rules, err := rm.redisClient.GetRulesByEventType(ctx, models.EventType(et.Key))
		if err != nil {
			log.Printf("Warning: failed to get rules for event type %s: %v", et.Key, err)
			continue
		}
		result[models.EventType(et.Key)] = rules
	}

	return result, nil
}

// MatchSimpleCondition evaluates a simple condition against an event.
// Simple conditions are direct comparisons that don't require aggregation or time windows.
//
// Examples:
// - event.type == "goal" AND event.minute >= 45
// - event.team_id == "team_abc" AND event.player_id == "player_xyz"
//
// The condition should have:
// - Field: the event field to compare (e.g., "minute", "team_id")
// - Operator: comparison operator (e.g., ">=", "==", "<")
// - Value: the value to compare against
func (rm *RuleMatcher) MatchSimpleCondition(rule models.Rule, event models.MatchEvent, condition models.RuleCondition) bool {
	// Get the value from the event based on the field
	var eventValue any
	switch condition.Field {
	case "event_type":
		eventValue = string(event.EventType)
	case "match_id":
		eventValue = event.MatchID
	case "team_id":
		eventValue = event.TeamID
	case "player_id":
		eventValue = event.PlayerID
	case "minute":
		eventValue = event.Minute
	case "timestamp":
		eventValue = event.Timestamp.Unix()
	default:
		// Try to get from metadata if present
		log.Printf("Unknown field in simple condition: %s", condition.Field)
		return false
	}

	// Compare based on operator - use existing compareValues from engine.go
	return compareValues(eventValue, condition.Operator, condition.Value)
}

// MatchAggregationCondition evaluates an aggregation condition.
// Aggregation conditions count events over a time window.
//
// Examples:
// - "3 corners in last 5 minutes" (consecutive_count >= 3)
// - "2 goals in the match" (total_goals >= 2)
// - "5 fouls in first half" (total_fouls >= 5)
//
// The condition should have:
// - Field: the aggregation field (e.g., "consecutive_count", "total_goals")
// - Operator: comparison operator (e.g., ">=", "==")
// - Value: the threshold value
// - Params: optional parameters like "window_minutes"
func (rm *RuleMatcher) MatchAggregationCondition(ctx context.Context, rule models.Rule, event models.MatchEvent, condition models.RuleCondition) (bool, error) {
	// Get the current count from Redis
	var count int64
	var err error

	switch condition.Field {
	case "consecutive_count":
		// Count of consecutive events of this type for the player in this match
		count, err = rm.redisClient.GetEventCount(ctx, event.MatchID, event.PlayerID, event.EventType)
		if err != nil {
			return false, fmt.Errorf("failed to get event count: %w", err)
		}
	case "global_count":
		// Count of total events of this type for the player globally (lifetime)
		count, err = rm.redisClient.GetGlobalEventCount(ctx, event.PlayerID, event.EventType)
		log.Printf("DEBUG: Evaluated global_count for %s:%s = %d (Rule Target: %v)", event.PlayerID, event.EventType, count, condition.Value)
		if err != nil {
			return false, fmt.Errorf("failed to get global event count: %w", err)
		}
	case "daily_streak":
		// Sequence of days in a row the player performed this event
		count, err = rm.redisClient.GetDailyStreak(ctx, event.PlayerID, event.EventType)
		if err != nil {
			return false, fmt.Errorf("failed to get daily streak: %w", err)
		}
	case "total_goals":
		// Total goals in the match (requires separate tracking)
		// For now, this would need additional Redis keys
		log.Println("total_goals aggregation not fully implemented")
		return false, nil
	default:
		log.Printf("Unknown aggregation field: %s", condition.Field)
		return false, nil
	}

	// Compare the count against the threshold - use existing compareValues from engine.go
	return compareValues(count, condition.Operator, condition.Value), nil
}

// MatchTemporalCondition evaluates a temporal condition.
// Temporal conditions check for patterns within specific time windows.
//
// Examples:
// - "3 corners in last 5 minutes" (temporal window check)
// - "First goal of the match" (temporal - first occurrence)
// - "Goal in stoppage time" (temporal - minute range)
//
// The condition should have:
// - Field: the temporal field (e.g., "window_events", "first_occurrence")
// - Operator: comparison operator
// - Value: the threshold or target value
// - Params: window configuration like "window_minutes"
func (rm *RuleMatcher) MatchTemporalCondition(ctx context.Context, rule models.Rule, event models.MatchEvent, condition models.RuleCondition) (bool, error) {
	// Get window parameters from condition params
	windowMinutes := 5 // Default 5 minutes
	if params, ok := condition.Value.(map[string]any); ok {
		if w, ok := params["window_minutes"].(float64); ok {
			windowMinutes = int(w)
		}
	}

	// For temporal conditions, we need to check event history
	// This is a simplified implementation - full version would query event history
	switch condition.Field {
	case "window_events":
		// Count events in the time window
		count, err := rm.countEventsInWindow(ctx, event, windowMinutes)
		if err != nil {
			return false, err
		}
		return compareValues(count, condition.Operator, condition.Value), nil

	case "first_occurrence":
		// Check if this is the first event of this type in the match
		count, err := rm.redisClient.GetEventCount(ctx, event.MatchID, event.PlayerID, event.EventType)
		if err != nil {
			return false, err
		}
		// First occurrence means count == 1 (before incrementing)
		return count == 1, nil

	case "minute_range":
		// Check if event minute is within a range
		params, ok := condition.Value.(map[string]any)
		if !ok {
			return false, fmt.Errorf("invalid minute_range params: %v", condition.Value)
		}
		minMinute, _ := params["min"].(float64)
		maxMinute, _ := params["max"].(float64)
		return event.Minute >= int(minMinute) && event.Minute <= int(maxMinute), nil

	default:
		log.Printf("Unknown temporal field: %s", condition.Field)
		return false, nil
	}
}

// countEventsInWindow counts events within a time window
func (rm *RuleMatcher) countEventsInWindow(ctx context.Context, event models.MatchEvent, windowMinutes int) (int64, error) {
	// For now, this returns the current count which represents events up to now
	// A full implementation would query event history with time filtering
	count, err := rm.redisClient.GetEventCount(ctx, event.MatchID, event.PlayerID, event.EventType)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetRuleByID retrieves a single rule by ID from Redis
func (rm *RuleMatcher) GetRuleByID(ctx context.Context, ruleID string) (*models.Rule, error) {
	return rm.redisClient.GetRuleByID(ctx, ruleID)
}

// InvalidateCache clears the in-memory rule cache
func (rm *RuleMatcher) InvalidateCache() {
	rm.rulesMu.Lock()
	defer rm.rulesMu.Unlock()
	rm.rulesCache = make(map[models.EventType][]models.Rule)
	log.Println("Rule cache invalidated")
}

// sortRulesByPriority sorts rules by priority (higher first)
func sortRulesByPriority(rules []models.Rule) {
	for i := 0; i < len(rules)-1; i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i].Priority < rules[j].Priority {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}
}
