---
sidebar_position: 2
---

# Create Badge

Create a new badge that can be earned by users.

**Endpoint:** `POST /api/v1/badges`

**Auth:** JWT + Admin

**Example:**
```json
{
  "id": "badge_first_goal",
  "name": "First Goal",
  "description": "Score your first goal",
  "points": 50,
  "icon": "🏆",
  "rarity": "common"
}