# AI-Native Gamification Platform

An AI-Native Gamification Platform with Knowledge Graph built with Go, Neo4j, Redis, and Kafka.

## Overview

This platform provides:
- Real-time gamification rule engine
- Knowledge graph-based user analytics
- Badge and points management
- Event-driven architecture
- LLM-powered rule transformation

## Project Structure

```
├── admin/              # React admin dashboard
├── internal/muscle/   # Go API server (muscle layer)
│   ├── cmd/mcp-server/ # MCP server entry point
│   ├── mcp/           # MCP server implementation
│   │   └── backend/   # Service layer for MCP
│   └── ...
├── mobile/            # Flutter mobile app
├── plans/             # Technical specifications
├── scripts/           # Automation scripts
├── docker-compose.yml # Infrastructure setup
└── README.md          # This file
```

## Quick Start

### Prerequisites

- Go 1.25+
- Docker and Docker Compose
- Node.js 18+ (for admin UI)

### Running the API

```bash
cd internal/muscle
go run main.go
```

The API runs on port 3000 by default.

## API Documentation

### Tek Kaynak Prensibi

Bu proje "Single Source of Truth" prensibiyle dokümantasyon yönetir. Tüm API dokümantasyonu tek bir kaynaktan üretilir ve senkronize edilir:

| Konum | Açıklama | Kaynak |
|-------|-----------|--------|
| `internal/muscle/docs/swagger.json` | Backend Swagger çıktısı | **TEK KAYNAK** (swag init ile üretilir) |
| `internal/muscle/mcp/resources/swagger.json` | MCP embed kaynağı | sync-docs scripti ile |
| `/swagger/index.html` | Canlı Swagger UI | Backend tarafından servis edilir |
| `/docs/api-reference/` | API Reference (redirect) | `/swagger/index.html`'a yönlendirir |

**Not:** Docs portal OpenAPI kopyası (`docs-portal/docs/openapi.yaml`) kaldırılmıştır. Docs portal API Reference sayfası (`/api-reference`) canlı Swagger UI'ya (`/swagger/index.html`) yönlendirme yapar.

### Developer Portal (Docusaurus)

The project includes a comprehensive developer documentation portal built with Docusaurus. This provides a more user-friendly interface than Swagger for understanding the API and workflows.

#### Running the Docs Portal

```bash
# Preview docs locally
cd docs-portal && npm run start

# Build docs for production
cd docs-portal && npm run build
```

**Local Development:** When running the API server locally (`go run main.go`), access the docs at http://localhost:3000/docs/

**Production (Docker):** The Docker image includes the pre-built docs. Run `docker compose up -d muscle` and access at http://localhost:3000/docs/

#### Expected URLs

| Path | Description |
|------|-------------|
| `/docs` | Redirects to `/docs/` |
| `/docs/` | API Overview (Docusaurus landing page) |
| `/docs/overview/` | Redirects to `/docs/` (backward compatibility) |
| `/docs/quick-start/` | Quick Start Guide |
| `/docs/authentication/` | Authentication Guide |
| `/docs/workflows/` | Workflow Guides |
| `/docs/api-reference/` | Docs portal page that redirects users to Swagger UI |
| `/swagger/index.html` | Swagger API Reference |
| `/docs/swagger/index.html` | Redirects to `/swagger/index.html` |

#### Docs Structure

```
docs-portal/
├── docs/
│   ├── overview.md          # API overview
│   ├── quick-start.md        # Getting started
│   ├── authentication.md     # JWT auth guide
│   ├── workflows/            # Step-by-step guides
│   │   ├── login.md
│   │   ├── create-rule.md
│   │   ├── test-event-dryrun.md
│   │   ├── test-event-execute.md
│   │   ├── update-points.md
│   │   ├── assign-badge.md
│   │   └── read-analytics.md
│   ├── rules/                # Rules API docs
│   ├── users/               # Users API docs
│   ├── badges/              # Badges API docs
│   ├── events/              # Events API docs
│   ├── analytics/           # Analytics API docs
│   └── error-handling.md    # Error handling guide
└── docusaurus.config.ts     # Docusaurus config
```

**Not:** Docs portal API Reference sayfası canlı Swagger UI'ya yönlendirme yapar; ayrı OpenAPI spec artefact'ı tutulmaz.

### Swagger Documentation

The API documentation is also available via Swagger/OpenAPI. To regenerate the docs:

```bash
cd internal/muscle
swag init -g main.go -o docs
```

**⚠️ ÖNEMLI:** Artık tek komutla tüm dokümantasyonu senkronize edebilirsiniz:

```bash
# Windows
powershell -ExecutionPolicy Bypass -File .\scripts\sync-docs.ps1

# Linux/Mac
bash scripts/sync-docs.sh
```

Bu script:
1. `swag init` ile backend swagger'ı üretir
2. MCP embed kaynağını (`mcp/resources/swagger.json`) günceller
3. Git status ile değişiklikleri raporlar

