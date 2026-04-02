package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"gamification/config"
	"gamification/models"
	"gamification/neo4j"
	"gamification/redis"
)

// RuleEngine is the core rule processing engine
type RuleEngine struct {
	redisClient *redis.Client
	neo4jClient *neo4j.Client
	config      *config.Config
	workers     int
	eventChan   chan *models.MatchEvent
	wg          sync.WaitGroup
	mu          sync.RWMutex
	running     bool
	// Reward layer for executing rewards
	rewardLayer *RewardLayer
	// RuleMatcher for processing conditions
	ruleMatcher *RuleMatcher
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine(cfg *config.Config, redisClient *redis.Client, neo4jClient *neo4j.Client) *RuleEngine {
	return &RuleEngine{
		redisClient: redisClient,
		neo4jClient: neo4jClient,
		config:      cfg,
		workers:     cfg.Engine.WorkerPoolSize,
		eventChan:   make(chan *models.MatchEvent, cfg.Engine.EventBufferSize),
		running:     false,
		ruleMatcher: NewRuleMatcher(redisClient),
	}
}

// SetRewardLayer sets the reward layer for the engine
func (e *RuleEngine) SetRewardLayer(rewardLayer *RewardLayer) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rewardLayer = rewardLayer
}

// Start starts the rule engine workers
func (e *RuleEngine) Start(ctx context.Context) {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return
	}
	e.running = true
	e.mu.Unlock()

	log.Printf("Starting rule engine with %d workers...", e.workers)

	for i := 0; i < e.workers; i++ {
		e.wg.Add(1)
		go e.worker(ctx, i)
	}
}

// Stop stops the rule engine
func (e *RuleEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return
	}

	close(e.eventChan)
	e.wg.Wait()
	e.running = false
	log.Println("Rule engine stopped")
}

// ProcessMatchEvent processes a match event through the rule engine
// This is the main entry point for event processing
// Set dryRun=true to evaluate without executing actions or writing to storage
func (e *RuleEngine) ProcessMatchEvent(ctx context.Context, event *models.MatchEvent, dryRun bool) *models.RuleEngineResult {
	startTime := time.Now()
	result := &models.RuleEngineResult{
		Event:          event,
		TriggeredRules: []models.RuleEvaluationResult{},
		Success:        true,
	}

	// Check if event type is enabled in the registry
	et, err := e.redisClient.GetEventType(ctx, string(event.EventType))
	if err != nil {
		log.Printf("Warning: failed to get event type from registry: %v", err)
		// Continue processing - don't fail on registry errors
	} else if et != nil && !et.Enabled {
		// Event type is disabled - skip processing
		result.Success = true
		result.Error = fmt.Errorf("event type '%s' is disabled", event.EventType)
		result.TotalTimeMs = float64(time.Since(startTime).Milliseconds())
		result.Skipped = true
		result.SkipReason = "event_type_disabled"
		return result
	}

	// Store event in Redis for history (skip in dry-run mode)
	if !dryRun {
		if err := e.redisClient.StoreMatchEvent(ctx, event); err != nil {
			log.Printf("Failed to store event: %v", err)
		}
		
		// Increment generic event counts for aggregations
		if event.PlayerID != "" {
			if event.MatchID != "" {
				_, _ = e.redisClient.IncrementEventCount(ctx, event.MatchID, event.PlayerID, event.EventType)
			}
			_, _ = e.redisClient.IncrementGlobalEventCount(ctx, event.PlayerID, event.EventType)
			_, _ = e.redisClient.UpdateDailyStreak(ctx, event.PlayerID, event.EventType)
		}
	}

	// Find matching rules from Redis
	var rules []models.Rule
	err = nil
	if dryRun {
		rules, err = e.MatchRulesNoSideEffects(ctx, event)
	} else {
		rules, err = e.MatchRules(ctx, event)
	}
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("failed to match rules: %w", err)
		result.TotalTimeMs = float64(time.Since(startTime).Milliseconds())
		return result
	}

	// Evaluate each matching rule
	for _, rule := range rules {
		var evalResult *models.RuleEvaluationResult
		if dryRun {
			evalResult = e.EvaluateRulesNoSideEffects(ctx, event, &rule)
		} else {
			evalResult = e.evaluateRule(ctx, event, &rule)
		}
		result.TriggeredRules = append(result.TriggeredRules, *evalResult)

		// Execute actions if rule matched (skip in dry-run mode)
		if evalResult.Matched && !dryRun {
			e.executeActions(ctx, event, evalResult)
		}
	}

	result.TotalTimeMs = float64(time.Since(startTime).Milliseconds())

	// Log performance
	if result.TotalTimeMs > 10 {
		log.Printf("Warning: Rule processing took %.2fms (target: <10ms)", result.TotalTimeMs)
	}

	if !dryRun {
		if err := e.redisClient.LogEventEvaluation(ctx, result); err != nil {
			log.Printf("Warning: failed to log event evaluation: %v", err)
		}
	}

	return result
}

