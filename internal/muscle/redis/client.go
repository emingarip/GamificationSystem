package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"gamification/config"
	"gamification/models"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Key patterns for Redis
const (
	// Rule keys
	RuleKeyPrefix      = "rule:"         // rule:{rule_id}
	RuleListKey        = "rules:all"     // Set of all rule IDs
	RuleActiveSetKey   = "rules:active"  // Sorted set of active rules by event type
	RuleByEventTypeKey = "rules:by_type" // Sorted set per event type: rules:by_type:{event_type}
	RuleCooldownKey    = "rule:cooldown" // rule:cooldown:{rule_id}:{match_id}:{player_id}

	// Badge keys
	BadgeKeyPrefix     = "badge:"       // badge:{badge_id}
	BadgeListKey       = "badges:all"   // List of all badge IDs
	UserBadgeKeyPrefix = "user:badges:" // user:badges:{user_id}

	// Leaderboard keys
	LeaderboardKey       = "leaderboard"  // Sorted set of users by points
	LeaderboardByTypeKey = "leaderboard:" // leaderboard:{event_type}

	// Event tracking keys
	EventHistoryKey     = "events:history"     // List of recent events
	EventDebugLogKey    = "events:debug_logs"  // List of recent event evaluations for debugger
	EventCountKey       = "events:count"       // events:count:{match_id}:{player_id}:{event_type}
	PlayerMatchStatsKey = "player:match:stats" // player:match:stats:{player_id}:{match_id}

	// Achievement keys
	UserAchievementsKey = "user:achievements" // user:achievements:{user_id}
	BadgeProgressKey    = "badge:progress"    // badge:progress:{user_id}:{badge_id}

	// Idempotency keys
	ProcessedEventKey  = "processed_event:"  // processed_event:{event_id}
	ProcessedActionKey = "processed_action:" // processed_action:{event_id}:{rule_id}:{user_id}:{action_type}

	// Event Type Registry keys
	EventTypeKeyPrefix = "event_type:"     // event_type:{key}
	EventTypeListKey   = "event_types:all" // List of all event type keys
)

// Client wraps the Redis connection
type Client struct {
	client *redis.Client
	config *config.RedisConfig
}

// NewTestClient creates a Redis client for testing (e.g., with miniredis)
// This bypasses the connection check that NewClient does
func NewTestClient(rdb *redis.Client, cfg *config.RedisConfig) *Client {
	return &Client{
		client: rdb,
		config: cfg,
	}
}
func NewClient(cfg *config.RedisConfig) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolTimeout:  cfg.PoolTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Client{
		client: client,
		config: cfg,
	}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.client.Close()
}

// Raw returns the underlying go-redis client
func (c *Client) Raw() *redis.Client {
	return c.client
}

// GetRulesByEventType retrieves active rules for a specific event type
// Uses sorted set for efficient matching - rules sorted by priority
func (c *Client) GetRulesByEventType(ctx context.Context, eventType models.EventType) ([]models.Rule, error) {
	key := fmt.Sprintf("%s:%s", RuleByEventTypeKey, eventType)

	// Get all rule IDs for this event type
	ruleIDs, err := c.client.ZRevRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get rules by event type: %w", err)
	}

	if len(ruleIDs) == 0 {
		return nil, nil
	}

	// Fetch full rule objects. Missing keys can exist if a rule was deleted and a
	// stale member was left behind in the event-type sorted set, so treat those as
	// cleanup work instead of failing the entire lookup.
	keys := make([]string, len(ruleIDs))
	for i, ruleID := range ruleIDs {
		keys[i] = RuleKeyPrefix + ruleID
	}

	values, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rules: %w", err)
	}

	rules := make([]models.Rule, 0, len(ruleIDs))
	staleRuleIDs := make([]any, 0)

	for i, value := range values {
		if value == nil {
			staleRuleIDs = append(staleRuleIDs, ruleIDs[i])
			continue
		}

		data, ok := value.(string)
		if !ok {
			continue
		}

		var rule models.Rule
		if err := json.Unmarshal([]byte(data), &rule); err != nil {
			continue
		}
		if rule.IsActive {
			rules = append(rules, rule)
		}
	}

	if len(staleRuleIDs) > 0 {
		c.client.ZRem(ctx, key, staleRuleIDs...)
	}

	return rules, nil
}

