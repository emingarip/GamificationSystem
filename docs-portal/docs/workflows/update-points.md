---
sidebar_position: 5
---

# Update User Points Workflow

## Purpose (Amaç)

Kullanıcının puanını manuel olarak günceller. Üç farklı operation desteklenir: add, subtract, set.

## When to Use (Ne Zaman Kullanılır)

- Manuel puan düzeltme (hata düzeltme)
- Bonus puan verme (promotion, özel gün)
- Ceza puanı kesme
- Belirli bir puan seviyesine set etme (level reset)

## Who Uses (Kim Kullanır)

- Admin panel kullanıcıları
- Destek sistemleri
- Manual correction sistemleri

## Auth Requirement

**Gerekli** - JWT token + Admin rolü

## Side Effects (Yan Etki)

- **Veri yazar mı?** Evet - Neo4j ve Redis güncellenir
- Kullanıcı puanı değişir
- Leaderboard güncellenir
- Activity history kaydedilir

## Operations

### 1. Add (Ekle)
Mevcut puana yeni puan ekler:

```bash
{
  "points": 100,
  "operation": "add"
}
```

### 2. Subtract (Çıkar)
Mevcut puandan puan çıkarır:

```bash
{
  "points": 50,
  "operation": "subtract"
}
```

### 3. Set (Ayarla)
Puanı doğrudan belirtilen değere ayarlar:

```bash
{
  "points": 500,
  "operation": "set"
}
```

## Example Request - Add

```bash
curl -X PUT http://localhost:3000/api/v1/users/user_001/points \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "points": 100,
    "operation": "add"
  }'
```

## Example Response

```json
{
  "user_id": "user_001",
  "points": 150,
  "message": "Points updated successfully"
}
```

## Example Request - Subtract

```bash
curl -X PUT http://localhost:3000/api/v1/users/user_001/points \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "points": 50,
    "operation": "subtract"
  }'
```

## Example Request - Set

```bash
curl -X PUT http://localhost:3000/api/v1/users/user_001/points \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "points": 1000,
    "operation": "set"
  }'
```

## Common Errors (Sık Hata Nedenleri)

| Hata Kodu | Mesaj | Çözüm |
|-----------|-------|-------|
| 400 | Points value is required | points değeri girin |
| 404 | User not found | Geçerli user ID girin |
| 400 | Insufficient points (for subtract) | Yeterli puan yok |

## Best Practices

1. operation parametresini her zaman belirtin
2. Negative puan önlemek için validation yapın
3. Değişiklik loglarını tutun
4. İşlem sonrası kullanıcı bilgisini doğrulayın