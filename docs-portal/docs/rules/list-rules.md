---
sidebar_position: 1
---

# List Rules

Get all gamification rules, optionally filtered by event type.

**Endpoint:** `GET /api/v1/rules`

**Auth:** None (public)

**Query Parameters:**
- `event_type` (optional) - Filter by event type (goal, foul, etc.)

**Example:**
```bash
curl http://localhost:3000/api/v1/rules
```

**Response:**
```json
{
  "rules": [...],
  "count": 5
}
```

See [Workflow: Create Rule](../workflows/create-rule/) for detailed documentation.