**Not:** Docs portal API Reference sayfası canlı Swagger UI'ya yönlendirme yapar; ayrı OpenAPI spec artefact'ı tutulmaz.

### Accessing Documentation

Once the server is running:

- **Swagger UI**: http://localhost:3000/swagger/index.html
- **OpenAPI JSON**: http://localhost:3000/swagger/doc.json

### API Endpoints

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | /health | Health check | No |
| GET | /metrics | Prometheus metrics | No |
| POST | /api/v1/auth/login | Admin login | No |
| GET | /api/v1/auth/me | Get current user | Bearer |
| POST | /api/v1/auth/logout | Logout | Bearer |
| GET | /api/v1/leaderboard | Get leaderboard | No |
| GET | /api/v1/rules | List rules | No |
| POST | /api/v1/rules | Create rule | Bearer (Admin) |
| GET | /api/v1/rules/:id | Get rule | No |
| PUT | /api/v1/rules/:id | Update rule | Bearer (Admin) |
| DELETE | /api/v1/rules/:id | Delete rule | Bearer (Admin) |
| GET | /api/v1/users | List users | Bearer |
| GET | /api/v1/users/:id | Get user profile | Bearer |
| PUT | /api/v1/users/:id | Update user | Bearer (Admin) |
| DELETE | /api/v1/users/:id | Delete user | Bearer (Admin) |
| PUT | /api/v1/users/:id/points | Update user points | Bearer (Admin) |
| GET | /api/v1/users/:id/stats | Get user stats | Bearer |
| POST | /api/v1/users/:id/badges | Assign badge | Bearer (Admin) |
| GET | /api/v1/badges | List badges | Bearer |
| POST | /api/v1/badges | Create badge | Bearer (Admin) |
| GET | /api/v1/badges/:id | Get badge | Bearer |
| PUT | /api/v1/badges/:id | Update badge | Bearer (Admin) |
| DELETE | /api/v1/badges/:id | Delete badge | Bearer (Admin) |
| GET | /api/v1/analytics/summary | Analytics summary | Bearer |
| GET | /api/v1/analytics/activity | Recent activity | Bearer |
| GET | /api/v1/analytics/points-history | Points history | Bearer |
| GET | /api/v1/matches/:id/stats | Match stats | Bearer |
| POST | /api/v1/events/test | Test event | Bearer (Admin) |

### Authentication

The API uses JWT Bearer token authentication. Include the token in the Authorization header:

```
Authorization: Bearer <your-token>
```

To obtain a token, POST to `/api/v1/auth/login` with admin credentials.

## Development

### Running Tests

```bash
cd internal/muscle
go test ./...
```

### Environment Variables

Copy `.env.example` to `.env` and configure:

- `REDIS_HOST`, `REDIS_PORT` - Redis connection
- `NEO4J_URI`, `NEO4J_USERNAME`, `NEO4J_PASSWORD` - Neo4j connection
- `KAFKA_BROKER` - Kafka broker address
- `JWT_SECRET_KEY` - JWT signing key

## MCP Server (Model Context Protocol)

An MCP server provides AI agents with tools to interact with the gamification system. It offers a standardized protocol for reading rules, testing events, managing users, and analyzing analytics.

### Tek Komut Kurulum

Windows tarafında en pratik yol:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\setup-mcp.ps1
```

Bu script:
- `.env` yoksa oluşturur
- Redis ve Neo4j'i ayağa kaldırır
- `mcp-server.exe` binary'sini üretir
- agent'lara verebileceğin hazır MCP config snippet dosyalarını oluşturur

Üretilen dosyalar:

- `scripts\run-mcp-server.cmd`
- `scripts\mcp-configs\generic-mcp.json`
- `scripts\mcp-configs\cursor.mcp.json`
- `scripts\mcp-configs\claude-desktop.mcp.json`

### Running the MCP Server

Docker'da remote MCP olarak calistirmak icin:

```bash
docker compose -f docker-compose-mcp.yml up -d --build mcp-server
```

Kilo Code icin remote MCP config:

```json
{
  "mcpServers": {
    "gamification": {
      "type": "streamable-http",
      "url": "http://localhost:3002/mcp",
      "headers": {},
      "disabled": false,
      "alwaysAllow": []
    }
  }
}
```

Docker remote transport aktifken MCP endpoint:

```text
http://localhost:3002/mcp
```

Lokal stdio alternatifine ihtiyacin varsa:

```bash
# From project root
cd internal/muscle
go run ./cmd/mcp-server/main.go

