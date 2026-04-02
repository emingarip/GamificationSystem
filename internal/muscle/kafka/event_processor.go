package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gamification/config"
	"gamification/engine"
	"gamification/models"
	"gamification/neo4j"
	"gamification/redis"

	"github.com/segmentio/kafka-go"
)

// EventProcessor handles the processing of match events from Kafka.
// It coordinates between the synchronous Redis/Kafka "Muscle" layer and the
// asynchronous LLM/Knowledge Graph "Brain" layer.
//
// Architecture Notes:
// - Muscle (this file): Low-latency, synchronous event processing via Redis/Kafka
// - Brain (LLM/Knowledge Graph): Async, high-latency inference and graph updates
// - This processor is the bridge that triggers Brain processing after Muscle actions
type EventProcessor struct {
	redisClient     *redis.Client
	neo4jClient     *neo4j.Client
	ruleMatcher     *engine.RuleMatcher
	rewardLayer     *engine.RewardLayer
	kafkaConsumer   *Consumer
	ruleEngineTopic string
}

// NewEventProcessor creates a new event processor with dependencies
func NewEventProcessor(
	redisClient *redis.Client,
	neo4jClient *neo4j.Client,
	ruleMatcher *engine.RuleMatcher,
	rewardLayer *engine.RewardLayer,
	kafkaConfig *config.KafkaConfig,
) *EventProcessor {
	return &EventProcessor{
		redisClient:     redisClient,
		neo4jClient:     neo4jClient,
		ruleMatcher:     ruleMatcher,
		rewardLayer:     rewardLayer,
		ruleEngineTopic: "rule-triggered",
	}
}

// MatchEventConsumer starts the Kafka consumer loop that reads from the match-events topic.
// This is the main entry point for the synchronous event processing pipeline.
//
// Flow:
// 1. Read event from Kafka (match-events topic)
// 2. Process event through rule engine (synchronous)
// 3. Execute reward actions (synchronous)
// 4. Optionally trigger Brain async processing for complex rules
//
// The consumer runs until the context is cancelled or an unrecoverable error occurs.
func (ep *EventProcessor) MatchEventConsumer(ctx context.Context) error {
	log.Println("Starting MatchEventConsumer - listening on 'match-events' topic...")

	// Create a dedicated Kafka reader for the match-events topic
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{"localhost:9092"},
		GroupID:        "muscle-event-processor",
		Topic:          "match-events",
		MinBytes:       1e3,
		MaxBytes:       10e6,
		MaxWait:        1 * time.Second,
		StartOffset:    kafka.LastOffset,
		CommitInterval: 1 * time.Second,
	})
	defer reader.Close()

	for {
		select {
		case <-ctx.Done():
			log.Println("MatchEventConsumer shutting down...")
			return ctx.Err()
		default:
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				log.Printf("Error fetching message: %v", err)
				continue
			}

			// Process the event
			if err := ep.processKafkaMessage(ctx, msg); err != nil {
				log.Printf("Error processing message: %v", err)
				// Continue processing other messages even if one fails
			}

			// Commit the message after processing
			if err := reader.CommitMessages(ctx, msg); err != nil {
				log.Printf("Error committing message: %v", err)
			}
		}
	}
}

// processKafkaMessage handles the processing of a single Kafka message
func (ep *EventProcessor) processKafkaMessage(ctx context.Context, msg kafka.Message) error {
	var event models.MatchEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	log.Printf("Received event: type=%s, match=%s, player=%s",
		event.EventType, event.MatchID, event.PlayerID)

	// Process the event through the rule engine
	return ep.ProcessEvent(ctx, event)
}

