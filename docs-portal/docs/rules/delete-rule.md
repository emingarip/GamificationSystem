---
sidebar_position: 4
---

# Delete Rule

Delete a gamification rule by ID.

**Endpoint:** `DELETE /api/v1/rules/{id}`

**Auth:** JWT + Admin

**Response:**
```json
{
  "id": "rule_123",
  "message": "Rule deleted successfully"
}