# Or use the built binary
cd internal/muscle
./mcp-server
```

The MCP server runs with stdio transport by default. It can be connected to AI agents like Claude Desktop or other MCP-compatible clients.

En kolayı, agent config içinde `command` olarak şu launcher'ı kullanmak:

```text
scripts\run-mcp-server.cmd
```

Bu launcher:
- repo `.env` dosyasını yükler
- local Docker portlarına uygun env değerlerini set eder
- `internal\muscle\mcp-server.exe` binary'sini başlatır

Yani agent config'inde ayrıca `cwd`, `env`, `go run` veya build komutu yazman gerekmez.

### MCP Tools

| Tool | Description | Backend |
|------|-------------|---------|
| `list_rules` | Lists active rules from Redis, filtered by event type. Returns only enabled/active rules. | Redis |
| `get_rule` | Get detailed information about a specific rule | Redis |
| `test_event` | Test how an event would be processed (dry-run or execute). **Dinamik event type desteği**: `event_type` Redis'ten dinamik olarak okunur. Sport events (`goal`, `corner`, vb.) için `match_id` ve `player_id` zorunludur. Custom/Engagement events için opsiyoneldir. | Rule Engine |
| `assign_badge_to_user` | Manually assign a badge to a user | Reward Layer |
| `update_user_points` | Add, subtract, or set user points | Neo4j |
| `list_users` | List all users with pagination (max 100) | Neo4j |
| `get_user_profile` | Get detailed user profile with points, badges, and activity | Neo4j |
| `get_analytics_summary` | Get analytics summary | Neo4j + Redis |

### MCP Resources

| Resource URI | Description | MIME Type |
|--------------|-------------|----------|
| `rules://list` | List of all gamification rules | application/json |
| `rules://{id}` | Single rule by ID | application/json |
| `analytics://summary` | Analytics summary | application/json |
| `users://{id}` | User profile by ID | application/json |
| `docs://real-time-badge-flow` | Real-time badge flow documentation | text/markdown |
| `openapi://current` | OpenAPI specification | application/json |

### MCP Prompts

| Prompt | Description |
|--------|-------------|
| `debug-badge-flow` | Analyze why a badge was or was not awarded |
| `draft-rule-from-text` | Generate a rule draft from natural language |
| `analyze-user-state` | Analyze and interpret user engagement |

### Known Limitations

- The MCP server requires Redis and Neo4j to be running for full functionality
- If backend services are unavailable, the server runs in limited mode:
  - Without Redis: analytics, rules, and test_event unavailable
  - Without Neo4j: users, badges, and points operations unavailable
- test_event requires the rule engine to be initialized (both Redis and Neo4j must be connected)

## Release Checklist

### Prerequisites
- [ ] Docker & Docker Compose installed
- [ ] Go 1.25+ (for local development)
- [ ] Node.js 18+ (for admin UI development)

### Quick Start (Docker)
```bash
# 1. Start infrastructure
docker compose up -d redis neo4j kafka

# 2. Start API server
docker compose up -d muscle

# 3. Start admin dashboard
docker compose up -d admin
```

### Local Development
```bash
# Backend
cd internal/muscle
go run main.go

# Admin UI
cd admin
npm run dev
```

### Admin Panel - First Use
1. Open http://localhost:5173
2. Login with: `admin@admin.com` / `admin123`
3. Navigate to "Event Types" - verify sport event types (goal, corner, etc.) are seeded
4. Navigate to "Rules" - create a test rule
5. Use "Test Event" to verify rule matching works

### Testing the Platform
1. **Rule Engine**: Create a rule with `event_type=goal`, test with `test_event` endpoint
2. **Event Types**: Toggle `enabled` status - verify disabled types don't trigger rules
3. **MCP Server**: Test with `test_event` dry-run to verify rule matching

### Production Considerations
- [ ] Change default admin password (`admin123`)
- [ ] Configure strong `JWT_SECRET_KEY`
- [ ] Set appropriate Redis memory limits (current: 256mb)
- [ ] Configure Neo4j heap sizes for expected load
- [ ] Set up monitoring for Redis/Neo4j/Kafka health
- [ ] Configure LLM endpoint for rule transformation (optional)

### Health Checks
| Service | Port | Endpoint |
|---------|------|----------|
| API Server | 3000 | `/health` |
| Neo4j | 7687 | Bolt protocol |
| Redis | 6379 | `redis-cli ping` |
| Kafka | 9092 | Broker API |

### Project Structure

```
internal/muscle/
├── cmd/mcp-server/
│   └── main.go          # MCP server entry point
├── mcp/
│   └── backend/
│       ├── service.go   # Backend service implementation
│       └── service_test.go  # Unit tests
├── redis/               # Redis client
├── neo4j/               # Neo4j client  
└── engine/              # Rule engine and reward layer
```

### Security Note

The MCP server operates as an internal trusted tool surface. Write operations (`update_user_points`, `assign_badge_to_user`) bypass user-level authorization and should only be used by trusted AI agents with appropriate system-level access.

## Docker

See [DOCKER_README.md](DOCKER_README.md) for Docker Compose infrastructure setup.
