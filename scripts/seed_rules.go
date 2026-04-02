package main

import (
	"context"
	"log"

	"gamification/config"
	"gamification/models"
	"gamification/redis"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Force localhost to ensure we connect to Docker's exposed port 6379 from the host
	cfg.Redis.Host = "localhost"
	cfg.Redis.Port = 6379

	rdb, err := redis.NewClient(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}
	defer rdb.Close()

	ctx := context.Background()

	// 1. Daily Reward Rule
	dailyRule := &models.Rule{
		RuleID:          "rule_daily_reward",
		Name:            "Daily Login Reward",
		Description:     "Award 50 points for logging in daily",
		EventType:       "daily_login",
		IsActive:        true,
		Priority:        100,
		CooldownSeconds: 86000, 
		Actions: []models.RuleAction{
			{
				ActionType: "award_points",
				Params: map[string]any{
					"points": float64(50),
					"reason": "Daily Reward",
				},
			},
		},
	}

	if err := rdb.SaveRule(ctx, dailyRule); err != nil {
		log.Fatalf("Failed to save daily rule: %v", err)
	}

	log.Printf("Successfully seeded rule: %s", dailyRule.Name)

	// 1b. First Login / Welcome Rule
	welcomeRule := &models.Rule{
		RuleID:          "rule_first_login",
		Name:            "Hoş Geldin Ödülü",
		Description:     "Uygulamaya ilk kez kayıt olup giriş yapan kullanıcılara verilir.",
		EventType:       "daily_login",
		IsActive:        true,
		Priority:        100,
		CooldownSeconds: 0, 
		Conditions: []models.RuleCondition{
			{
				EvaluationType: "aggregation",
				Field:          "global_count",
				Operator:       "==",
				Value:          float64(1),
			},
		},
		Actions: []models.RuleAction{
			{
				ActionType: "award_points",
				Params: map[string]any{"points": float64(500), "reason": "Kayıt Ödülü"},
			},
			{
				ActionType: "grant_badge",
				Params: map[string]any{"badge_id": "badge_first_login", "points": float64(0)},
			},
		},
	}

	if err := rdb.SaveRule(ctx, welcomeRule); err != nil {
		log.Fatalf("Failed to save welcome rule: %v", err)
	}
	log.Printf("Successfully seeded rule: %s", welcomeRule.Name)

	// 2. 7-Day Streak Rule
	streakRule := &models.Rule{
		RuleID:          "rule_7_day_streak",
		Name:            "7-Day Login Streak",
		Description:     "Award 250 points for logging in 7 days in a row",
		EventType:       "daily_login",
		IsActive:        true,
		Priority:        200, 
		CooldownSeconds: 604800, // 7 days cooldown so it only happens once per week
		Conditions: []models.RuleCondition{
			{
				EvaluationType: "aggregation",
				Field:          "daily_streak",
				Operator:       "==",
				Value:          float64(7), // Must be exactly 7
			},
		},
		Actions: []models.RuleAction{
			{
				ActionType: "award_points",
				Params: map[string]any{
					"points": float64(250),
					"reason": "7-Day Streak Bonus",
				},
			},
			{
				ActionType: "grant_badge",
				Params: map[string]any{
					"badge_id": "badge_streak_7",
					"points": float64(0),
				},
			},
		},
	}

	if err := rdb.SaveRule(ctx, streakRule); err != nil {
		log.Fatalf("Failed to save streak rule: %v", err)
	}

	log.Printf("Successfully seeded rule: %s", streakRule.Name)

	// 3. Ad Watch Milestone Rule (Recurring every 5 ads)
	adWatchRule := &models.Rule{
		RuleID:          "rule_5_ad_watch_milestone",
		Name:            "Every 5 Ads Watched Bonus",
		Description:     "Award 250 points for every 5 ads watched",
		EventType:       "ad_reward",
		IsActive:        true,
		Priority:        150, 
		CooldownSeconds: 0, // No cooldown, can happen whenever they hit multiple of 5
		Conditions: []models.RuleCondition{
			{
				EvaluationType: "aggregation",
				Field:          "global_count",
				Operator:       "every",
				Value:          float64(5), // Triggers on 5, 10, 15, etc.
			},
		},
		Actions: []models.RuleAction{
			{
				ActionType: "award_points",
				Params: map[string]any{
					"points": float64(250),
					"reason": "5 Ads Milestone Bonus",
				},
			},
		},
	}

	if err := rdb.SaveRule(ctx, adWatchRule); err != nil {
		log.Fatalf("Failed to save ad watch rule: %v", err)
	}

	log.Printf("Successfully seeded rule: %s", adWatchRule.Name)

	// 4. Basic Ad Watch Rule (Every time)
	basicAdRule := &models.Rule{
		RuleID:          "rule_basic_ad_reward",
		Name:            "Ad Watched Reward",
		Description:     "Award 50 points for watching an ad",
		EventType:       "ad_reward",
		IsActive:        true,
		Priority:        100,
		CooldownSeconds: 0, 
		Actions: []models.RuleAction{
			{
				ActionType: "award_points",
				Params: map[string]any{
					"points": float64(50),
					"reason": "Ad Watch Reward",
				},
			},
		},
	}

	if err := rdb.SaveRule(ctx, basicAdRule); err != nil {
		log.Fatalf("Failed to save basic ad rule: %v", err)
	}

	log.Printf("Successfully seeded rule: %s", basicAdRule.Name)

	// 5. App Invite/Referral Rule
	inviteFriendRule := &models.Rule{
		RuleID:          "rule_invite_friend",
		Name:            "Arkadaş Davet Et",
		Description:     "Uygulamaya arkadaşını başarıyla davet eden kullanıcılara verilen ödül.",
		EventType:       "invite_friend",
		IsActive:        true,
		Priority:        300,
		CooldownSeconds: 0, 
		Actions: []models.RuleAction{
			{
				ActionType: "award_points",
				Params: map[string]any{
					"points": float64(1000),
					"reason": "Arkadaş Daveti Ödülü",
				},
			},
			{
				ActionType: "grant_badge",
				Params: map[string]any{
					"badge_id": "badge_social",
					"points":   float64(0),
				},
			},
		},
	}

	if err := rdb.SaveRule(ctx, inviteFriendRule); err != nil {
		log.Fatalf("Failed to save invite_friend rule: %v", err)
	}

	log.Printf("Successfully seeded rule: %s", inviteFriendRule.Name)
}
