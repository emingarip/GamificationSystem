---
sidebar_position: 4
---

# Test Event (Execute) Workflow

## Purpose (Amaç)

Gerçek bir event'i işleyerek rule engine'ı tetikler ve action'ları (puan, badge) gerçekten uygular.

## When to Use (Ne Zaman Kullanılır)

- Gerçek maç event'lerini işlemek
- Otomatik puan dağıtımı
- Badge kazanma olaylarını tetiklemek

## Who Uses (Kim Kullanır)

- Maç simülasyon sistemleri
- Canlı maç verisi işleyen backend sistemler
- Webhook'lar

## Auth Requirement

**Gerekli** - JWT token + Admin rolü

## Side Effects (Yan Etki)

- **Veri yazar mı?** Evet - Neo4j ve Redis güncellenir
- Kullanıcı puanları güncellenir
- Badge'ler verilir
- Leaderboard güncellenir

## ⚠️ Important Warning

Bu endpoint gerçek veri değiştirir! **Önce dry_run ile test edin.**

```bash
# Önce dry_run ile test et
curl -X POST "http://localhost:3000/api/v1/events/test?dry_run=true" ...

# Sonra execute et
curl -X POST "http://localhost:3000/api/v1/events/test?dry_run=false" ...
```

## Example Request

```bash
curl -X POST "http://localhost:3000/api/v1/events/test?dry_run=false" \
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
        "goal_type": "penalty"
      }
    },
    "dry_run": false
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
    },
    {
      "rule_id": "rule_penalty_scorer",
      "name": "Penalty Scorer",
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
    },
    {
      "action_type": "award_points",
      "params": {
        "points": 15
      }
    }
  ],
  "executed": true
}
```

## What Happens During Execution

1. Event tipine göre kurallar filtre edilir
2. Koşullar değerlendirilir (evaluation_type'a göre)
3. Eşleşen kurallar için action'lar belirlenir
4. Puanlar kullanıcı hesabına eklenir (Neo4j)
5. Badge'ler kullanıcıya atanır (Neo4j)
6. Leaderboard güncellenir (Redis)
7. Action history kaydedilir

## Common Errors (Sık Hata Nedenleri)

| Hata Kodu | Mesaj | Çözüm |
|-----------|-------|-------|
| 400 | Invalid event data | Event formatını kontrol edin |
| 500 | Failed to process event | Sunucu logs kontrol edin |

## Best Practices

1. **Always test with dry_run first**
2. Event'leri sıralı işleyin (timestamp order)
3. Duplicate event'leri önlemek için event_id kullanın
4. Error handling implement edin