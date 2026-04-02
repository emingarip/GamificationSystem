// Package backend provides a simple backend service for the MCP server
package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gamification/engine"
	"gamification/models"
	"gamification/neo4j"
	"gamification/redis"
)

// Service represents a backend service for MCP operations
type Service struct {
	redis       *redis.Client
	neo4j       *neo4j.Client
	ruleEngine  *engine.RuleEngine
	rewardLayer *engine.RewardLayer
}

// NewService creates a new backend service
func NewService(redisClient *redis.Client, neo4jClient *neo4j.Client) *Service {
	s := &Service{
		redis: redisClient,
		neo4j: neo4jClient,
	}
	return s
}

// SetRuleEngine sets the rule engine for the service
func (s *Service) SetRuleEngine(ruleEngine *engine.RuleEngine) {
	s.ruleEngine = ruleEngine
}

// SetRewardLayer sets the reward layer for the service
func (s *Service) SetRewardLayer(rewardLayer *engine.RewardLayer) {
	s.rewardLayer = rewardLayer
}

// ListRules returns a list of rules from Redis
func (s *Service) ListRules(ctx context.Context, eventType string) ([]Rule, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("Redis client not available")
	}

	var rules []models.Rule
	var err error

	if eventType != "" {
		rules, err = s.redis.GetRulesByEventType(ctx, models.EventType(eventType))
	} else {
		// Get all active rules by using event type registry (dynamic, no hardcoded list)
		eventTypes, err := s.redis.GetEnabledEventTypes(ctx)
		if err != nil || len(eventTypes) == 0 {
			// Fallback: use event type list from Redis if registry is empty
			eventTypes, err = s.redis.ListEventTypes(ctx)
			if err != nil || len(eventTypes) == 0 {
				// Last resort: try to get all rules directly
				rules, err = s.redis.GetAllRules(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to get rules: %w", err)
				}
				goto convert
			}
		}

		seen := make(map[string]bool)
		for _, et := range eventTypes {
			rs, err := s.redis.GetRulesByEventType(ctx, models.EventType(et.Key))
			if err != nil {
				continue
			}
			for _, r := range rs {
				if !seen[r.RuleID] {
					seen[r.RuleID] = true
					rules = append(rules, r)
				}
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

convert:
	result := make([]Rule, len(rules))
	for i, r := range rules {
		// Extract points from actions
		points := 0
		for _, action := range r.Actions {
			if action.ActionType == "award_points" {
				if p, ok := action.Params["points"].(float64); ok {
					points = int(p)
					break
				}
			}
		}
		result[i] = Rule{
			ID:          r.RuleID,
			Name:        r.Name,
			EventType:   string(r.EventType),
			Points:      points,
			Enabled:     r.IsActive,
			Description: r.Description,
		}
	}

	return result, nil
}

// GetRule returns a rule by ID from Redis
func (s *Service) GetRule(ctx context.Context, ruleID string) (*Rule, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("Redis client not available")
	}

	rule, err := s.redis.GetRuleByID(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}
	if rule == nil {
		return nil, fmt.Errorf("rule not found: %s", ruleID)
	}

	// Extract points from actions
	points := 0
	for _, action := range rule.Actions {
		if action.ActionType == "award_points" {
			if p, ok := action.Params["points"].(float64); ok {
				points = int(p)
				break
			}
		}
	}

	return &Rule{
		ID:          rule.RuleID,
		Name:        rule.Name,
		EventType:   string(rule.EventType),
		Points:      points,
		Enabled:     rule.IsActive,
		Description: rule.Description,
	}, nil
}

// TestEvent tests an event using the rule engine
// IsSportEvent checks if an event type requires sport-specific fields (match_id, player_id)
// by querying the event type registry in Redis. Returns false for non-sport events or unknown event types.
// If registry is unavailable, defaults to false (safe assumption - don't require sport fields).
func (s *Service) IsSportEvent(ctx context.Context, eventType string) bool {
	if s.redis == nil {
		// No Redis - assume generic (safe default)
		return false
	}

	et, err := s.redis.GetEventType(ctx, eventType)
	if err != nil || et == nil {
		// Event type not found in registry - assume generic (safe assumption)
		return false
	}

	// Only sport category requires match_id and player_id
	return et.Category == "sport"
}

// parseMatchEvent parses event JSON into a MatchEvent model
// This helper is exported for testing purposes to enable production parser regression tests
func parseMatchEvent(eventJSON map[string]any) *models.MatchEvent {
	event := &models.MatchEvent{}

	// Parse event ID
	if v, ok := eventJSON["event_id"].(string); ok && v != "" {
		event.EventID = v
	}

	// Parse event type
	if v, ok := eventJSON["event_type"].(string); ok {
		event.EventType = models.EventType(v)
	}

	// Parse sport-specific fields
	if v, ok := eventJSON["match_id"].(string); ok && v != "" {
		event.MatchID = v
	}
	if v, ok := eventJSON["team_id"].(string); ok && v != "" {
		event.TeamID = v
	}
	if v, ok := eventJSON["player_id"].(string); ok && v != "" {
		event.PlayerID = v
	}
	if v, ok := eventJSON["minute"].(float64); ok {
		event.Minute = int(v)
	}

	// Parse metadata
	if v, ok := eventJSON["metadata"].(map[string]any); ok {
		metadataBytes, err := json.Marshal(v)
		if err == nil {
			event.Metadata = metadataBytes
		}
	}

	// Parse generic fields
	if v, ok := eventJSON["subject_id"].(string); ok && v != "" {
		event.SubjectID = v
	}
	if v, ok := eventJSON["actor_id"].(string); ok && v != "" {
		event.ActorID = v
	}
	if v, ok := eventJSON["source"].(string); ok && v != "" {
		event.Source = v
	}
	if v, ok := eventJSON["context"].(map[string]any); ok {
		event.Context = v
	}

	// Generate event ID if not provided
	if event.EventID == "" {
		event.EventID = fmt.Sprintf("test_%d", time.Now().UnixNano())
	}

	return event
}

func (s *Service) TestEvent(ctx context.Context, eventJSON map[string]any, dryRun bool) (map[string]any, error) {
	// Validate required fields: event is required
	if eventJSON == nil {
		return nil, fmt.Errorf("validation error: event is required")
	}

	eventType, _ := eventJSON["event_type"].(string)
	if eventType == "" {
		return nil, fmt.Errorf("validation error: event_type is required")
	}

	// For sport events (category=sport in registry), validate match_id and player_id
	if s.IsSportEvent(ctx, eventType) {
		if matchID, _ := eventJSON["match_id"].(string); matchID == "" {
			return nil, fmt.Errorf("validation error: match_id is required for event type '%s'", eventType)
		}
		if playerID, _ := eventJSON["player_id"].(string); playerID == "" {
			return nil, fmt.Errorf("validation error: player_id is required for event type '%s'", eventType)
		}
	}

	// Check rule engine availability AFTER payload validation
	if s.ruleEngine == nil {
		return nil, fmt.Errorf("Rule engine not available")
	}

	// Parse event from JSON using the helper
	event := parseMatchEvent(eventJSON)

	// Process the event
	result := s.ruleEngine.ProcessMatchEvent(ctx, event, dryRun)

	// Format the result
	response := map[string]any{
		"success":       result.Success,
		"event_id":      event.EventID,
		"event_type":    event.EventType,
		"dry_run":       dryRun,
		"total_time_ms": result.TotalTimeMs,
		"matched_rules": len(result.TriggeredRules),
	}

	if result.Skipped {
		response["skipped"] = true
		response["skip_reason"] = result.SkipReason
	}

	if result.Error != nil {
		response["error"] = result.Error.Error()
	}

	// Add triggered rules details
	rules := make([]map[string]any, 0)
	for _, r := range result.TriggeredRules {
		ruleInfo := map[string]any{
			"rule_id":      r.Rule.RuleID,
			"name":         r.Rule.Name,
			"matched":      r.Matched,
			"eval_time_ms": r.EvalTimeMs,
			"users":        r.Users,
			"actions":      len(r.Actions),
		}
		if r.Matched {
			ruleInfo["actions_detail"] = r.Actions
		}
		rules = append(rules, ruleInfo)
	}
	response["triggered_rules"] = rules

	return response, nil
}

// AssignBadgeToUser assigns a badge to a user using the reward layer
func (s *Service) AssignBadgeToUser(ctx context.Context, userID, badgeID, reason string) error {
	if s.rewardLayer == nil {
		return fmt.Errorf("Reward layer not available")
	}

	// Use a unique event ID for tracking
	eventID := fmt.Sprintf("mcp_assign_%d", time.Now().UnixNano())

	_, err := s.rewardLayer.GrantBadge(ctx, userID, badgeID, eventID, reason)
	if err != nil {
		return fmt.Errorf("failed to assign badge: %w", err)
	}

	log.Printf("MCP: Assigned badge %s to user %s", badgeID, userID)
	return nil
}

// UpdateUserPoints updates user points using the reward layer or Neo4j
func (s *Service) UpdateUserPoints(ctx context.Context, userID string, points int, operation string) (int, error) {
	if s.neo4j == nil {
		return 0, fmt.Errorf("Neo4j client not available")
	}

	// Update points in Neo4j
	err := s.neo4j.UpdateUserPoints(ctx, userID, points, operation)
	if err != nil {
		return 0, fmt.Errorf("failed to update user points: %w", err)
	}

	// Get updated points
	user, err := s.neo4j.GetUserByID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get updated user: %w", err)
	}

	log.Printf("MCP: Updated points for user %s: %s %d points", userID, operation, points)
	return user.Points, nil
}

