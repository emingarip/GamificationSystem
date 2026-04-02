---
sidebar_position: 100
---

# Error Handling Guide

This guide explains how to handle errors from the Gamification API.

## Error Response Format

All error responses follow a consistent JSON format:

```json
{
  "error": "Error message description"
}
```

## HTTP Status Codes

| Status Code | Meaning | Description |
|-------------|---------|-------------|
| 200 | OK | Request successful |
| 201 | Created | Resource created successfully |
| 400 | Bad Request | Invalid request body or parameters |
| 401 | Unauthorized | Invalid or missing JWT token |
| 403 | Forbidden | Admin access required |
| 404 | Not Found | Resource not found |
| 409 | Conflict | Resource already exists |
| 500 | Internal Server Error | Server-side error |

## Common Errors by Category

### Authentication Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `Invalid or expired token` | Token süresi dolmuş veya geçersiz | Login olup yeni token alın |
| `Invalid credentials` | Email veya şifre hatalı | Doğru credentials girin |
| `Admin access required` | Endpoint için admin yetkisi gerekli | Admin token kullanın |

### Validation Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `Invalid request body` | JSON parse hatası | JSON formatını kontrol edin |
| `Rule name is required` | Rule name boş | name alanını girin |
| `Event type is required` | Event type boş | event_type alanını girin |
| `Points value is required` | Points değeri sıfır | Pozitif değer girin |
| `Badge ID is required` | Badge ID boş | badge_id girin |

### Resource Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `User not found` | ID mevcut değil | Geçerli user ID kullanın |
| `Rule not found` | Rule mevcut değil | Geçerli rule ID kullanın |
| `Badge not found` | Badge mevcut değil | Geçerli badge ID kullanın |

### Data Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `Badge with this name or ID already exists` | Duplicate badge | Farklı ID/name kullanın |
| `Insufficient points` | Yetersiz puan | Daha düşük çıkarma değeri kullanın |

### Server Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `Failed to create rule` | Redis hatası | Tekrar deneyin |
| `Failed to update user points` | Neo4j hatası | Tekrar deneyin |
| `Failed to fetch users` | Database hatası | Tekrar deneyin |

## Error Handling Best Practices

### 1. Always Handle 401

```javascript
if (response.status === 401) {
  // Redirect to login or refresh token
  window.location.href = '/login';
}
```

### 2. Validate Input Client-Side

```javascript
function validatePointsInput(points) {
  if (!points || points <= 0) {
    throw new Error('Points must be greater than 0');
  }
}
```

### 3. Use Retry Logic for 500 Errors

```javascript
async function withRetry(fn, retries = 3) {
  for (let i = 0; i < retries; i++) {
    try {
      return await fn();
    } catch (e) {
      if (i === retries - 1) throw e;
      await delay(1000 * (i + 1));
    }
  }
}
```

### 4. Log Errors for Debugging

```javascript
console.error('API Error:', error.response?.data);
```

## Health Check for Debugging

Sistem durumunu kontrol edin:

```bash
GET /health
```

Response:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "services": {
    "redis": { "status": "healthy" },
    "neo4j": { "status": "healthy" },
    "kafka": { "status": "healthy" }
  }
}
```

If any service is `unhealthy`, check that service's status.

## Support

If you encounter persistent errors:
1. Check `/health` endpoint
2. Review server logs
3. Contact development team with error details