// SubmitEvent submits an event for async processing
func (e *RuleEngine) SubmitEvent(event *models.MatchEvent) {
	select {
	case e.eventChan <- event:
	default:
		log.Printf("Warning: Event channel full, dropping event %s", event.EventID)
	}
}

// MatchRules finds rules that match the given event (no side effects)
func (e *RuleEngine) MatchRules(ctx context.Context, event *models.MatchEvent) ([]models.Rule, error) {
	// Get rules from Redis for this event type
	rules, err := e.redisClient.GetRulesByEventType(ctx, event.EventType)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	// Filter and evaluate conditions
	matchedRules := make([]models.Rule, 0)
	for _, rule := range rules {
		if e.evaluateConditions(ctx, event, &rule) {
			matchedRules = append(matchedRules, rule)
		}
	}

	return matchedRules, nil
}

// MatchRulesNoSideEffects finds rules that match without any side effects (dry-run)
// This does NOT write to Redis or Neo4j
func (e *RuleEngine) MatchRulesNoSideEffects(ctx context.Context, event *models.MatchEvent) ([]models.Rule, error) {
	// Get rules from Redis for this event type
	rules, err := e.redisClient.GetRulesByEventType(ctx, event.EventType)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	// Filter and evaluate conditions (no side effects)
	matchedRules := make([]models.Rule, 0)
	for _, rule := range rules {
		if e.evaluateConditions(ctx, event, &rule) {
			matchedRules = append(matchedRules, rule)
		}
	}

	return matchedRules, nil
}

// evaluateConditions evaluates all conditions for a rule
func (e *RuleEngine) evaluateConditions(ctx context.Context, event *models.MatchEvent, rule *models.Rule) bool {
	for _, condition := range rule.Conditions {
		if !e.evaluateCondition(ctx, event, rule, &condition) {
			return false
		}
	}
	return true
}

// evaluateCondition evaluates a single condition by delegating to the comprehensive ruleMatcher
func (e *RuleEngine) evaluateCondition(ctx context.Context, event *models.MatchEvent, rule *models.Rule, condition *models.RuleCondition) bool {
	switch condition.EvaluationType {
	case "simple":
		return e.ruleMatcher.MatchSimpleCondition(*rule, *event, *condition)
	case "aggregation":
		matched, err := e.ruleMatcher.MatchAggregationCondition(ctx, *rule, *event, *condition)
		if err != nil {
			log.Printf("Error evaluating aggregation condition for rule %s: %v", rule.Name, err)
			return false
		}
		return matched
	case "temporal":
		matched, err := e.ruleMatcher.MatchTemporalCondition(ctx, *rule, *event, *condition)
		if err != nil {
			log.Printf("Error evaluating temporal condition for rule %s: %v", rule.Name, err)
			return false
		}
		return matched
	default:
		log.Printf("Unknown evaluation type: %s", condition.EvaluationType)
		return false
	}
}

