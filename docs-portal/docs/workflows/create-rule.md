---
sidebar_position: 2
---

# Create Rule Workflow

## Purpose (Amaç)

Sistemde yeni bir gamification kuralı oluşturur. Bu kurallar, event'ler tetiklendiğinde otomatik olarak puan ve badge verir.

## When to Use (Ne Zaman Kullanılır)

- Yeni ödül sistemi oluşturma
- Oyunculara belirli actions için puan verme
- Otomatik badge dağıtımı kuralları oluşturma

## Who Uses (Kim Kullanır)

- Admin panel kullanıcıları
- Backend sistemleri (otomatik rule oluşturma)

## Auth Requirement

**Gerekli** - JWT token + Admin rolü

## Side Effects (Yan Etki)

- **Veri yazar mı?** Evet - Redis'e rule kaydedilir
- Redis'te rule list güncellenir

## Example Request

```bash
curl -X POST http://localhost:3000/api/v1/rules \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "rule_goal_scorer",
    "name": "Goal Scorer",
    "description": "Award 10 points for scoring a goal",
    "event_type": "goal",
    "points": 10,
    "multiplier": 1.0,
    "cooldown": 0,
    "enabled": true,
    "conditions": [
      {
        "field": "event_type",
        "operator": "==",
        "value": "goal",
        "evaluation_type": "simple"
      }
    ],
    "rewards": {
      "badge_id": "badge_first_goal"
    }
  }'
```

## Example Response

```json
{
  "id": "rule_goal_scorer",
  "message": "Rule created successfully"
}
```

## Field Description

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | Evet | Unique rule ID |
| name | string | Evet | Rule adı |
| description | string | Hayır | Rule açıklaması |
| event_type | string | Evet | Tetiklenecek event tipi (goal, foul, yellow_card, vs.) |
| points | integer | Hayır | Verilecek puan |
| multiplier | number | Hayır | Puan çarpanı |
| cooldown | integer | Hayır | Tekrar tetikleme arası süre (saniye) |
| enabled | boolean | Hayır | Rule aktif mi |
| conditions | array | Hayır | Koşul dizisi |
| rewards | object | Hayır | Badge rewards |

## Event Types

- `goal` - Gol atma
- `corner` - Korner
- `foul` - Faul
- `yellow_card` - Sarı kart
- `red_card` - Kırmızı kart
- `penalty` - Penaltı
- `offside` - Ofsayt

## Common Errors (Sık Hata Nedenleri)

| Hata Kodu | Mesaj | Çözüm |
|-----------|-------|-------|
| 400 | Rule name is required | name alanını girin |
| 400 | Event type is required | event_type alanını girin |
| 500 | Failed to create rule | Sunucu hatası, tekrar deneyin |

## Best Practices

1. Anlamlı rule ID'ler kullanın (örn: `rule_goal_scorer`)
2. Description ile rule amacını açıklayın
3. Test event ile rule'ı doğrulayın