// GetRuleByID retrieves a single rule by ID
func (c *Client) GetRuleByID(ctx context.Context, ruleID string) (*models.Rule, error) {
	key := RuleKeyPrefix + ruleID
	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	var rule models.Rule
	if err := json.Unmarshal([]byte(data), &rule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rule: %w", err)
	}

	return &rule, nil
}

// SaveRule saves a rule to Redis
func (c *Client) SaveRule(ctx context.Context, rule *models.Rule) error {
	var existing *models.Rule
	var err error

	if rule.RuleID == "" {
		rule.RuleID = "rule_" + uuid.NewString()
	} else {
		existing, err = c.GetRuleByID(ctx, rule.RuleID)
		if err != nil {
			return fmt.Errorf("failed to load existing rule: %w", err)
		}
	}

	data, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("failed to marshal rule: %w", err)
	}

	// Save rule data
	ruleKey := RuleKeyPrefix + rule.RuleID
	if err := c.client.Set(ctx, ruleKey, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to save rule: %w", err)
	}

	if err := c.client.SAdd(ctx, RuleListKey, rule.RuleID).Err(); err != nil {
		return fmt.Errorf("failed to add to rule list: %w", err)
	}

	// Keep secondary indexes in sync on both create and update.
	c.client.ZRem(ctx, RuleActiveSetKey, rule.RuleID)

	oldEventType := rule.EventType
	if existing != nil && existing.EventType != "" {
		oldEventType = existing.EventType
	}
	if oldEventType != "" {
		oldTypeKey := fmt.Sprintf("%s:%s", RuleByEventTypeKey, oldEventType)
		c.client.ZRem(ctx, oldTypeKey, rule.RuleID)
	}

	// Add to active rules sorted set by priority
	if rule.IsActive {
		activeKey := RuleActiveSetKey
		if err := c.client.ZAdd(ctx, activeKey, redis.Z{
			Score:  float64(rule.Priority),
			Member: rule.RuleID,
		}).Err(); err != nil {
			return fmt.Errorf("failed to add to active set: %w", err)
		}

		// Add to event type specific sorted set
		typeKey := fmt.Sprintf("%s:%s", RuleByEventTypeKey, rule.EventType)
		if err := c.client.ZAdd(ctx, typeKey, redis.Z{
			Score:  float64(rule.Priority),
			Member: rule.RuleID,
		}).Err(); err != nil {
			return fmt.Errorf("failed to add to type set: %w", err)
		}
	}

	return nil
}

// SetCooldown sets a cooldown for a rule to prevent duplicate triggers
func (c *Client) SetCooldown(ctx context.Context, ruleID, matchID, playerID string, duration time.Duration) error {
	key := fmt.Sprintf("%s:%s:%s:%s", RuleCooldownKey, ruleID, matchID, playerID)
	return c.client.Set(ctx, key, "1", duration).Err()
}

// CheckCooldown checks if a rule is in cooldown
func (c *Client) CheckCooldown(ctx context.Context, ruleID, matchID, playerID string) (bool, error) {
	key := fmt.Sprintf("%s:%s:%s:%s", RuleCooldownKey, ruleID, matchID, playerID)
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check cooldown: %w", err)
	}
	return exists > 0, nil
}

// IncrementEventCount increments the event count for tracking consecutive events
func (c *Client) IncrementEventCount(ctx context.Context, matchID, playerID string, eventType models.EventType) (int64, error) {
	key := fmt.Sprintf("%s:%s:%s:%s", EventCountKey, matchID, playerID, eventType)
	count, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment event count: %w", err)
	}
	// Set expiry to match duration (90 mins + extra time)
	c.client.Expire(ctx, key, 120*time.Minute)
	return count, nil
}

// GetEventCount gets the current event count
func (c *Client) GetEventCount(ctx context.Context, matchID, playerID string, eventType models.EventType) (int64, error) {
	key := fmt.Sprintf("%s:%s:%s:%s", EventCountKey, matchID, playerID, eventType)
	count, err := c.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

// ResetEventCount resets the match event count
func (c *Client) ResetEventCount(ctx context.Context, matchID, playerID string, eventType models.EventType) error {
	key := fmt.Sprintf("%s:%s:%s:%s", EventCountKey, matchID, playerID, eventType)
	return c.client.Del(ctx, key).Err()
}

// IncrementGlobalEventCount increments the lifetime event count for a player
func (c *Client) IncrementGlobalEventCount(ctx context.Context, playerID string, eventType models.EventType) (int64, error) {
	key := fmt.Sprintf("%s:global:%s:%s", EventCountKey, playerID, eventType)
	count, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment global event count: %w", err)
	}
	return count, nil
}

