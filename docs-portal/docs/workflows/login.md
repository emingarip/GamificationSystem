---
sidebar_position: 1
---

# Admin Login Workflow

## Purpose (Amaç)

Admin kullanıcılarının sisteme giriş yapmasını ve JWT token almasını sağlar.

## When to Use (Ne Zaman Kullanılır)

- Admin panel kullanıcı girişi
- API entegrasyonu için token alma
- Uygulama başlangıcında kimlik doğrulama

## Who Uses (Kim Kullanır)

- Frontend admin uygulaması
- Harici sistem entegrasyonları
- CI/CD pipeline'ları

## Auth Requirement

**Yok** - Bu endpoint herkes tarafından erişilebilir (public).

## Side Effects (Yan Etki)

- **Veri yazar mı?** Hayır - Sadece okuma
- **Token üretir** - JWT access ve refresh token döner

## Example Request

```bash
curl -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@gamification.local",
    "password": "admin123"
  }'
```

## Example Response

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhZG1pbl8wMDEiLCJlbWFpbCI6ImFkbWluQGdhbWlm... ",
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhZG1pbl8wMDEiLCJlbWFpbCI6ImFkbWluQGdhbWlm... ",
  "user": {
    "id": "admin_001",
    "name": "Admin User",
    "email": "admin@gamification.local",
    "level": 1,
    "points": 0,
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-15T10:30:00Z",
    "badges": []
  }
}
```

## Common Errors (Sık Hata Nedenleri)

| Hata Kodu | Mesaj | Çözüm |
|-----------|-------|-------|
| 400 | Invalid request body | JSON formatını kontrol edin |
| 401 | Invalid credentials | Email veya şifre hatalı |
| 500 | Internal server error | Sunucu logs kontrol edilmeli |

## Implementation Tips

1. Token'ı güvenli bir şekilde saklayın
2. Token süresi dolduğında refresh token kullanın
3. Logout yapınca token'ı silin