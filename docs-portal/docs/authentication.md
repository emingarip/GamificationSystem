---
sidebar_position: 3
---

# Authentication

The Gamification API uses JWT (JSON Web Tokens) for authentication. All protected endpoints require a valid JWT token in the `Authorization` header.

## Authentication Flow

### 1. Login

To authenticate, send a POST request to `/auth/login` with admin credentials:

```bash
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "admin@gamification.local",
  "password": "admin123"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "admin_001",
    "name": "Admin User",
    "email": "admin@gamification.local",
    "level": 1,
    "points": 0,
    "badges": []
  }
}
```

### 2. Use the Token

Include the JWT token in subsequent API requests:

```bash
GET /api/v1/users
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### 3. Get Current User

Verify your token and get current user info:

```bash
GET /api/v1/auth/me
Authorization: Bearer YOUR_TOKEN_HERE
```

### 4. Logout

Logout (client should discard the token):

```bash
POST /api/v1/auth/logout
Authorization: Bearer YOUR_TOKEN_HERE
```

## Token Details

### Access Token
- **Purpose**: Used for API authentication
- **Expiration**: Configurable (default: 15 minutes)
- **Header Format**: `Authorization: Bearer <token>`

### Refresh Token
- **Purpose**: Used to obtain new access tokens
- **Expiration**: Configurable (default: 7 days)
- **Storage**: Should be stored securely on client side

## Security

### Protected Endpoints

The following endpoints require authentication:
- `POST /api/v1/rules` - Create rule
- `PUT /api/v1/rules/{id}` - Update rule
- `DELETE /api/v1/rules/{id}` - Delete rule
- `GET /api/v1/users` - List users
- `GET /api/v1/users/{id}` - Get user profile
- `PUT /api/v1/users/{id}` - Update user
- `DELETE /api/v1/users/{id}` - Delete user
- `PUT /api/v1/users/{id}/points` - Update points
- `POST /api/v1/users/{id}/badges` - Assign badge
- `GET /api/v1/badges` - List badges
- `POST /api/v1/badges` - Create badge
- `PUT /api/v1/badges/{id}` - Update badge
- `DELETE /api/v1/badges/{id}` - Delete badge
- `POST /api/v1/events/test` - Test event
- `GET /api/v1/analytics/*` - Analytics endpoints

### Public Endpoints
- `GET /api/v1/rules` - List rules (read-only)
- `GET /api/v1/leaderboard` - View leaderboard
- `POST /api/v1/auth/login` - Login
- `GET /health` - Health check

## Error Responses

### 401 Unauthorized
```json
{
  "error": "Invalid or expired token"
}
```

### 403 Forbidden
```json
{
  "error": "Admin access required"
}
```

## Best Practices

1. **Store tokens securely** - Use secure storage, never localStorage for sensitive apps
2. **Handle token expiration** - Implement refresh token logic
3. **Use HTTPS** - Always use secure connections in production
4. **Token rotation** - Implement proper logout to invalidate tokens