// GetGlobalEventCount gets the lifetime event count for a player
func (c *Client) GetGlobalEventCount(ctx context.Context, playerID string, eventType models.EventType) (int64, error) {
	key := fmt.Sprintf("%s:global:%s:%s", EventCountKey, playerID, eventType)
	count, err := c.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

// UpdateDailyStreak increments the daily streak if event happens today, handles resets
func (c *Client) UpdateDailyStreak(ctx context.Context, playerID string, eventType models.EventType) (int64, error) {
	streakKey := fmt.Sprintf("%s:streak:%s:%s", EventCountKey, playerID, eventType)
	lastDateKey := fmt.Sprintf("%s:last_date:%s:%s", EventCountKey, playerID, eventType)
	
	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	
	// Get last active date
	lastDate, err := c.client.Get(ctx, lastDateKey).Result()
	if err != nil && err != redis.Nil {
		return 0, fmt.Errorf("failed to get last date: %w", err)
	}
	
	var streak int64
	if lastDate == today {
		// Already updated today, just get the current streak
		streak, err = c.client.Get(ctx, streakKey).Int64()
		if err != nil {
			streak = 1 // Fallback
		}
		return streak, nil
	}
	
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	if lastDate == yesterday {
		// Streak continues
		streak, err = c.client.Incr(ctx, streakKey).Result()
		if err != nil {
			return 0, fmt.Errorf("failed to increment streak: %w", err)
		}
	} else {
		// Streak broken or first time
		err = c.client.Set(ctx, streakKey, 1, 0).Err()
		if err != nil {
			return 0, fmt.Errorf("failed to reset streak: %w", err)
		}
		streak = 1
	}
	
	// Update last date
	err = c.client.Set(ctx, lastDateKey, today, 0).Err()
	if err != nil {
		return 0, fmt.Errorf("failed to set last date: %w", err)
	}
	
	return streak, nil
}

// GetDailyStreak gets the current daily streak
func (c *Client) GetDailyStreak(ctx context.Context, playerID string, eventType models.EventType) (int64, error) {
	streakKey := fmt.Sprintf("%s:streak:%s:%s", EventCountKey, playerID, eventType)
	lastDateKey := fmt.Sprintf("%s:last_date:%s:%s", EventCountKey, playerID, eventType)
	
	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	
	lastDate, err := c.client.Get(ctx, lastDateKey).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get last date: %w", err)
	}
	
	if lastDate != today && lastDate != yesterday {
		return 0, nil
	}
	
	streak, err := c.client.Get(ctx, streakKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return streak, err
}

// StoreMatchEvent stores the match event in Redis for history
func (c *Client) StoreMatchEvent(ctx context.Context, event *models.MatchEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	return c.client.LPush(ctx, EventHistoryKey, data).Err()
}

// Ping checks the connection
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// DeleteRule deletes a rule from Redis
func (c *Client) DeleteRule(ctx context.Context, ruleID string) error {
	existing, err := c.GetRuleByID(ctx, ruleID)
	if err != nil {
		return fmt.Errorf("failed to load rule before delete: %w", err)
	}

	// Delete the rule key
	key := RuleKeyPrefix + ruleID
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	// Remove from active rules sorted set
	c.client.ZRem(ctx, RuleActiveSetKey, ruleID)
	c.client.SRem(ctx, RuleListKey, ruleID)

	if existing != nil && existing.EventType != "" {
		typeKey := fmt.Sprintf("%s:%s", RuleByEventTypeKey, existing.EventType)
		c.client.ZRem(ctx, typeKey, ruleID)
	}

	return nil
}

// GetAllBadges retrieves all badges
func (c *Client) GetAllBadges(ctx context.Context) ([]models.Badge, error) {
	// Get all badge IDs from the list
	badgeIDs, err := c.client.LRange(ctx, BadgeListKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get badge list: %w", err)
	}

	if len(badgeIDs) == 0 {
		return []models.Badge{}, nil
	}

	// Fetch full badge objects
	badges := make([]models.Badge, 0, len(badgeIDs))
	for _, badgeID := range badgeIDs {
		key := BadgeKeyPrefix + badgeID
		data, err := c.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		var badge models.Badge
		if err := json.Unmarshal([]byte(data), &badge); err != nil {
			continue
		}
		badges = append(badges, badge)
	}

	return badges, nil
}

// GetBadgeByID retrieves a single badge by ID
func (c *Client) GetBadgeByID(ctx context.Context, badgeID string) (*models.Badge, error) {
	key := BadgeKeyPrefix + badgeID
	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get badge: %w", err)
	}

	var badge models.Badge
	if err := json.Unmarshal([]byte(data), &badge); err != nil {
		return nil, fmt.Errorf("failed to unmarshal badge: %w", err)
	}

	return &badge, nil
}

