package main

import (
	"context"
	"fmt"
	"gamification/config"
	"gamification/engine"
	"gamification/models"
	"gamification/neo4j"
	gredis "gamification/redis"
	"log"
	"time"
)

func main() {
	cfg, _ := config.Load()
	redisClient, _ := gredis.NewClient(&cfg.Redis)
	defer redisClient.Close()
	ruleEngine := engine.NewRuleEngine(cfg, redisClient, (*neo4j.Client)(nil))

	ctx := context.Background()

	// 1. Get the rule
	rule, err := redisClient.GetRuleByID(ctx, "rule_5_ad_watch_milestone")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Rule: %+v\nConditions: %+v\n\n", rule.Name, rule.Conditions)

	// 2. Mock event matching
	event := &models.MatchEvent{
		EventID:   "test_script_evt",
		EventType: "ad_reward",
		PlayerID:  "test_player_ad123",
		Timestamp: time.Now(),
	}

	// Calculate counts to emulate engine logic
	count, _ := redisClient.GetGlobalEventCount(ctx, event.PlayerID, event.EventType)
	fmt.Printf("Global count via Redis: %d\n", count)

	// 3. Evaluate using the engine's internals (if possible) or directly MatchRules
	rules, err := ruleEngine.MatchRules(ctx, event)
	if err != nil {
		log.Fatal(err)
	}
	
	for _, r := range rules {
		fmt.Printf("Matched rule: %s\n", r.Name)
		fmt.Printf("  Actions: %+v\n", r.Actions)
	}
}
