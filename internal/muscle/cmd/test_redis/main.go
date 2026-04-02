package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
	"gamification/models"
)

func main() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx := context.Background()

	keys, err := client.ZRevRange(ctx, "rules:by_type:ad_reward", 0, -1).Result()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Rule IDs:", keys)

	for _, k := range keys {
		val, err := client.Get(ctx, "rule:"+k).Result()
		if err != nil {
			fmt.Printf("Get error for %s: %v\n", k, err)
			continue
		}
		
		fmt.Printf("Raw Data Length for %s: %d\n", k, len(val))

		var rule map[string]interface{}
		json.Unmarshal([]byte(val), &rule)
		fmt.Printf("JSON map IsActive for %s: %v\n", k, rule["is_active"])

		var typedRule models.Rule
		err = json.Unmarshal([]byte(val), &typedRule)
		if err != nil {
			fmt.Printf("Model Unmarshal Error for %s: %v\n", k, err)
		} else {
			fmt.Printf("Model Unmarshaled Successfully. Name: %s, EventType: %s, IsActive: %v\n", typedRule.Name, typedRule.EventType, typedRule.IsActive)
		}
	}
}