// CreateBadge creates a new badge
func (c *Client) CreateBadge(ctx context.Context, badge *models.Badge) (string, error) {
	// Generate ID if not provided
	if badge.BadgeID == "" {
		badge.BadgeID = fmt.Sprintf("badge_%d", time.Now().UnixNano())
	}
	badge.CreatedAt = time.Now()

	data, err := json.Marshal(badge)
	if err != nil {
		return "", fmt.Errorf("failed to marshal badge: %w", err)
	}

	// Save badge data
	key := BadgeKeyPrefix + badge.BadgeID
	if err := c.client.Set(ctx, key, data, 0).Err(); err != nil {
		return "", fmt.Errorf("failed to save badge: %w", err)
	}

	// Add to badge list
	c.client.RPush(ctx, BadgeListKey, badge.BadgeID)

	return badge.BadgeID, nil
}

// UpdateBadge updates an existing badge in Redis.
func (c *Client) UpdateBadge(ctx context.Context, badge *models.Badge) error {
	if badge.BadgeID == "" {
		return fmt.Errorf("badge id is required")
	}

	data, err := json.Marshal(badge)
	if err != nil {
		return fmt.Errorf("failed to marshal badge: %w", err)
	}

	key := BadgeKeyPrefix + badge.BadgeID
	if err := c.client.Set(ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to update badge: %w", err)
	}

	return nil
}

// DeleteBadge removes a badge from Redis.
func (c *Client) DeleteBadge(ctx context.Context, badgeID string) error {
	key := BadgeKeyPrefix + badgeID
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete badge: %w", err)
	}
	if err := c.client.LRem(ctx, BadgeListKey, 0, badgeID).Err(); err != nil {
		return fmt.Errorf("failed to remove badge from list: %w", err)
	}
	return nil
}

// AssignBadgeToUser assigns a badge to a user
func (c *Client) AssignBadgeToUser(ctx context.Context, userID, badgeID string) error {
	key := UserBadgeKeyPrefix + userID

	// Check if user already has this badge
	badges, err := c.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("failed to get user badges: %w", err)
	}

	for _, b := range badges {
		if b == badgeID {
			return nil // Already has the badge
		}
	}

	// Add badge to user's list
	userBadge := models.UserBadge{
		UserID:   userID,
		BadgeID:  badgeID,
		EarnedAt: time.Now(),
	}

	badgeData, err := json.Marshal(userBadge)
	if err != nil {
		return fmt.Errorf("failed to marshal user badge: %w", err)
	}

	return c.client.RPush(ctx, key, string(badgeData)).Err()
}

// GetUserBadges retrieves all badges for a user
func (c *Client) GetUserBadges(ctx context.Context, userID string) ([]models.UserBadge, error) {
	key := UserBadgeKeyPrefix + userID

	badges, err := c.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user badges: %w", err)
	}

	userBadges := make([]models.UserBadge, 0, len(badges))
	for _, b := range badges {
		var ub models.UserBadge
		if err := json.Unmarshal([]byte(b), &ub); err != nil {
			continue
		}
		userBadges = append(userBadges, ub)
	}

	return userBadges, nil
}