// ProcessEvent is the main processor that matches rules and finds users.
// This is the core of the Muscle layer - high-performance, synchronous processing.
//
// Steps:
// 1. Check for duplicate events (idempotency)
// 2. Match event against active rules in Redis
// 3. Query Neo4j for affected users based on rule targeting
// 4. Execute rule actions (award points/badges)
// 5. Trigger async Brain processing if needed for complex rules
func (ep *EventProcessor) ProcessEvent(ctx context.Context, event models.MatchEvent) error {
	// Step 1: Check for duplicate event (idempotency)
	alreadyProcessed, err := ep.redisClient.IsEventProcessed(ctx, event.EventID)
	if err != nil {
		log.Printf("Warning: failed to check event idempotency: %v", err)
	}
	if alreadyProcessed {
		log.Printf("Skipping duplicate event: %s", event.EventID)
		return nil
	}

	// Step 2: Match event against active rules
	matchedRules := ep.MatchRulesWithEvent(ctx, event)
	if len(matchedRules) == 0 {
		log.Printf("No rules matched for event: %s", event.EventID)
		// Still mark as processed to avoid reprocessing
		ep.redisClient.MarkEventProcessed(ctx, event.EventID, 24*time.Hour)
		return nil
	}

	log.Printf("Matched %d rules for event: %s", len(matchedRules), event.EventID)

	// Step 3 & 4: For each matched rule, find affected users and execute actions
	for _, rule := range matchedRules {
		// Check cooldown to prevent duplicate triggers
		inCooldown, err := ep.redisClient.CheckCooldown(ctx, rule.RuleID, event.MatchID, event.PlayerID)
		if err != nil {
			log.Printf("Warning: failed to check cooldown: %v", err)
		}
		if inCooldown {
			log.Printf("Rule %s is in cooldown, skipping", rule.RuleID)
			continue
		}

		// Query Neo4j for affected users
		affectedUsers := ep.QueryAffectedUsersFromGraph(ctx, rule, event)
		if len(affectedUsers) == 0 {
			log.Printf("No users found for rule: %s", rule.RuleID)
			continue
		}

		log.Printf("Found %d affected users for rule: %s", len(affectedUsers), rule.RuleID)

		// Execute rule actions
		if err := ep.ExecuteRuleActions(ctx, affectedUsers, rule, event); err != nil {
			log.Printf("Error executing rule actions: %v", err)
			continue
		}

		// Set cooldown after successful execution
		if rule.CooldownSeconds > 0 {
			ep.redisClient.SetCooldown(ctx, rule.RuleID, event.MatchID, event.PlayerID,
				time.Duration(rule.CooldownSeconds)*time.Second)
		}
	}

	// Mark event as processed
	if err := ep.redisClient.MarkEventProcessed(ctx, event.EventID, 24*time.Hour); err != nil {
		log.Printf("Warning: failed to mark event processed: %v", err)
	}

	return nil
}

// MatchRulesWithEvent queries Redis for rules matching the given event.
// It uses the rule matcher's in-memory cache for fast rule lookups.
//
// Returns a list of rules that match the event based on:
// - Event type (goal, corner, foul, etc.)
// - Simple conditions (threshold, value comparisons)
// - Aggregation conditions (counts over time windows)
// - Temporal conditions (events within time periods)
func (ep *EventProcessor) MatchRulesWithEvent(ctx context.Context, event models.MatchEvent) []models.Rule {
	// Use the rule matcher to find matching rules
	if ep.ruleMatcher == nil {
		log.Println("RuleMatcher not initialized")
		return nil
	}

	rules, err := ep.ruleMatcher.FindMatchingRules(ctx, event)
	if err != nil {
		log.Printf("Error finding matching rules: %v", err)
		return nil
	}

	return rules
}

// QueryAffectedUsersFromGraph queries Neo4j Knowledge Graph to find users
// affected by a rule based on the event context.
//
// The Brain/LLM layer maintains the Knowledge Graph with user relationships,
// preferences, and historical data. This function queries that graph to find
// users who should receive rewards based on:
// - Team supporters for the event's team
// - Match participants
// - Active players in the match
// - Player followers
// - Team followers
// - Achievement progress
func (ep *EventProcessor) QueryAffectedUsersFromGraph(ctx context.Context, rule models.Rule, event models.MatchEvent) []string {
	if ep.neo4jClient == nil {
		log.Println("Neo4j client not initialized")
		return nil
	}

	// Use the target users configuration from the rule
	if rule.TargetUsers.QueryPattern == "" {
		// Default to active players in match
		rule.TargetUsers.QueryPattern = "active_players"
	}

	result, err := ep.neo4jClient.QueryAffectedUsers(
		ctx,
		event.MatchID,
		event.TeamID,
		event.PlayerID,
		rule.TargetUsers.QueryPattern,
		rule.TargetUsers.Params,
	)

	if err != nil {
		log.Printf("Error querying affected users: %v", err)
		return nil
	}

	return result.UserIDs
}

