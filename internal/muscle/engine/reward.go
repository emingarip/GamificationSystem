package engine

import (
	"context"
	"fmt"
	"log"
	"time"

	"gamification/models"
	"gamification/neo4j"
	"gamification/redis"
	"gamification/websocket"
)

// RewardLayer handles reward execution with Neo4j as source of truth
type RewardLayer struct {
	redisClient     *redis.Client
	neo4jClient     *neo4j.Client
	websocketServer *websocket.Server
}

// NewRewardLayer creates a new reward layer
func NewRewardLayer(redisClient *redis.Client, neo4jClient *neo4j.Client, wsServer *websocket.Server) *RewardLayer {
	return &RewardLayer{
		redisClient:     redisClient,
		neo4jClient:     neo4jClient,
		websocketServer: wsServer,
	}
}

// AwardPoints awards points to a user and creates an append-only action record
// Uses event_id for idempotency - if event already processed, returns early
func (r *RewardLayer) AwardPoints(ctx context.Context, userID string, points int, eventID, reason string) error {
	if userID == "" {
		return fmt.Errorf("userID is required")
	}
	if points <= 0 {
		return fmt.Errorf("points must be positive")
	}
	if eventID == "" {
		return fmt.Errorf("eventID is required for idempotency")
	}

	// Check idempotency - if this action already processed, skip
	alreadyProcessed, err := r.redisClient.IsActionProcessed(ctx, eventID, "", userID, "award_points")
	if err != nil {
		log.Printf("Warning: failed to check action idempotency: %v", err)
	}
	if alreadyProcessed {
		log.Printf("Skipping duplicate points award: event=%s user=%s", eventID, userID)
		return nil
	}

	// Update user points in Neo4j (source of truth)
	if err := r.neo4jClient.AwardPoints(ctx, userID, points, eventID, reason); err != nil {
		return fmt.Errorf("failed to award points in Neo4j: %w", err)
	}

	// Create append-only action record in Neo4j
	if err := r.neo4jClient.RecordRewardAction(ctx, userID, "award_points", points, eventID, reason); err != nil {
		log.Printf("Warning: failed to record action: %v", err)
	}

	// Update Redis cache for points (optional, for fast reads)
	if err := r.redisClient.IncrementUserPoints(ctx, userID, points); err != nil {
		log.Printf("Warning: failed to update Redis points cache: %v", err)
	}

	// Update leaderboard
	if err := r.redisClient.UpdateLeaderboard(ctx, userID, points, "add"); err != nil {
		log.Printf("Warning: failed to update leaderboard: %v", err)
	}

	// Mark action as processed with TTL (24 hours for processed event keys)
	if err := r.redisClient.MarkActionProcessed(ctx, eventID, "", userID, "award_points", 24*time.Hour); err != nil {
		log.Printf("Warning: failed to mark action processed: %v", err)
	}

	// Get updated points for WebSocket notification
	user, err := r.neo4jClient.GetUserByID(ctx, userID)
	if err != nil {
		log.Printf("Warning: failed to get user for points notification: %v", err)
	} else if r.websocketServer != nil {
		r.websocketServer.SendPointsUpdated(userID, user.Points)
	}

	log.Printf("Awarded %d points to user %s (event: %s)", points, userID, eventID)
	return nil
}