// evaluateRule evaluates a complete rule and returns the result
func (e *RuleEngine) evaluateRule(ctx context.Context, event *models.MatchEvent, rule *models.Rule) *models.RuleEvaluationResult {
	startTime := time.Now()
	result := &models.RuleEvaluationResult{
		Rule:    rule,
		Matched: false,
	}

	// Check cooldown
	inCooldown, err := e.redisClient.CheckCooldown(ctx, rule.RuleID, event.MatchID, event.PlayerID)
	if err == nil && inCooldown {
		result.EvalTimeMs = float64(time.Since(startTime).Milliseconds())
		return result
	}

	// Check conditions
	if !e.evaluateConditions(ctx, event, rule) {
		result.EvalTimeMs = float64(time.Since(startTime).Milliseconds())
		return result
	}

	result.Matched = true

	// Fetch affected users from Neo4j
	users, err := e.QueryAffectedUsers(ctx, event, rule)
	if err != nil {
		log.Printf("Failed to query affected users: %v", err)
		result.Users = []string{event.PlayerID} // Fallback to event player
	} else {
		result.Users = users
	}

	result.Actions = rule.Actions

	// Set cooldown if configured
	if rule.CooldownSeconds > 0 {
		e.redisClient.SetCooldown(ctx, rule.RuleID, event.MatchID, event.PlayerID, time.Duration(rule.CooldownSeconds)*time.Second)
	}

	result.EvalTimeMs = float64(time.Since(startTime).Milliseconds())
	return result
}

// EvaluateRulesNoSideEffects evaluates a rule without any side effects (dry-run)
// Does NOT set cooldowns in Redis or execute actions, but still queries Neo4j for user targeting
func (e *RuleEngine) EvaluateRulesNoSideEffects(ctx context.Context, event *models.MatchEvent, rule *models.Rule) *models.RuleEvaluationResult {
	startTime := time.Now()
	result := &models.RuleEvaluationResult{
		Rule:    rule,
		Matched: false,
	}

	// Check cooldown (read-only, no side effect)
	inCooldown, err := e.redisClient.CheckCooldown(ctx, rule.RuleID, event.MatchID, event.PlayerID)
	if err == nil && inCooldown {
		result.EvalTimeMs = float64(time.Since(startTime).Milliseconds())
		return result
	}

	// Check conditions (no side effects)
	if !e.evaluateConditions(ctx, event, rule) {
		result.EvalTimeMs = float64(time.Since(startTime).Milliseconds())
		return result
	}

	result.Matched = true

	// Query Neo4j for affected users (targeting) - this is safe for dry-run as it only reads
	users, err := e.QueryAffectedUsers(ctx, event, rule)
	if err != nil {
		log.Printf("Failed to query affected users in dry-run: %v", err)
		result.Users = []string{event.PlayerID} // Fallback to event player
	} else {
		result.Users = users
	}

	result.Actions = rule.Actions

	// NO cooldown set in dry-run mode

	result.EvalTimeMs = float64(time.Since(startTime).Milliseconds())
	return result
}

func (e *RuleEngine) QueryAffectedUsers(ctx context.Context, event *models.MatchEvent, rule *models.Rule) ([]string, error) {
	if rule.TargetUsers.QueryPattern == "" {
		return []string{event.PlayerID}, nil
	}

	params := make(map[string]string)
	for k, v := range rule.TargetUsers.Params {
		params[k] = v
	}

	result, err := e.neo4jClient.QueryAffectedUsers(
		ctx,
		event.MatchID,
		event.TeamID,
		event.PlayerID,
		rule.TargetUsers.QueryPattern,
		params,
	)
	if err != nil {
		return nil, err
	}

	return result.UserIDs, nil
}

// executeActions executes the actions for a matched rule
func (e *RuleEngine) executeActions(ctx context.Context, event *models.MatchEvent, result *models.RuleEvaluationResult) {
	for _, userID := range result.Users {
		for _, action := range result.Actions {
			err := e.executeAction(ctx, userID, result.Rule.RuleID, event, &action)
			if err != nil {
				log.Printf("Failed to execute action for user %s: %v", userID, err)
				continue // Don't record action if it failed - allows retry
			}
			e.neo4jClient.RecordUserAction(ctx, userID, action.ActionType, event.MatchID, event.EventID)
		}
	}
}

// ExecuteActions is a public wrapper for executing actions (used by API)
func (e *RuleEngine) ExecuteActions(ctx context.Context, event *models.MatchEvent, result *models.RuleEvaluationResult) error {
	e.executeActions(ctx, event, result)
	return nil
}

