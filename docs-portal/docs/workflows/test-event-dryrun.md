---
sidebar_position: 3
---

# Test Event (Dry Run) Workflow

## Purpose (Amaç)

Bir event'in rule engine tarafından nasıl işleneceğini test eder. `dry_run=true` kullanıldığında hiçbir action (puan, badge) verilmez - sadece hangi kuralların eşleşeceği döner.

## When to Use (Ne Zaman Kullanılır)

- Yeni oluşturulan kuralları test etmek
- Event'in hangi kuralları tetikleyeceğini görmek
- Üretim ortamında değişiklik yapmadan önce test etmek

## Who Uses (Kim Kullanır)

- Admin panel kullanıcıları (rule test)
- QA mühendisleri
- CI/CD test süreçleri

## Auth Requirement

**Gerekli** - JWT token + Admin rolü

## Side Effects (Yan Etki)

- **Veri yazar mı?** Hayır - Dry run mode'da veri değiştirilmez
- Sadece evaluation yapılır, action'lar execute edilmez

## Important: dry_run Parameter

`dry_run` parametresi query string veya body içinde gönderilebilir:

**Priority**: Query param > Body > Default (true)

```bash
# Query param ile
curl -X POST "http://localhost:3000/api/v1/events/test?dry_run=true" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{ "event": {...} }'

# Body ile
curl -X POST "http://localhost:3000/api/v1/events/test" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{ "event": {...}, "dry_run": true }'
```

## Example Request

```bash
curl -X POST "http://localhost:3000/api/v1/events/test?dry_run=true" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "event": {
      "event_id": "evt_001",
      "event_type": "goal",
      "match_id": "match_123",
      "team_id": "team_a",
      "player_id": "player_456",
      "minute": 45,
      "timestamp": "2024-01-15T10:30:00Z",
      "metadata": {
        "scorer_id": "player_456",
        "assist_player": "player_789",
        "goal_type": "open_play"
      }
    },
    "dry_run": true
  }'
```

## Example Response

```json
{
  "matches": [
    {
      "rule_id": "rule_goal_scorer",
      "name": "Goal Scorer",
      "matched": true
    }
  ],
  "affected_users": ["player_456"],
  "actions": [
    {
      "action_type": "award_points",
      "params": {
        "points": 10
      }
    },
    {
      "action_type": "grant_badge",
      "params": {
        "badge_id": "badge_first_goal"
      }
    }
  ],
  "executed": false
}
```

## Response Fields

| Field | Type | Description |
|-------|------|-------------|
| matches | array | Eşleşen kurallar |
| affected_users | array | Etkilenen kullanıcı ID'leri |
| actions | array | Yapılacak action'lar |
| executed | boolean | Action'lar execute edildi mi |

## Common Errors (Sık Hata Nedenleri)

| Hata Kodu | Mesaj | Çözüm |
|-----------|-------|-------|
| 400 | Invalid request body | Event formatını kontrol edin |
| 500 | Rule engine not available | Sunucu hatası |

## Dry Run vs Execute

| Özellik | Dry Run | Execute |
|---------|---------|---------|
| Puan verir | ❌ | ✅ |
| Badge verir | ❌ | ✅ |
| Log tutar | ✅ | ✅ |
| Güvenli | ✅ | ⚠️ |

**Önce dry_run ile test edin, sonra execute edin!**