// GrantBadge grants a badge to a user with idempotency check
// Returns (bool, error) - bool indicates if badge was newly granted
func (r *RewardLayer) GrantBadge(ctx context.Context, userID, badgeID, eventID, reason string) (bool, error) {
	if userID == "" {
		return false, fmt.Errorf("userID is required")
	}
	if badgeID == "" {
		return false, fmt.Errorf("badgeID is required")
	}
	if eventID == "" {
		return false, fmt.Errorf("eventID is required for idempotency")
	}

	// Check idempotency - if this badge already granted for this event, skip
	alreadyProcessed, err := r.redisClient.IsActionProcessed(ctx, eventID, badgeID, userID, "grant_badge")
	if err != nil {
		log.Printf("Warning: failed to check badge idempotency: %v", err)
	}
	if alreadyProcessed {
		log.Printf("Skipping duplicate badge grant: event=%s badge=%s user=%s", eventID, badgeID, userID)
		return false, nil
	}

	// Check if user already has this badge in Neo4j (source of truth)
	hasBadge, err := r.neo4jClient.CheckBadgeOwnership(ctx, userID, badgeID)
	if err != nil {
		return false, fmt.Errorf("failed to check badge ownership: %w", err)
	}
	if hasBadge {
		log.Printf("User %s already has badge %s", userID, badgeID)
		// Still mark as processed to prevent reprocessing
		r.redisClient.MarkActionProcessed(ctx, eventID, badgeID, userID, "grant_badge", 24*time.Hour)
		return false, nil
	}

	// Get badge details for notification, preferring cache but falling back to Neo4j.
	badgeName := badgeID
	description := ""
	points := 100

	if cachedBadge, err := r.redisClient.GetBadgeByID(ctx, badgeID); err == nil && cachedBadge != nil {
		if cachedBadge.Name != "" {
			badgeName = cachedBadge.Name
		}
		description = cachedBadge.Description
		if cachedBadge.Points > 0 {
			points = cachedBadge.Points
		}
	} else {
		if err != nil {
			log.Printf("Warning: failed to get badge details from Redis: %v", err)
		}

		neo4jBadge, neo4jErr := r.neo4jClient.GetBadgeByID(ctx, badgeID)
		if neo4jErr != nil {
			log.Printf("Warning: failed to get badge details from Neo4j: %v", neo4jErr)
		} else if neo4jBadge != nil {
			if neo4jBadge.Name != "" {
				badgeName = neo4jBadge.Name
			}
			description = neo4jBadge.Description
			if neo4jBadge.Points > 0 {
				points = neo4jBadge.Points
			}

			// Refresh the Redis cache opportunistically.
			cacheBadge := &models.Badge{
				BadgeID:     neo4jBadge.ID,
				Name:        neo4jBadge.Name,
				Description: neo4jBadge.Description,
				Icon:        neo4jBadge.Icon,
				Points:      neo4jBadge.Points,
				Category:    neo4jBadge.Category,
			}
			if _, cacheErr := r.redisClient.CreateBadge(ctx, cacheBadge); cacheErr != nil {
				log.Printf("Warning: failed to cache badge in Redis: %v", cacheErr)
			}
		}
	}

	// Grant badge in Neo4j (creates ownership relationship)
	if err := r.neo4jClient.GrantBadge(ctx, userID, badgeID, eventID, reason); err != nil {
		return false, fmt.Errorf("failed to grant badge in Neo4j: %w", err)
	}

	// Create append-only action record
	if err := r.neo4jClient.RecordRewardAction(ctx, userID, "grant_badge", 0, eventID, reason); err != nil {
		log.Printf("Warning: failed to record badge action: %v", err)
	}

	// Update Redis cache
	if err := r.redisClient.AssignBadgeToUser(ctx, userID, badgeID); err != nil {
		log.Printf("Warning: failed to update Redis badge cache: %v", err)
	}

	// Mark action as processed
	if err := r.redisClient.MarkActionProcessed(ctx, eventID, badgeID, userID, "grant_badge", 24*time.Hour); err != nil {
		log.Printf("Warning: failed to mark badge action processed: %v", err)
	}

	// Publish WebSocket event for badge earned
	if r.websocketServer != nil {
		r.websocketServer.SendBadgeEarned(userID, badgeID, badgeName, description, points)
	}

	log.Printf("Granted badge %s to user %s (event: %s)", badgeID, userID, eventID)
	return true, nil
}

// ProcessEventIdempotently checks if an event has been processed and marks it as processed
// Returns true if the event is a duplicate (already processed)
func (r *RewardLayer) ProcessEventIdempotently(ctx context.Context, eventID string) (bool, error) {
	if eventID == "" {
		return false, fmt.Errorf("eventID is required")
	}

	// Check if event already processed
	return r.redisClient.IsEventProcessed(ctx, eventID)
}

// MarkEventProcessed marks an event as processed in Redis
func (r *RewardLayer) MarkEventProcessed(ctx context.Context, eventID string, ruleID, userID, actionType string) error {
	if eventID == "" {
		return fmt.Errorf("eventID is required")
	}

	// Mark the main event as processed
	if err := r.redisClient.MarkEventProcessed(ctx, eventID, 24*time.Hour); err != nil {
		return fmt.Errorf("failed to mark event processed: %w", err)
	}

	// Mark the specific action as processed for finer-grained idempotency
	if ruleID != "" && userID != "" && actionType != "" {
		if err := r.redisClient.MarkActionProcessed(ctx, eventID, ruleID, userID, actionType, 24*time.Hour); err != nil {
			log.Printf("Warning: failed to mark action processed: %v", err)
		}
	}

	return nil
}

// ExecuteRewardAction executes a reward action based on action type
// This is the main entry point for the reward layer
func (r *RewardLayer) ExecuteRewardAction(ctx context.Context, userID, ruleID string, action *models.RuleAction, event *models.MatchEvent) error {
	idempotencyKey := fmt.Sprintf("%s:%s", event.EventID, ruleID)
	switch action.ActionType {
	case "award_points":
		points, _ := action.Params["points"].(float64)
		reason, _ := action.Params["reason"].(string)
		if reason == "" {
			reason = fmt.Sprintf("Event: %s", event.EventType)
		}
		return r.AwardPoints(ctx, userID, int(points), idempotencyKey, reason)

	case "grant_badge":
		badgeID, _ := action.Params["badge_id"].(string)
		reason, _ := action.Params["reason"].(string)
		if reason == "" {
			reason = fmt.Sprintf("Earned badge from event: %s", event.EventType)
		}
		_, err := r.GrantBadge(ctx, userID, badgeID, idempotencyKey, reason)
		return err

	default:
		return fmt.Errorf("unknown action type: %s", action.ActionType)
	}
}