// UpdateLeaderboard updates a user's score in the leaderboard
// operation can be "add", "subtract", or "set"
func (c *Client) UpdateLeaderboard(ctx context.Context, userID string, points int, operation string) error {
	// Determine the delta based on operation type
	var delta float64

	switch operation {
	case "subtract":
		// For subtract: decrement by points (negative delta)
		delta = -float64(points)
	case "set":
		// For set: calculate delta between new value and current Redis value
		currentScore, err := c.client.ZScore(ctx, LeaderboardKey, userID).Result()
		if err == redis.Nil {
			// User not in leaderboard, treat as 0
			currentScore = 0
		} else if err != nil {
			// On error, fallback to add behavior
			delta = float64(points)
			goto update
		}
		delta = float64(points) - currentScore
	default:
		// For "add" or any other operation: increment by points (default behavior)
		delta = float64(points)
	}

update:
	// Update main leaderboard
	err := c.client.ZIncrBy(ctx, LeaderboardKey, delta, userID).Err()
	if err != nil {
		return fmt.Errorf("failed to update leaderboard: %w", err)
	}

	return nil
}

// GetLeaderboard retrieves the top users by points
func (c *Client) GetLeaderboard(ctx context.Context, eventType string, limit int) ([]redis.Z, error) {
	var key string
	if eventType != "" {
		key = LeaderboardByTypeKey + eventType
	} else {
		key = LeaderboardKey
	}

	// Get top users (highest score first)
	results, err := c.client.ZRevRangeWithScores(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get leaderboard: %w", err)
	}

	return results, nil
}

// SetUserPoints sets a user's total points (not increment)
func (c *Client) SetUserPoints(ctx context.Context, userID string, points int) error {
	key := "user:points:" + userID
	return c.client.Set(ctx, key, points, 0).Err()
}

// GetUserPoints gets a user's total points
func (c *Client) GetUserPoints(ctx context.Context, userID string) (int, error) {
	key := "user:points:" + userID
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	points, _ := strconv.Atoi(val)
	return points, nil
}

// DeleteUserData removes Redis state owned by a user.
func (c *Client) DeleteUserData(ctx context.Context, userID string) error {
	keys := []string{
		"user:points:" + userID,
		UserBadgeKeyPrefix + userID,
	}

	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to delete user keys: %w", err)
	}
	if err := c.client.ZRem(ctx, LeaderboardKey, userID).Err(); err != nil {
		return fmt.Errorf("failed to remove user from leaderboard: %w", err)
	}
	return nil
}

// IsEventProcessed checks if an event has already been processed
// Returns true if the event was already processed
func (c *Client) IsEventProcessed(ctx context.Context, eventID string) (bool, error) {
	key := ProcessedEventKey + eventID
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check processed event: %w", err)
	}
	return exists > 0, nil
}

// MarkEventProcessed marks an event as processed in Redis with TTL
func (c *Client) MarkEventProcessed(ctx context.Context, eventID string, ttl time.Duration) error {
	key := ProcessedEventKey + eventID
	if err := c.client.Set(ctx, key, "1", ttl).Err(); err != nil {
		return fmt.Errorf("failed to mark event processed: %w", err)
	}
	return nil
}

// IsActionProcessed checks if a specific action has already been processed
// ruleID can be empty for actions that don't have a rule (like direct award_points)
// Returns true if the action was already processed
func (c *Client) IsActionProcessed(ctx context.Context, eventID, ruleID, userID, actionType string) (bool, error) {
	key := ProcessedActionKey + eventID + ":" + ruleID + ":" + userID + ":" + actionType
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check processed action: %w", err)
	}
	return exists > 0, nil
}

// MarkActionProcessed marks a specific action as processed in Redis with TTL
func (c *Client) MarkActionProcessed(ctx context.Context, eventID, ruleID, userID, actionType string, ttl time.Duration) error {
	key := ProcessedActionKey + eventID + ":" + ruleID + ":" + userID + ":" + actionType
	if err := c.client.Set(ctx, key, "1", ttl).Err(); err != nil {
		return fmt.Errorf("failed to mark action processed: %w", err)
	}
	return nil
}

// IncrementUserPoints increments a user's points in Redis cache
func (c *Client) IncrementUserPoints(ctx context.Context, userID string, points int) error {
	key := "user:points:" + userID
	return c.client.IncrBy(ctx, key, int64(points)).Err()
}

