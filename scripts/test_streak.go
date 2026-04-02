package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/redis/go-redis/v9"
)

func main() {
	// Use go-redis directly instead of using internal model to manipulate private keys
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}

	playerID := "test_player_1"
	eventType := "ad_reward"

	// Mock previous 4 global counts
	globalKey := fmt.Sprintf("event_count:global:%s:%s", playerID, eventType)
	
	// Set 4 global count
	if err := rdb.Set(ctx, globalKey, 4, 0).Err(); err != nil {
		log.Fatalf("Failed to set global count: %v", err)
	}
	
	log.Printf("Successfully mocked 4 global ad counts for %s", playerID)
	
	// Trigger the 5th ad via API
	payload := []byte(`{
		"user_id": "test_player_1",
		"event_type": "ad_reward",
		"match_id": "",
		"team_id": ""
	}`)
	
	resp, err := http.Post("http://localhost:3000/api/v1/events", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Fatalf("Failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	log.Printf("API Response (%d): %s", resp.StatusCode, string(body))
}