// ExecuteRuleActions executes all actions defined in a rule for the affected users.
// This is the "Muscle" layer in action - synchronously awarding points and badges.
//
// Actions are executed with idempotency checks to prevent duplicate awards.
// Each action is recorded in Neo4j for auditability.
//
// Supported action types:
// - "award_points": Add points to user accounts
// - "grant_badge": Award a badge to users
// - "send_notification": Trigger notification (future enhancement)
func (ep *EventProcessor) ExecuteRuleActions(ctx context.Context, users []string, rule models.Rule, event models.MatchEvent) error {
	if ep.rewardLayer == nil {
		return fmt.Errorf("reward layer not initialized")
	}

	for _, userID := range users {
		for _, action := range rule.Actions {
			// Check action idempotency to prevent duplicate executions
			alreadyProcessed, err := ep.redisClient.IsActionProcessed(
				ctx, event.EventID, rule.RuleID, userID, action.ActionType)
			if err != nil {
				log.Printf("Warning: failed to check action idempotency: %v", err)
			}
			if alreadyProcessed {
				log.Printf("Skipping duplicate action: event=%s rule=%s user=%s action=%s",
					event.EventID, rule.RuleID, userID, action.ActionType)
				continue
			}

			// Execute the action through the reward layer
			if err := ep.rewardLayer.ExecuteRewardAction(ctx, userID, rule.RuleID, &action, &event); err != nil {
				log.Printf("Error executing action for user %s: %v", userID, err)
				continue
			}

			// Mark action as processed
			if err := ep.redisClient.MarkActionProcessed(
				ctx, event.EventID, rule.RuleID, userID, action.ActionType, 24*time.Hour); err != nil {
				log.Printf("Warning: failed to mark action processed: %v", err)
			}

			log.Printf("Executed action %s for user %s (rule: %s)", action.ActionType, userID, rule.RuleID)
		}
	}

	return nil
}

// TriggerBrainProcessing triggers asynchronous Brain (LLM/Knowledge Graph) processing
// for complex rules that require inference or graph updates.
//
// This is called after Muscle processing completes for rules that need:
// - Complex user targeting (ML-based recommendations)
// - Knowledge graph updates (user preference changes)
// - LLM-generated insights or summaries
// - Async badge criteria evaluation
func (ep *EventProcessor) TriggerBrainProcessing(ctx context.Context, event models.MatchEvent, rules []models.Rule) error {
	// This is a placeholder for async Brain processing integration.
	// In production, this would:
	// 1. Publish an event to a separate topic for async processing
	// 2. Or call the LLM client for complex rule evaluation
	// 3. Or update the Knowledge Graph asynchronously
	//
	// For now, this serves as a documentation of the Brain/Muscle separation.
	log.Printf("Brain processing triggered for event %s (rules: %d) - async processing not yet implemented",
		event.EventID, len(rules))
	return nil
}

// StartEventProcessor starts the event processor with configuration
func StartEventProcessor(ctx context.Context, cfg *config.Config) error {
	// Initialize dependencies
	redisClient, err := redis.NewClient(&cfg.Redis)
	if err != nil {
		return fmt.Errorf("failed to create Redis client: %w", err)
	}

	neo4jClient, err := neo4j.NewClient(&cfg.Neo4j)
	if err != nil {
		return fmt.Errorf("failed to create Neo4j client: %w", err)
	}

	// Initialize rule matcher
	ruleMatcher := engine.NewRuleMatcher(redisClient)

	// Initialize reward layer
	rewardLayer := engine.NewRewardLayer(redisClient, neo4jClient, nil)

	// Create event processor
	processor := NewEventProcessor(
		redisClient,
		neo4jClient,
		ruleMatcher,
		rewardLayer,
		&cfg.Kafka,
	)

	// Start consuming events
	return processor.MatchEventConsumer(ctx)
}