// GetEventHistoryCount returns the number of events in history
func (c *Client) GetEventHistoryCount(ctx context.Context) (int, error) {
	count, err := c.client.LLen(ctx, EventHistoryKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get event count: %w", err)
	}
	return int(count), nil
}

// GetAllRules retrieves all rules regardless of event type
// Used when no event_type filter is provided in the API
func (c *Client) GetAllRules(ctx context.Context) ([]models.Rule, error) {
	ruleIDs, err := c.client.SMembers(ctx, RuleListKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get all rules: %w", err)
	}

	// Backfill legacy installs that predate the rules:all index.
	if len(ruleIDs) == 0 {
		var cursor uint64
		seen := make(map[string]struct{})

		for {
			keys, nextCursor, scanErr := c.client.Scan(ctx, cursor, RuleKeyPrefix+"*", 100).Result()
			if scanErr != nil {
				return nil, fmt.Errorf("failed to scan rules: %w", scanErr)
			}

			for _, key := range keys {
				if strings.HasPrefix(key, RuleCooldownKey+":") {
					continue
				}
				ruleID := strings.TrimPrefix(key, RuleKeyPrefix)
				if ruleID == "" {
					migratedID, migrateErr := c.migrateLegacyBlankRuleID(ctx, key)
					if migrateErr != nil || migratedID == "" {
						continue
					}
					ruleID = migratedID
				}
				if _, ok := seen[ruleID]; ok {
					continue
				}
				seen[ruleID] = struct{}{}
				ruleIDs = append(ruleIDs, ruleID)
			}

			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}

		if len(ruleIDs) > 0 {
			members := make([]any, len(ruleIDs))
			for i, ruleID := range ruleIDs {
				members[i] = ruleID
			}
			c.client.SAdd(ctx, RuleListKey, members...)
		}
	}

	if len(ruleIDs) == 0 {
		return []models.Rule{}, nil
	}

	// Fetch full rule objects
	keys := make([]string, len(ruleIDs))
	for i, ruleID := range ruleIDs {
		keys[i] = RuleKeyPrefix + ruleID
	}

	values, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rules: %w", err)
	}

	rules := make([]models.Rule, 0, len(ruleIDs))
	for _, value := range values {
		if value == nil {
			continue
		}

		data, ok := value.(string)
		if !ok {
			continue
		}

		var rule models.Rule
		if err := json.Unmarshal([]byte(data), &rule); err != nil {
			continue
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func (c *Client) migrateLegacyBlankRuleID(ctx context.Context, key string) (string, error) {
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}

	var rule models.Rule
	if err := json.Unmarshal([]byte(data), &rule); err != nil {
		return "", err
	}

	if rule.RuleID != "" {
		return rule.RuleID, nil
	}

	oldEventType := rule.EventType
	oldActive := rule.IsActive

	rule.RuleID = "rule_" + uuid.NewString()
	if err := c.SaveRule(ctx, &rule); err != nil {
		return "", err
	}

	c.client.Del(ctx, key)
	c.client.ZRem(ctx, RuleActiveSetKey, "")
	if oldActive && oldEventType != "" {
		oldTypeKey := fmt.Sprintf("%s:%s", RuleByEventTypeKey, oldEventType)
		c.client.ZRem(ctx, oldTypeKey, "")
	}

	return rule.RuleID, nil
}

// GetTotalActiveRules returns count of active rules with conditions
func (c *Client) GetTotalActiveRules(ctx context.Context) (int, error) {
	// Get all rule IDs from the active rules sorted set
	ruleIDs, err := c.client.ZRevRange(ctx, RuleActiveSetKey, 0, -1).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get active rules: %w", err)
	}

	if len(ruleIDs) == 0 {
		return 0, nil
	}

	// Count rules that have conditions
	count := 0
	for _, ruleID := range ruleIDs {
		key := RuleKeyPrefix + ruleID
		data, err := c.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		var rule models.Rule
		if err := json.Unmarshal([]byte(data), &rule); err != nil {
			continue
		}
		// Count rules that have conditions defined
		if len(rule.Conditions) > 0 {
			count++
		}
	}

	return count, nil
}

