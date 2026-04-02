---
sidebar_position: 7
---

# Read Analytics Workflow

## Purpose (Amaç)

Sistem genelinde istatistikleri ve analitik verileri getirir. Dashboard için gerekli tüm metric'leri tek seferde alır.

## When to Use (Ne Zaman Kullanılır)

- Admin dashboard oluşturma
- Sistem sağlığını kontrol etme
- Raporlama
- Kullanıcı aktivite analizi

## Who Uses (Kim Kullanır)

- Admin panel dashboard
- BI sistemleri
- Monitoring sistemleri

## Auth Requirement

**Gerekli** - JWT token

## Side Effects (Yan Etki)

- **Veri yazar mı?** Hayır - Sadece okuma
- Veritabanlarından (Neo4j, Redis) aggregation sorgusu yapılır

## Analytics Summary

Temel sistem metric'lerini getirir:

```bash
curl -X GET http://localhost:3000/api/v1/analytics/summary \
  -H "Authorization: Bearer YOUR_TOKEN"
```

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

## Recent Activity

Son aktiviteleri getirir:

```bash
curl -X GET "http://localhost:3000/api/v1/analytics/activity?limit=20" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**Response:**
```json
{
  "activities": [
    {
      "user_id": "user_123",
      "action_type": "award_points",
      "points": 10,
      "reason": "Goal scored in match_456",
      "timestamp": "2024-01-15T14:30:00Z"
    }
  ],
  "count": 1
}
```

## Points History

Belirli bir period için puan geçmişini getirir:

```bash
curl -X GET "http://localhost:3000/api/v1/analytics/points-history?period=day" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**Period Options:** `day`, `week`, `month`

**Response:**
```json
{
  "period": "day",
  "history": [
    {
      "date": "2024-01-15",
      "points": 1500
    },
    {
      "date": "2024-01-14",
      "points": 2300
    }
  ]
}
```

## Common Errors (Sık Hata Nedenleri)

| Hata Kodu | Mesaj | Çözüm |
|-----------|-------|-------|
| 500 | Internal server error | Veritabanı bağlantısını kontrol edin |

## Best Practices

1. Dashboard için summary endpoint kullanın
2. Activity için pagination veya limit kullanın
3. Period parametresini doğru formatta gönderin
4. Cache'lenmiş veriler içinRedis'e bakın