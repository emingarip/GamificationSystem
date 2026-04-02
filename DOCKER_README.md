# Docker Compose Infrastructure for AI-Native Gamification Platform

This repository contains a Docker Compose stack for the internal beta environment: Neo4j, Redis, Kafka, the Go API, and the React admin UI.

## Services

| Service | Port | Description |
|---------|------|-------------|
| Neo4j | 7475 (HTTP), 7688 (Bolt) | Graph database for knowledge graph |
| Redis | 6379 | In-memory store for rules and caching |
| Kafka | 9092 | Message broker for event streaming |
| Zookeeper | 2181 | Kafka's Zookeeper (internal) |
| Admin UI | 5173 | React admin dashboard |
| Kafka UI | 8080 | Kafka management UI (optional) |
| Redis Commander | 8081 | Redis management UI (optional) |

## Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+

## Quick Start

### 1. Copy Environment Variables

```bash
cp .env.example .env
```

### 2. Start All Services

```bash
# Start all services (including optional UIs)
docker compose up -d

# Or start only core services
docker compose up -d neo4j redis zookeeper kafka muscle admin
```

### 3. Verify Services

```bash
# Check service status
docker compose ps

# View logs
docker compose logs -f neo4j
docker compose logs -f redis
docker compose logs -f kafka
```

### 4. Access Services

| Service | URL | Credentials |
|---------|-----|-------------|
| Admin UI | http://localhost:5173 | `ADMIN_USERNAME` / seeded password `admin123` unless overridden |
| API Health | http://localhost:3000/health | - |
| Neo4j Browser | http://localhost:7475 | neo4j/neo4j_password |
| Kafka UI | http://localhost:8080 | - |
| Redis Commander | http://localhost:8081 | - |

### 5. Run The Internal Beta Smoke Test

PowerShell smoke validation lives at [scripts/internal-beta-smoke.ps1](scripts/internal-beta-smoke.ps1).

```powershell
./scripts/internal-beta-smoke.ps1 -StartStack
```

What it verifies:
- API and admin UI availability
- admin login
- user listing and profile reads
- badge create/delete
- rule create/delete
- `/api/v1/events/test` dry-run and execute paths
- points/badge side effects plus cleanup
- analytics summary and recent activity reads

There is also a manual checklist at [plans/internal-beta-smoke-checklist.md](plans/internal-beta-smoke-checklist.md).

## Configuration

Default ports and settings are defined in [docker-compose.yml](docker-compose.yml). All required environment variables, including admin credentials, are documented in [.env.example](.env.example).

### Neo4j

- HTTP: http://localhost:7475
- Bolt: bolt://localhost:7688
- Initial password: `neo4j_password`

### Redis

- Host: localhost
- Port: 6379
- No password by default

### Kafka

- Bootstrap Server: localhost:9092
- Zookeeper: localhost:2181
- Auto-creates topics: `match-events`, `rule-triggered`, `user-actions`

## Management Commands

```bash
# Stop all services
docker compose stop

# Stop and remove volumes (data loss)
docker compose down -v

# Restart a specific service
docker compose restart kafka

# View service health
docker inspect gamification-neo4j | grep -A 10 Health
```

## Data Persistence

Data is stored in Docker volumes:
- `neo4j_data` - Neo4j database files
- `redis_data` - Redis persistence
- `kafka_data` - Kafka broker data

To backup:

```bash
docker run --rm -v gamification-neo4j_data:/data -v $(pwd)/backup:/backup alpine tar czf /backup/neo4j-backup.tar.gz /data
```

## Troubleshooting

### Neo4j won't start

```bash
docker compose logs neo4j
docker system df
```

### Kafka not ready

```bash
docker compose logs zookeeper
```

### Redis connection refused

```bash
docker inspect gamification-redis | grep -A 5 Health
```

## Production Considerations

1. Change default passwords in `.env`.
2. Enable Redis authentication.
3. Configure Neo4j SSL/TLS.
4. Set up Kafka security.
5. Add health checks for all services.
6. Use external volumes for data persistence.
7. Configure resource limits based on hardware.