// GetBadgeInfoWithDetails retrieves badge info including metadata from Redis
func (c *Client) GetBadgeInfoWithDetails(ctx context.Context, badgeID string) (map[string]any, error) {
	badge, err := c.GetBadgeByID(ctx, badgeID)
	if err != nil || badge == nil {
		return nil, err
	}

	return map[string]any{
		"id":          badge.BadgeID,
		"name":        badge.Name,
		"description": badge.Description,
		"icon":        badge.Icon,
		"points":      badge.Points,
	}, nil
}

// ==================== Event Type Registry ====================

// EventType represents a dynamic event type in the registry
type EventType struct {
	Key           string         `json:"key"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Category      string         `json:"category"`
	Enabled       bool           `json:"enabled"`
	SamplePayload map[string]any `json:"sample_payload,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// CreateEventType creates a new event type in the registry
func (c *Client) CreateEventType(ctx context.Context, eventType *EventType) (string, error) {
	if eventType.Key == "" {
		return "", fmt.Errorf("event type key is required")
	}

	// Check if event type already exists
	existing, err := c.GetEventType(ctx, eventType.Key)
	if err != nil {
		return "", fmt.Errorf("failed to check event type existence: %w", err)
	}
	if existing != nil {
		return "", fmt.Errorf("event type with key '%s' already exists", eventType.Key)
	}

	// Set timestamps
	now := time.Now()
	eventType.CreatedAt = now
	eventType.UpdatedAt = now

	// Default category to "custom" if not specified
	if eventType.Category == "" {
		eventType.Category = "custom"
	}

	data, err := json.Marshal(eventType)
	if err != nil {
		return "", fmt.Errorf("failed to marshal event type: %w", err)
	}

	// Save event type data
	key := EventTypeKeyPrefix + eventType.Key
	if err := c.client.Set(ctx, key, data, 0).Err(); err != nil {
		return "", fmt.Errorf("failed to save event type: %w", err)
	}

	// Add to event type list
	c.client.RPush(ctx, EventTypeListKey, eventType.Key)

	return eventType.Key, nil
}

// GetEventType retrieves a single event type by key
func (c *Client) GetEventType(ctx context.Context, key string) (*EventType, error) {
	keyStr := EventTypeKeyPrefix + key
	data, err := c.client.Get(ctx, keyStr).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get event type: %w", err)
	}

	var eventType EventType
	if err := json.Unmarshal([]byte(data), &eventType); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event type: %w", err)
	}

	return &eventType, nil
}

// ListEventTypes retrieves all event types
func (c *Client) ListEventTypes(ctx context.Context) ([]EventType, error) {
	// Get all event type keys
	keys, err := c.client.LRange(ctx, EventTypeListKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get event type list: %w", err)
	}

	if len(keys) == 0 {
		return []EventType{}, nil
	}

	// Fetch full event type objects
	eventTypes := make([]EventType, 0, len(keys))
	for _, key := range keys {
		eventType, err := c.GetEventType(ctx, key)
		if err != nil {
			continue
		}
		if eventType != nil {
			eventTypes = append(eventTypes, *eventType)
		}
	}

	return eventTypes, nil
}

