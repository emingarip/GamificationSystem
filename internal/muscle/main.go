// @title Gamification & Sports Analytics API
// @version 1.0.0
// @description AI-Native Gamification Platform with Knowledge Graph
// @basePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
// Package main provides the Gamification API server
//
//go:generate swag init -g main.go -o docs
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gamification/api"
	"gamification/config"
	"gamification/engine"
	"gamification/kafka"
	"gamification/llm"
	"gamification/metrics"
	"gamification/models"
	"gamification/neo4j"
	"gamification/redis"
	"gamification/seed"
	"gamification/websocket"

	_ "gamification/docs"
)

func main() {
	// Parse command line flags
	seedMode := flag.Bool("seed", false, "Run seed data population to Neo4j")
	clearData := flag.Bool("clear", false, "Clear existing data before seeding (only with --seed)")
	verbose := flag.Bool("v", false, "Verbose output")
	logLevel := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	flag.Parse()

	// Initialize logging
	level := metrics.LogLevel(*logLevel)
	logger := metrics.Init(level, nil)
	ctx := context.Background()

	// Initialize Prometheus metrics
	metrics.NewMetrics()

	logger.Info(ctx, "Starting Muscle Layer - High-Performance Rule Engine...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error(ctx, fmt.Sprintf("Failed to load configuration: %v", err))
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Run seed mode if requested
	if *seedMode {
		if err := seed.Run(cfg, *verbose, *clearData); err != nil {
			logger.Error(ctx, fmt.Sprintf("Seed failed: %v", err))
			log.Fatalf("Seed failed: %v", err)
		}
		return
	}

	// Initialize Redis client
	logger.Info(ctx, "Connecting to Redis...")
	redisClient, err := redis.NewClient(&cfg.Redis)
	if err != nil {
		logger.Error(ctx, fmt.Sprintf("Failed to connect to Redis: %v", err))
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	logger.Info(ctx, "Redis connected")

	// Seed default event types to Redis registry (idempotent - safe to call on every startup)
	logger.Info(ctx, "Seeding default event types to registry...")
	if err := redisClient.SeedDefaultEventTypes(ctx); err != nil {
		logger.Warn(ctx, fmt.Sprintf("Warning: failed to seed event types: %v", err))
	} else {
		logger.Info(ctx, "Default event types seeded successfully")
	}

	// Initialize Neo4j client
	logger.Info(ctx, "Connecting to Neo4j...")
	neo4jClient, err := neo4j.NewClient(&cfg.Neo4j)
	if err != nil {
		logger.Error(ctx, fmt.Sprintf("Failed to connect to Neo4j: %v", err))
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer neo4jClient.Close()
	logger.Info(ctx, "Neo4j connected")

	// Initialize API server
	logger.Info(ctx, "Initializing API server...")

	// Create rule engine first
	ruleEngine := engine.NewRuleEngine(cfg, redisClient, neo4jClient)

	// Create WebSocket server first
	wsServer := websocket.NewServer(cfg)
	if err := wsServer.Start(); err != nil {
		logger.Error(ctx, fmt.Sprintf("Failed to start WebSocket server: %v", err))
		log.Fatalf("Failed to start WebSocket server: %v", err)
	}
	defer wsServer.Stop()

	// Create reward layer and connect to engine
	rewardLayer := engine.NewRewardLayer(redisClient, neo4jClient, wsServer)
	ruleEngine.SetRewardLayer(rewardLayer)

	// Create API server with rule engine
	apiServer := api.NewServer(cfg, redisClient, neo4jClient, ruleEngine)
	apiAddr := cfg.ServerAddr()
	if apiAddr == "" {
		apiAddr = ":8080"
	}

	// Start API server in a goroutine
	go func() {
		logger.Info(ctx, fmt.Sprintf("Starting API server on %s", apiAddr))
		if err := apiServer.Start(apiAddr); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, fmt.Sprintf("API server error: %v", err))
		}
	}()

	// Start rule engine workers
	ctx, cancel := context.WithCancel(context.Background())
	ruleEngine.Start(ctx)
	defer func() {
		cancel()
		ruleEngine.Stop()
	}()

	// Create Kafka consumer
	kafkaConsumer := kafka.NewConsumer(&cfg.Kafka)
	defer kafkaConsumer.Close()

	// Set up Kafka consumer to broadcast badge events via WebSocket
	kafkaConsumer.RegisterBadgeHandler(func(userID, badgeID, badgeName, description string, points int) {
		logger.Info(ctx, fmt.Sprintf("Broadcasting badge earned to user %s: %s", userID, badgeName))
		wsServer.SendBadgeEarned(userID, badgeID, badgeName, description, points)

		// Record badge grant metrics
		m := metrics.GetMetrics()
		m.RecordBadgeGranted(badgeID, badgeName, userID)
	})

	// Register event handler
	kafkaConsumer.RegisterHandler(func(ctx context.Context, event *models.MatchEvent) error {
		logger.Info(ctx, fmt.Sprintf("Received event: %s - %s", event.EventType, event.EventID))

		startTime := time.Now()
		result := ruleEngine.ProcessMatchEvent(ctx, event, false)

		// Record rule evaluation metrics
		m := metrics.GetMetrics()
		for _, ruleResult := range result.TriggeredRules {
			m.RecordRuleEvaluated(string(event.EventType), ruleResult.Rule.RuleID)
			m.RecordRuleEvalDuration(string(event.EventType), ruleResult.Rule.RuleID, ruleResult.EvalTimeMs)
			if ruleResult.Matched {
				m.RecordRuleMatched(string(event.EventType), ruleResult.Rule.RuleID)
				logger.Info(ctx, fmt.Sprintf("Rule triggered: %s -> Users: %v (%.2fms)",
					ruleResult.Rule.Name, ruleResult.Users, ruleResult.EvalTimeMs))
			}
		}

		// Record Kafka message metrics
		m.RecordKafkaMessageProcessed("events", "0", 0, time.Since(startTime))

		if !result.Success {
			logger.Error(ctx, fmt.Sprintf("Error processing event: %v", result.Error))
			m.RecordKafkaError("events", "0", "processing_error")
			return result.Error
		}

		if result.TotalTimeMs > 10 {
			logger.Warn(ctx, fmt.Sprintf("Warning: Total processing time %.2fms exceeds target of 10ms", result.TotalTimeMs))
		}

		return nil
	})

	// Start Kafka consumer in a goroutine
	go func() {
		if err := kafkaConsumer.Start(ctx); err != nil && err != context.Canceled {
			logger.Error(ctx, fmt.Sprintf("Kafka consumer error: %v", err))
		}
	}()

	logger.Info(ctx, "Muscle Layer started successfully")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info(ctx, "Shutting down...")
}

// Example functions demonstrating usage

// ExampleProcessEvent shows how to process an event directly
func ExampleProcessEvent(eng *engine.RuleEngine) {
	ctx := context.Background()

	event := &models.MatchEvent{
		EventID:   "evt-001",
		EventType: models.EventTypeGoal,
		MatchID:   "match-123",
		TeamID:    "team-a",
		PlayerID:  "player-456",
		Minute:    45,
		Timestamp: time.Now(),
		Metadata:  []byte(`{"scorer_id": "player-456", "goal_type": "open_play"}`),
	}

	result := eng.ProcessMatchEvent(ctx, event, false)
	fmt.Printf("Processed in %.2fms, triggered %d rules\n",
		result.TotalTimeMs, len(result.TriggeredRules))
}

// ExampleRuleDefinition shows a sample rule structure
func ExampleRuleDefinition() *models.Rule {
	return &models.Rule{
		RuleID:      "rule-hat-trick",
		Name:        "Hat Trick Bonus",
		Description: "Award bonus points for scoring 3 goals",
		EventType:   models.EventTypeGoal,
		IsActive:    true,
		Priority:    100,
		Conditions: []models.RuleCondition{
			{
				Field:          "consecutive_count",
				Operator:       ">=",
				Value:          3,
				EvaluationType: "aggregation",
			},
		},
		TargetUsers: models.TargetUsers{
			QueryPattern: "team_supporters",
			Params:       map[string]string{},
		},
		Actions: []models.RuleAction{
			{
				ActionType: "award_points",
				Params:     map[string]any{"points": 500.0},
			},
			{
				ActionType: "grant_badge",
				Params:     map[string]any{"badge_id": "hat_trick"},
			},
		},
		CooldownSeconds: 3600,
	}
}

// ExampleLoadRule demonstrates loading a rule into Redis
func ExampleLoadRule(redisClient *redis.Client, rule *models.Rule) error {
	ctx := context.Background()
	return redisClient.SaveRule(ctx, rule)
}

// ExampleLLMTransform demonstrates using the LLM client to transform natural language rules
// This is the "Brain Layer" integration test that transforms natural language to JSON
func ExampleLLMTransform() {
	// Create LLM client with vLLM configuration
	llmClient := llm.NewClient(
		"localhost",  // LLM host
		8000,         // vLLM port
		"llama-3-8b", // Model name
		llm.LLMClientConfig{
			Temperature: 0.1,
			TopP:        0.9,
			MaxTokens:   2048,
			Timeout:     30 * time.Second,
			MaxRetries:  3,
			RetryDelay:  1 * time.Second,
		},
	)

	// Test cases with Turkish and English natural language rules
	testCases := []string{
		// Turkish test cases
		"Derbide 3 korner üst üste olursa takımın taraftarlarına 50 puan ver",
		"Maçın son 5 dakikasında gol olursa tüm izleyenlere bildirim gönder",
		"Her maçta ev sahibi takım attılan her golden sonra taraftarlara 10 puan ver",

		// English test cases
		"When a player scores 2 goals in the first half, give them the 'First Half Striker' badge",
		"Award 25 points to team supporters when their team scores the first goal in the league",
		"If the team wins a derby match, give them 100 points and the 'Derby Champion' badge",
	}

	ctx := context.Background()

	for i, testCase := range testCases {
		fmt.Printf("\n--- Test Case %d ---", i+1)
		fmt.Printf("\nInput: %s\n", testCase)

		// Transform natural language to JSON rule
		rule, err := llmClient.TransformRule(ctx, testCase)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Print the resulting rule
		fmt.Printf("Rule ID: %s\n", rule.RuleID)
		fmt.Printf("Name: %s\n", rule.Name)
		fmt.Printf("Event Type: %s\n", rule.EventType)
		fmt.Printf("Priority: %d\n", rule.Priority)
		fmt.Printf("Cooldown: %ds\n", rule.CooldownSec)
		fmt.Printf("Conditions: %d\n", len(rule.Conditions))
		for j, cond := range rule.Conditions {
			fmt.Printf("  [%d] %s %s %v (%s)\n", j, cond.Field, cond.Operator, cond.Value, cond.EvaluationType)
		}
		fmt.Printf("Target Users: %s\n", rule.TargetUsers.QueryPattern)
		fmt.Printf("Actions: %d\n", len(rule.Actions))
		for j, action := range rule.Actions {
			fmt.Printf("  [%d] %s: %v\n", j, action.ActionType, action.Params)
		}
	}
}

// ExampleLLMHealthCheck demonstrates checking if the LLM service is healthy
func ExampleLLMHealthCheck() {
	llmClient := llm.NewClient(
		"localhost",  // LLM host
		8000,         // vLLM port
		"llama-3-8b", // Model name
		llm.LLMClientConfig{
			Timeout: 10 * time.Second,
		},
	)

	ctx := context.Background()

	if err := llmClient.CheckHealth(ctx); err != nil {
		log.Printf("LLM service unhealthy: %v", err)
	} else {
		log.Println("LLM service is healthy")
	}
}
