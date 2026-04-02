---
sidebar_position: 2
---

# Quick Start

This guide will help you get started with the Gamification API in just a few minutes.

## Prerequisites

- Running Gamification server (default: `http://localhost:3000`)
- cURL or Postman for testing API calls

## Step 1: Login as Admin

First, authenticate to get your access token:

```bash
curl -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@gamification.local",
    "password": "admin123"
  }'
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "admin_001",
    "name": "Admin User",
    "email": "admin@gamification.local",
    "level": 1,
    "points": 0,
    "badges": []
  }
}
```

Save the `token` value - you'll need it for subsequent requests.

## Step 2: List Available Rules

```bash
curl -X GET http://localhost:3000/api/v1/rules \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

## Step 3: Create a New Rule

```bash
curl -X POST http://localhost:3000/api/v1/rules \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "rule_goal_scorer",
    "name": "Goal Scorer",
    "description": "Award 10 points for scoring a goal",
    "event_type": "goal",
    "points": 10,
    "enabled": true,
    "conditions": [
      {
        "field": "event_type",
        "operator": "==",
        "value": "goal",
        "evaluation_type": "simple"
      }
    ]
  }'
```

## Step 4: Test Event (Dry Run)

Test how an event would be processed without executing actions:

```bash
curl -X POST "http://localhost:3000/api/v1/events/test?dry_run=true" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "event": {
      "event_id": "evt_001",
      "event_type": "goal",
      "match_id": "match_123",
      "team_id": "team_a",
      "player_id": "player_456",
      "minute": 45,
      "timestamp": "2024-01-15T10:30:00Z"
    },
    "dry_run": true
  }'
```

## Step 5: Update User Points

```bash
curl -X PUT http://localhost:3000/api/v1/users/user_001/points \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "points": 100,
    "operation": "add"
  }'
```

## Step 6: Get Analytics Summary

```bash
curl -X GET http://localhost:3000/api/v1/analytics/summary \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

## Next Steps

- Explore [Authentication](./authentication) for detailed JWT handling
- Follow the [Workflow Guides](./workflows) for common use cases
- Check the [API Reference](/api-reference) for complete endpoint documentation