// UpdateEventType updates an existing event type
func (c *Client) UpdateEventType(ctx context.Context, eventType *EventType) error {
	if eventType.Key == "" {
		return fmt.Errorf("event type key is required")
	}

	// Check if event type exists
	existing, err := c.GetEventType(ctx, eventType.Key)
	if err != nil {
		return fmt.Errorf("failed to get event type: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("event type not found")
	}

	// Preserve created_at, update updated_at
	eventType.CreatedAt = existing.CreatedAt
	eventType.UpdatedAt = time.Now()

	data, err := json.Marshal(eventType)
	if err != nil {
		return fmt.Errorf("failed to marshal event type: %w", err)
	}

	key := EventTypeKeyPrefix + eventType.Key
	if err := c.client.Set(ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to update event type: %w", err)
	}

	return nil
}

// DeleteEventType removes an event type from the registry
func (c *Client) DeleteEventType(ctx context.Context, key string) error {
	keyStr := EventTypeKeyPrefix + key

	// Delete the event type data
	if err := c.client.Del(ctx, keyStr).Err(); err != nil {
		return fmt.Errorf("failed to delete event type: %w", err)
	}

	// Remove from event type list
	c.client.LRem(ctx, EventTypeListKey, 0, key)

	return nil
}

// GetEnabledEventTypes retrieves only enabled event types
func (c *Client) GetEnabledEventTypes(ctx context.Context) ([]EventType, error) {
	allTypes, err := c.ListEventTypes(ctx)
	if err != nil {
		return nil, err
	}

	enabledTypes := make([]EventType, 0)
	for _, et := range allTypes {
		if et.Enabled {
			enabledTypes = append(enabledTypes, et)
		}
	}

	return enabledTypes, nil
}

// SeedDefaultEventTypes seeds the default sport event types if they don't exist
func (c *Client) SeedDefaultEventTypes(ctx context.Context) error {
	defaultTypes := []EventType{
		{Key: "goal", Name: "Goal", Description: "A goal scored in a match", Category: "sport", Enabled: true},
		{Key: "corner", Name: "Corner", Description: "A corner kick", Category: "sport", Enabled: true},
		{Key: "foul", Name: "Foul", Description: "A foul committed", Category: "sport", Enabled: true},
		{Key: "yellow_card", Name: "Yellow Card", Description: "A yellow card shown", Category: "sport", Enabled: true},
		{Key: "red_card", Name: "Red Card", Description: "A red card shown", Category: "sport", Enabled: true},
		{Key: "penalty", Name: "Penalty", Description: "A penalty kick", Category: "sport", Enabled: true},
		{Key: "offside", Name: "Offside", Description: "An offside call", Category: "sport", Enabled: true},
	}

	for _, et := range defaultTypes {
		_, err := c.CreateEventType(ctx, &et)
		if err != nil {
			// Ignore "already exists" errors
			if !strings.Contains(err.Error(), "already exists") {
				log.Printf("Warning: failed to seed event type %s: %v", et.Key, err)
			}
		}
	}

	return nil
}

// LogEventEvaluation stores a rule evaluation result in the debug log list (capped at 1000 entries)
func (c *Client) LogEventEvaluation(ctx context.Context, eval *models.RuleEngineResult) error {
	// For RuleEngineResult, Error interface cannot be serialized easily if present
	// So we might need to handle evaluating the error to string. We'll Marshal the struct directly.
	
	// Create a safe copy for serialization if needed, or just serialize directly
	type ErrorSafeResult struct {
		Event          *models.MatchEvent            `json:"event"`
		TriggeredRules []models.RuleEvaluationResult `json:"triggered_rules"`
		TotalTimeMs    float64                       `json:"total_time_ms"`
		Success        bool                          `json:"success"`
		Error          string                        `json:"error,omitempty"`
		Skipped        bool                          `json:"skipped"`
		SkipReason     string                        `json:"skip_reason,omitempty"`
	}
	
	safeEval := ErrorSafeResult{
		Event:          eval.Event,
		TriggeredRules: eval.TriggeredRules,
		TotalTimeMs:    eval.TotalTimeMs,
		Success:        eval.Success,
		Skipped:        eval.Skipped,
		SkipReason:     eval.SkipReason,
	}
	if eval.Error != nil {
		safeEval.Error = eval.Error.Error()
	}

	data, err := json.Marshal(safeEval)
	if err != nil {
		return fmt.Errorf("failed to marshal event evaluation: %w", err)
	}

	pipe := c.client.Pipeline()
	pipe.LPush(ctx, EventDebugLogKey, data)
	pipe.LTrim(ctx, EventDebugLogKey, 0, 999) // Keep last 1000 items

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save event debug log: %w", err)
	}

	return nil
}

// GetEventEvaluations retrieves the most recent event evaluations for the debugger
func (c *Client) GetEventEvaluations(ctx context.Context, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 100
	}
	
	data, err := c.client.LRange(ctx, EventDebugLogKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get event evaluations: %w", err)
	}

	var results []map[string]any
	for _, item := range data {
		var res map[string]any
		if err := json.Unmarshal([]byte(item), &res); err != nil {
			log.Printf("Warning: failed to unmarshal event evaluation: %v", err)
			continue
		}
		// ensure a timestamp string exists for the debugger
		if res["event"] != nil {
			if evtMap, ok := res["event"].(map[string]any); ok {
				res["timestamp"] = evtMap["timestamp"]
			}
		}
		results = append(results, res)
	}

	return results, nil
}
