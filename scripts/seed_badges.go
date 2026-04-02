package main

import (
	"context"
	"fmt"
	"log"

	"gamification/config"
	"gamification/neo4j"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Force localhost to ensure we connect to Docker's exposed port 7688 from the host
	cfg.Neo4j.URI = "neo4j://localhost:7688"
	cfg.Neo4j.Username = "neo4j"
	cfg.Neo4j.Password = "neo4j_password"

	client, err := neo4j.NewClient(&cfg.Neo4j)
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	badges := []*neo4j.Badge{
		{
			ID:          "badge_first_login",
			Name:        "Taze Kan",
			Description: "SportsApp ailesine hoş geldin! İlk girişinde bu rozeti kazandın.",
			Icon:        "person_add",
			Points:      50,
			Category:    "Başlangıç",
			Metric:      "global_count:daily_login",
			Target:      1,
		},
		{
			ID:          "badge_streak_7",
			Name:        "Seri Ustası",
			Description: "7 gün boyunca her gün giriş yaparak gerçek bir taraftar olduğunu kanıtladın.",
			Icon:        "local_fire_department",
			Points:      250,
			Category:    "Bağlılık",
			Metric:      "daily_streak",
			Target:      7,
		},
		{
			ID:          "badge_social",
			Name:        "Sosyal Kelebek",
			Description: "Arkadaşlarını platforma davet ederek ağını genişlettin.",
			Icon:        "loyalty",
			Points:      100,
			Category:    "Sosyal",
			Metric:      "global_count:invite_friend",
			Target:      1,
		},
	}

	for _, b := range badges {
		id, err := client.CreateBadge(ctx, b)
		if err != nil {
			log.Printf("Failed to create badge %s: %v", b.Name, err)
		} else {
			fmt.Printf("Successfully seeded badge: %s (ID: %s)\n", b.Name, id)
		}
	}
}