// ListUsers lists users from Neo4j
func (s *Service) ListUsers(ctx context.Context, limit, offset int) ([]UserInfo, error) {
	if s.neo4j == nil {
		return nil, fmt.Errorf("Neo4j client not available")
	}

	users, err := s.neo4j.GetAllUsers(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	result := make([]UserInfo, len(users))
	for i, u := range users {
		// Get badge count for user
		badges, _ := s.neo4j.GetUserBadges(ctx, u.ID)

		result[i] = UserInfo{
			ID:       u.ID,
			Name:     u.Name,
			Points:   u.Points,
			Badges:   len(badges),
			Level:    u.Level,
			JoinedAt: u.CreatedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}

// GetUserProfile gets user profile from Neo4j
func (s *Service) GetUserProfile(ctx context.Context, userID string) (map[string]any, error) {
	if s.neo4j == nil {
		return nil, fmt.Errorf("Neo4j client not available")
	}

	// Get user
	user, err := s.neo4j.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	// Get user stats
	stats, err := s.neo4j.GetUserStats(ctx, userID)
	if err != nil {
		log.Printf("Warning: failed to get user stats: %v", err)
		stats = map[string]any{
			"points":  user.Points,
			"level":   user.Level,
			"badges":  0,
			"matches": 0,
		}
	}

	// Get user badges
	badges, _ := s.neo4j.GetUserBadges(ctx, userID)
	badgeList := make([]map[string]any, len(badges))
	for i, b := range badges {
		badgeList[i] = map[string]any{
			"badge_id":    b.BadgeID,
			"name":        b.Name,
			"description": b.Description,
			"icon":        b.Icon,
			"points":      b.Points,
			"earned_at":   b.EarnedAt.Format(time.RFC3339),
		}
	}

	// Get recent activity
	activity, _ := s.neo4j.GetUserRecentActivity(ctx, userID, 10)
	recentActivity := make([]map[string]any, len(activity))
	for i, a := range activity {
		recentActivity[i] = map[string]any{
			"action_type": a.ActionType,
			"points":      a.Points,
			"reason":      a.Reason,
			"timestamp":   a.Timestamp.Format(time.RFC3339),
		}
	}

	return map[string]any{
		"user_id":         user.ID,
		"name":            user.Name,
		"email":           user.Email,
		"points":          user.Points,
		"level":           user.Level,
		"created_at":      user.CreatedAt.Format(time.RFC3339),
		"stats":           stats,
		"badges":          badgeList,
		"recent_activity": recentActivity,
	}, nil
}

// GetAnalyticsSummary gets analytics summary from Neo4j
func (s *Service) GetAnalyticsSummary(ctx context.Context) (map[string]any, error) {
	if s.neo4j == nil {
		return nil, fmt.Errorf("Neo4j client not available")
	}

	// Get various analytics
	totalUsers, err := s.neo4j.GetTotalUsers(ctx)
	if err != nil {
		log.Printf("Warning: failed to get total users: %v", err)
		totalUsers = 0
	}

	totalBadges, err := s.neo4j.GetTotalBadges(ctx)
	if err != nil {
		log.Printf("Warning: failed to get total badges: %v", err)
		totalBadges = 0
	}

	totalPoints, err := s.neo4j.GetTotalPointsDistributed(ctx)
	if err != nil {
		log.Printf("Warning: failed to get total points: %v", err)
		totalPoints = 0
	}

	activeUsers, err := s.neo4j.GetActiveUsersCount(ctx)
	if err != nil {
		log.Printf("Warning: failed to get active users: %v", err)
		activeUsers = 0
	}

	badgeCatalog, err := s.neo4j.GetBadgeCatalogCount(ctx)
	if err != nil {
		log.Printf("Warning: failed to get badge catalog: %v", err)
		badgeCatalog = 0
	}

	// Get active rules count from Redis if available
	activeRules := 0
	if s.redis != nil {
		count, err := s.redis.GetTotalActiveRules(ctx)
		if err == nil {
			activeRules = count
		}
	}

	return map[string]any{
		"total_users":         totalUsers,
		"total_badges_earned": totalBadges,
		"total_points":        totalPoints,
		"active_users_30d":    activeUsers,
		"badge_catalog":       badgeCatalog,
		"active_rules":        activeRules,
	}, nil
}

// Rule represents a gamification rule
type Rule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	EventType   string `json:"event_type"`
	Points      int    `json:"points"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

// UserInfo represents user information
type UserInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Points   int    `json:"points"`
	Badges   int    `json:"badges"`
	Level    int    `json:"level"`
	JoinedAt string `json:"joined_at"`
}

// ListEventTypes returns a list of event types from the registry
func (s *Service) ListEventTypes(ctx context.Context) ([]map[string]any, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("Redis client not available")
	}

	eventTypes, err := s.redis.ListEventTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get event types: %w", err)
	}

	result := make([]map[string]any, len(eventTypes))
	for i, et := range eventTypes {
		result[i] = map[string]any{
			"key":         et.Key,
			"name":        et.Name,
			"description": et.Description,
			"category":    et.Category,
			"enabled":     et.Enabled,
			"created_at":  et.CreatedAt.Format(time.RFC3339),
			"updated_at":  et.UpdatedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}
