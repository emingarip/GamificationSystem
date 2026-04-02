---
sidebar_position: 6
---

# Assign Badge to User Workflow

## Purpose (Amaç)

Bir kullanıcıya manuel olarak badge atar. Bu, otomatik kazanma yerine admin tarafından verilen badge'ler için kullanılır.

## When to Use (Ne Zaman Kullanılır)

- Özel başarılar için manuel badge verme
- Manual düzeltmeler
- Event ödülleri
- Lifetime achievement'lar

## Who Uses (Kim Kullanır)

- Admin panel kullanıcıları
- Community manager'lar
- Manual reward sistemleri

## Auth Requirement

**Gerekli** - JWT token + Admin rolü

## Side Effects (Yan Etki)

- **Veri yazar mı?** Evet - Neo4j ve Redis güncellenir
- Badge kullanıcıya atanır
- Activity history kaydedilir
- Redis cache güncellenir

## Pre-requisite

Önce badge'in sistemde mevcut olması gerekir. Badge'leri listeleyin:

```bash
GET /api/v1/badges
```

## Example Request

```bash
curl -X POST http://localhost:3000/api/v1/users/user_001/badges \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "badge_id": "badge_champion"
  }'
```

## Example Response

```json
{
  "user_id": "user_001",
  "badge_id": "badge_champion",
  "message": "Badge assigned successfully"
}
```

## Common Errors (Sık Hata Nedenleri)

| Hata Kodu | Mesaj | Çözüm |
|-----------|-------|-------|
| 400 | Badge ID is required | badge_id girin |
| 404 | User not found | Geçerli user ID girin |
| 500 | Failed to assign badge | Badge mevcut mu kontrol edin |

## Best Practices

1. Badge kataloğunu önceden kontrol edin
2. Aynı badge'i tekrar atamayı kontrol edin
3. Kullanıcının mevcut badge'lerini görmek için user profile kontrol edin
4. Atama nedenini not edin (future: reason parametresi)