// executeAction executes a single action and returns error if failed
func (e *RuleEngine) executeAction(ctx context.Context, userID, ruleID string, event *models.MatchEvent, action *models.RuleAction) error {
	// Use reward layer if available for reward actions
	if e.rewardLayer != nil {
		return e.rewardLayer.ExecuteRewardAction(ctx, userID, ruleID, action, event)
	}

	// Fallback to direct logging if no reward layer
	switch action.ActionType {
	case "award_points":
		points, _ := action.Params["points"].(float64)
		log.Printf("Awarding %.0f points to user %s (event: %s)", points, userID, event.EventID)
	case "grant_badge":
		badgeID, _ := action.Params["badge_id"].(string)
		badgeName, _ := action.Params["badge_name"].(string)
		log.Printf("Granting badge %s (%s) to user %s (event: %s)", badgeID, badgeName, userID, event.EventID)
	case "send_notification":
		notificationType, _ := action.Params["type"].(string)
		log.Printf("Sending %s notification to user %s", notificationType, userID)
	}
	return nil
}

// worker is the worker goroutine that processes events
func (e *RuleEngine) worker(ctx context.Context, id int) {
	defer e.wg.Done()
	log.Printf("Worker %d started", id)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopping", id)
			return
		case event, ok := <-e.eventChan:
			if !ok {
				log.Printf("Worker %d stopping", id)
				return
			}
			e.ProcessMatchEvent(ctx, event, false)
		}
	}
}

// compareValues compares two values using the given operator
func compareValues(fieldValue any, operator string, targetValue any) bool {
	switch operator {
	case "==":
		return areEqual(fieldValue, targetValue)
	case "!=":
		return !areEqual(fieldValue, targetValue)
	case ">":
		return toFloat(fieldValue) > toFloat(targetValue)
	case ">=":
		return toFloat(fieldValue) >= toFloat(targetValue)
	case "<":
		return toFloat(fieldValue) < toFloat(targetValue)
	case "<=":
		return toFloat(fieldValue) <= toFloat(targetValue)
	case "in":
		if arr, ok := targetValue.([]any); ok {
			for _, v := range arr {
				if areEqual(fieldValue, v) {
					return true
				}
			}
		}
		return false
	case "every":
		targetFloat := toFloat(targetValue)
		countFloat := toFloat(fieldValue)
		if targetFloat <= 0 {
			return false
		}
		// True if count > 0 and count is a perfect multiple of target
		return countFloat > 0 && int64(countFloat)%int64(targetFloat) == 0
	default:
		return false
	}
}

// areEqual compares two values for equality, handling numeric type differences
func areEqual(a, b any) bool {
	// First try direct comparison (handles same types)
	if a == b {
		return true
	}

	// Convert both to float64 and compare numeric values
	aFloat := toFloat(a)
	bFloat := toFloat(b)

	// If both converted to non-zero values, compare as floats
	// If both are 0.0, they might not be numeric at all (e.g., strings)
	if aFloat != 0 || bFloat != 0 {
		return aFloat == bFloat
	}

	// Fallback: both converted to 0, try string comparison
	aStr, aOk := toString(a)
	bStr, bOk := toString(b)
	if aOk && bOk {
		return aStr == bStr
	}

	return false
}

// toString converts a value to string
func toString(v any) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case int:
		return fmt.Sprintf("%d", val), true
	case int8:
		return fmt.Sprintf("%d", val), true
	case int16:
		return fmt.Sprintf("%d", val), true
	case int32:
		return fmt.Sprintf("%d", val), true
	case int64:
		return fmt.Sprintf("%d", val), true
	case uint:
		return fmt.Sprintf("%d", val), true
	case uint8:
		return fmt.Sprintf("%d", val), true
	case uint16:
		return fmt.Sprintf("%d", val), true
	case uint32:
		return fmt.Sprintf("%d", val), true
	case uint64:
		return fmt.Sprintf("%d", val), true
	case float32:
		return fmt.Sprintf("%f", val), true
	case float64:
		return fmt.Sprintf("%f", val), true
	default:
		return "", false
	}
}

// toFloat converts a value to float64
func toFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int8:
		return float64(val)
	case int16:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case uint:
		return float64(val)
	case uint8:
		return float64(val)
	case uint16:
		return float64(val)
	case uint32:
		return float64(val)
	case uint64:
		return float64(val)
	case json.Number:
		f, _ := val.Float64()
		return f
	default:
		return 0
	}
}
