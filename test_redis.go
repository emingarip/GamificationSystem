package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
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
		
		fmt.Printf("Raw JSON length for %s: %d\n", k, len(val))
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(val), &parsed)
		if err != nil {
			fmt.Printf("Unmarshal Error for %s: %v\n", k, err)
		} else {
			fmt.Printf("Parsed JSON IsActive: %v\n", parsed["is_active"])
		}
	}
}
