---
sidebar_position: 1
---

# Analytics Summary

Get summary statistics including total users, badges, points, and active rules.

**Endpoint:** `GET /api/v1/analytics/summary`

**Auth:** JWT

**Response:**
```json
{
  "total_users": 1250,
  "total_badges": 45,
  "badge_catalog_count": 30,
  "active_users": 890,
  "active_rules": 12,
  "points_distributed": 156780,
  "events_processed": 8934
}
```

See [Workflow: Read Analytics](../workflows/read-analytics/) for detailed documentation.