# Knowledge Graph Schema for Sports Gamification Platform

## Overview

This document defines the complete Neo4j Knowledge Graph schema for an AI-Native Gamification & Sports Analytics platform. The schema implements a Brain & Muscle separation architecture where strategic decisions (LLM + Neo4j) run asynchronously while instant execution (Go + Redis + Kafka) happens in milliseconds.

---

## 1. Node Definitions (Entities)

### 1.1 User Node

```cypher
// User: Platform users (fans, players)
CREATE CONSTRAINT user_id_unique IF NOT EXISTS
FOR (u:User) REQUIRE u.id IS UNIQUE;

CREATE CONSTRAINT user_email_unique IF NOT EXISTS
FOR (u:User) REQUIRE u.email IS UNIQUE;

CREATE INDEX user_username_idx IF NOT EXISTS
FOR (u:User) ON (u.username);

CREATE INDEX user_created_at_idx IF NOT EXISTS
FOR (u:User) ON (u.createdAt);
```

**Properties:**
| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `id` | UUID | Yes | Unique identifier |
| `email` | String | Yes | User email (unique) |
| `username` | String | Yes | Display name |
| `passwordHash` | String | Yes | Bcrypt hashed password |
| `role` | String | Yes | Enum: fan, player, admin |
| `avatarUrl` | String | No | Profile picture URL |
| `points` | Integer | Yes | Gamification points (default: 0) |
| `level` | Integer | Yes | User level (default: 1) |
| `createdAt` | DateTime | Yes | Account creation timestamp |
| `updatedAt` | DateTime | Yes | Last update timestamp |
| `lastActiveAt` | DateTime | No | Last activity timestamp |
| `isActive` | Boolean | Yes | Account status (default: true) |
| `preferences` | Map | No | User preferences JSON |

---

### 1.2 Team Node

```cypher
// Team: Sports teams
CREATE CONSTRAINT team_id_unique IF NOT EXISTS
FOR (t:Team) REQUIRE t.id IS UNIQUE;

CREATE CONSTRAINT team_name_unique IF NOT EXISTS
FOR (t:Team) REQUIRE t.name IS UNIQUE;

CREATE INDEX team_sport_idx IF NOT EXISTS
FOR (t:Team) ON (t.sport);
```

**Properties:**
| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `id` | UUID | Yes | Unique identifier |
| `name` | String | Yes | Team name (unique) |
| `shortName` | String | No | Short team name (e.g., "Lakers") |
| `sport` | String | Yes | Sport type (basketball, soccer, etc.) |
| `league` | String | No | Competition league |
| `logoUrl` | String | No | Team logo URL |
| `primaryColor` | String | No | Team brand color |
| `secondaryColor` | String | No | Team brand color |
| `foundedYear` | Integer | No | Year team was established |
| `city` | String | No | Team home city |
| `country` | String | No | Team country |
| `stadium` | String | No | Home venue name |
| `stats` | Map | No | Team statistics JSON |
| `createdAt` | DateTime | Yes | Creation timestamp |

---

### 1.3 Match Node

```cypher
// Match: Games/matches
CREATE CONSTRAINT match_id_unique IF NOT EXISTS
FOR (m:Match) REQUIRE m.id IS UNIQUE;

CREATE INDEX match_scheduled_at_idx IF NOT EXISTS
FOR (m:Match) ON (m.scheduledAt);

CREATE INDEX match_status_idx IF NOT EXISTS
FOR (m:Match) ON (m.status);
```

**Properties:**
| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `id` | UUID | Yes | Unique identifier |
| `homeTeamId` | UUID | Yes | Reference to home team |
| `awayTeamId` | UUID | Yes | Reference to away team |
| `sport` | String | Yes | Sport type |
| `league` | String | No | Competition league |
| `scheduledAt` | DateTime | Yes | Match start time |
| `endedAt` | DateTime | No | Match end time |
| `status` | String | Yes | Enum: scheduled, live, completed, cancelled |
| `homeScore` | Integer | No | Home team score |
| `awayScore` | Integer | No | Away team score |
| `venue` | String | No | Match venue |
| `round` | String | No | Competition round/phase |
| `stats` | Map | No | Match statistics JSON |
| `highlights` | List | No | Match highlight URLs |
| `viewerCount` | Integer | No | Live viewer count |
| `createdAt` | DateTime | Yes | Creation timestamp |

---

### 1.4 Player Node

```cypher
// Player: Individual athletes
CREATE CONSTRAINT player_id_unique IF NOT EXISTS
FOR (p:Player) REQUIRE p.id IS UNIQUE;

CREATE INDEX player_name_idx IF NOT EXISTS
FOR (p:Player) ON (p.name);

CREATE INDEX player_position_idx IF NOT EXISTS
FOR (p:Player) ON (p.position);
```

**Properties:**
| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `id` | UUID | Yes | Unique identifier |
| `name` | String | Yes | Player full name |
| `firstName` | String | No | First name |
| `lastName` | String | No | Last name |
| `position` | String | No | Player position |
| `jerseyNumber` | Integer | No | Jersey number |
| `sport` | String | Yes | Sport type |
| `nationality` | String | No | Country of origin |
| `dateOfBirth` | Date | No | Birth date |
| `height` | Integer | No | Height in cm |
| `weight` | Integer | No | Weight in kg |
| `photoUrl` | String | No | Player photo URL |
| `stats` | Map | No | Player statistics JSON |
| `createdAt` | DateTime | Yes | Creation timestamp |

---

### 1.5 Badge Node

```cypher
// Badge: Achievement badges
CREATE CONSTRAINT badge_id_unique IF NOT EXISTS
FOR (b:Badge) REQUIRE b.id IS UNIQUE;

CREATE INDEX badge_category_idx IF NOT EXISTS
FOR (b:Badge) ON (b.category);
```

**Properties:**
| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `id` | UUID | Yes | Unique identifier |
| `name` | String | Yes | Badge name |
| `description` | String | Yes | Badge description |
| `iconUrl` | String | No | Badge icon URL |
| `category` | String | Yes | Badge category (fan, player, social) |
| `rarity` | String | Yes | Enum: common, rare, epic, legendary |
| `pointsValue` | Integer | Yes | Points awarded when earned |
| `criteria` | Map | No | Earn criteria JSON |
| `createdAt` | DateTime | Yes | Creation timestamp |

---

### 1.6 Achievement Node

```cypher
// Achievement: Achievement definitions
CREATE CONSTRAINT achievement_id_unique IF NOT EXISTS
FOR (a:Achievement) REQUIRE a.id IS UNIQUE;

CREATE INDEX achievement_type_idx IF NOT EXISTS
FOR (a:Achievement) ON (a.type);
```

**Properties:**
| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `id` | UUID | Yes | Unique identifier |
| `name` | String | Yes | Achievement name |
| `description` | String | Yes | Achievement description |
| `type` | String | Yes | Enum: watch, participate, social, milestone |
| `targetValue` | Integer | Yes | Target count to unlock |
| `currentValue` | Integer | Yes | Current progress value |
| `pointsReward` | Integer | Yes | Points awarded on completion |
| `isActive` | Boolean | Yes | Active status (default: true) |
| `createdAt` | DateTime | Yes | Creation timestamp |

---

### 1.7 Action Node

```cypher
// Action: User actions/events
CREATE CONSTRAINT action_id_unique IF NOT EXISTS
FOR (a:Action) REQUIRE a.id IS UNIQUE;

CREATE INDEX action_type_idx IF NOT EXISTS
FOR (a:Action) ON (a.type);

CREATE INDEX action_timestamp_idx IF NOT EXISTS
FOR (a:Action) ON (a.timestamp);
```

**Properties:**
| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `id` | UUID | Yes | Unique identifier |
| `type` | String | Yes | Enum: watch, cheer, comment, share, predict, bet |
| `userId` | UUID | Yes | User who performed action |
| `matchId` | UUID | No | Match where action occurred |
| `teamId` | UUID | No | Team related to action |
| `pointsEarned` | Integer | Yes | Points earned from action |
| `metadata` | Map | No | Additional action data JSON |
| `timestamp` | DateTime | Yes | When action occurred |
| `ipAddress` | String | No | User IP for audit |
| `userAgent` | String | No | Browser/device info |

---

## 2. Relationship Definitions (Edges)

### 2.1 User-Team Relationships

```cypher
// SUPPORTS: User supports/follows a team
// PLAYER: User is a player on a team
```

```cypher
// Create SUPPORTS relationship
MATCH (u:User {id: $userId}), (t:Team {id: $teamId})
CREATE (u)-[r:SUPPORTS {
    since: datetime($since),
    isPrimary: $isPrimary,
    notificationsEnabled: true,
    createdAt: datetime()
}]->(t);

// Create PLAYER relationship
MATCH (u:User {id: $userId}), (t:Team {id: $teamId})
CREATE (u)-[r:PLAYS_FOR {
    jerseyNumber: $jerseyNumber,
    position: $position,
    startDate: date($startDate),
    endDate: date($endDate),
    isActive: true,
    createdAt: datetime()
}]->(t);
```

**SUPPORTS Properties:**
| Property | Type | Description |
|----------|------|-------------|
| `since` | DateTime | When user started supporting |
| `isPrimary` | Boolean | Primary team flag |
| `notificationsEnabled` | Boolean | Team notification preference |
| `createdAt` | DateTime | Relationship creation time |

**PLAYS_FOR Properties:**
| Property | Type | Description |
|----------|------|-------------|
| `jerseyNumber` | Integer | Player jersey number |
| `position` | String | Player position |
| `startDate` | Date | Contract start date |
| `endDate` | Date | Contract end date |
| `isActive` | Boolean | Active player status |
| `createdAt` | DateTime | Relationship creation time |

---

### 2.2 Player-Team Relationship

```cypher
// PLAYS_FOR: Player is on a team
MATCH (p:Player {id: $playerId}), (t:Team {id: $teamId})
CREATE (p)-[r:PLAYS_FOR {
    jerseyNumber: $jerseyNumber,
    position: $position,
    startDate: date($startDate),
    endDate: date($endDate),
    isActive: true,
    createdAt: datetime()
}]->(t);
```

**Properties:**
| Property | Type | Description |
|----------|------|-------------|
| `jerseyNumber` | Integer | Jersey number |
| `position` | String | Player position |
| `startDate` | Date | Start date |
| `endDate` | Date | End date |
| `isActive` | Boolean | Active status |
| `createdAt` | DateTime | Creation timestamp |

---

### 2.3 Match-Team Relationships

```cypher
// HOME_TEAM: Team is playing at home
// AWAY_TEAM: Team is playing away
MATCH (m:Match {id: $matchId}), (t:Team {id: $teamId})
CREATE (m)-[r:HOME_TEAM {
    score: $homeScore,
    isWinner: $isWinner,
    createdAt: datetime()
}]->(t);

MATCH (m:Match {id: $matchId}), (t:Team {id: $teamId})
CREATE (m)-[r:AWAY_TEAM {
    score: $awayScore,
    isWinner: $isWinner,
    createdAt: datetime()
}]->(t);
```

**Properties:**
| Property | Type | Description |
|----------|------|-------------|
| `score` | Integer | Team score in match |
| `isWinner` | Boolean | Win status |
| `createdAt` | DateTime | Creation timestamp |

---

### 2.4 User-Match Relationships

```cypher
// WATCHED: User watched a match
// PARTICIPATED: User participated in match (predictions, etc.)
MATCH (u:User {id: $userId}), (m:Match {id: $matchId})
CREATE (u)-[r:WATCHED {
    watchDuration: $duration,
    watchPercentage: $percentage,
    timestamp: datetime(),
    device: $device,
    createdAt: datetime()
}]->(m);

MATCH (u:User {id: $userId}), (m:Match {id: $matchId})
CREATE (u)-[r:PARTICIPATED {
    prediction: $prediction,
    isCorrect: $isCorrect,
    pointsEarned: $points,
    createdAt: datetime()
}]->(m);
```

**WATCHED Properties:**
| Property | Type | Description |
|----------|------|-------------|
| `watchDuration` | Integer | Seconds watched |
| `watchPercentage` | Float | % of match watched |
| `timestamp` | DateTime | Watch timestamp |
| `device` | String | Device type |
| `createdAt` | DateTime | Creation timestamp |

**PARTICIPATED Properties:**
| Property | Type | Description |
|----------|------|-------------|
| `prediction` | String | User prediction |
| `isCorrect` | Boolean | Prediction result |
| `pointsEarned` | Integer | Points earned |
| `createdAt` | DateTime | Creation timestamp |

---

### 2.5 User-Badge Relationship

```cypher
// EARNED: User earned a badge
MATCH (u:User {id: $userId}), (b:Badge {id: $badgeId})
CREATE (u)-[r:EARNED {
    earnedAt: datetime(),
    pointsAwarded: $points,
    source: $source,
    createdAt: datetime()
}]->(b);
```

**Properties:**
| Property | Type | Description |
|----------|------|-------------|
| `earnedAt` | DateTime | When badge was earned |
| `pointsAwarded` | Integer | Points awarded |
| `source` | String | How badge was earned |
| `createdAt` | DateTime | Creation timestamp |

---

### 2.6 Action-User Relationship

```cypher
// PERFORMED_BY: Action was performed by user
MATCH (a:Action {id: $actionId}), (u:User {id: $userId})
CREATE (a)-[r:PERFORMED_BY {
    userId: $userId,
    createdAt: datetime()
}]->(u);
```

**Properties:**
| Property | Type | Description |
|----------|------|-------------|
| `userId` | UUID | User who performed action |
| `createdAt` | DateTime | Creation timestamp |

---

### 2.7 Action-Match Relationship

```cypher
// OCCURRED_IN: Action occurred in match context
MATCH (a:Action {id: $actionId}), (m:Match {id: $matchId})
CREATE (a)-[r:OCCURRED_IN {
    matchId: $matchId,
    timestamp: datetime(),
    createdAt: datetime()
}]->(m);
```

**Properties:**
| Property | Type | Description |
|----------|------|-------------|
| `matchId` | UUID | Related match |
| `timestamp` | DateTime | When action occurred |
| `createdAt` | DateTime | Creation timestamp |

---

### 2.8 Achievement-Badge Relationship

```cypher
// UNLOCKS: Achievement unlocks a badge
MATCH (a:Achievement {id: $achievementId}), (b:Badge {id: $badgeId})
CREATE (a)-[r:UNLOCKS {
    badgeId: $badgeId,
    createdAt: datetime()
}]->(b);
```

**Properties:**
| Property | Type | Description |
|----------|------|-------------|
| `badgeId` | UUID | Badge unlocked |
| `createdAt` | DateTime | Creation timestamp |

---

## 3. Additional Relationships

### 2.9 User-Achievement Relationship

```cypher
// PROGRESS: User has progress on achievement
// COMPLETED: User completed achievement
MATCH (u:User {id: $userId}), (a:Achievement {id: $achievementId})
CREATE (u)-[r:PROGRESS {
    currentValue: 0,
    startedAt: datetime(),
    lastUpdated: datetime(),
    createdAt: datetime()
}]->(a);

MATCH (u:User {id: $userId}), (a:Achievement {id: $achievementId})
CREATE (u)-[r:COMPLETED {
    completedAt: datetime(),
    pointsAwarded: $points,
    createdAt: datetime()
}]->(a);
```

---

## 4. Indexes for Performance Optimization

```cypher
// Composite indexes for common query patterns

// User queries
CREATE INDEX user_points_idx IF NOT EXISTS
FOR (u:User) ON (u.points);

CREATE INDEX user_level_idx IF NOT EXISTS
FOR (u:User) ON (u.level);

// Match queries
CREATE INDEX match_league_idx IF NOT EXISTS
FOR (m:Match) ON (m.league);

CREATE INDEX match_date_range IF NOT EXISTS
FOR (m:Match) ON (m.scheduledAt);

// Player queries
CREATE INDEX player_team_idx IF NOT EXISTS
FOR (p:Player) ON (p.teamId);

// Action queries
CREATE INDEX action_user_match IF NOT EXISTS
FOR ()-[r:WATCHED]->(m:Match) ON (r.timestamp);

// Relationship indexes
CREATE INDEX supports_since IF NOT EXISTS
FOR ()-[r:SUPPORTS]->() ON (r.since);

CREATE INDEX earned_at IF NOT EXISTS
FOR ()-[r:EARNED]->() ON (r.earnedAt);
```

---

## 5. Sample Queries for Common Gamification Patterns

### 5.1 Find all users who cheered for Team X in Match Y and earned Badge Z

```cypher
MATCH (u:User)-[:SUPPORTS]->(t:Team {id: $teamId})
MATCH (u)-[:WATCHED]->(m:Match {id: $matchId})
MATCH (u)-[:EARNED]->(b:Badge {id: $badgeId})
WHERE (m)-[:HOME_TEAM|AWAY_TEAM]->(t)
RETURN u.id, u.username, b.name, b.rarity
ORDER BY b.rarity DESC;
```

### 5.2 Get user's gamification dashboard

```cypher
MATCH (u:User {id: $userId})
OPTIONAL MATCH (u)-[r:EARNED]->(b:Badge)
OPTIONAL MATCH (u)-[:SUPPORTS]->(t:Team)
OPTIONAL MATCH (u)-[:WATCHED]->(m:Match)
WHERE m.scheduledAt > datetime() - duration({days: 30})
RETURN {
    user: u,
    badges: collect(DISTINCT b),
    teams: collect(DISTINCT t),
    watchedMatches: count(m),
    totalPoints: u.points,
    level: u.level
} AS dashboard;
```

### 5.3 Find top supporters for a team

```cypher
MATCH (u:User)-[r:SUPPORTS]->(t:Team {id: $teamId})
MATCH (u)-[:WATCHED]->(m:Match)
WHERE (m)-[:HOME_TEAM|AWAY_TEAM]->(t)
WITH u, count(m) AS matchWatchCount
ORDER BY matchWatchCount DESC
LIMIT 10
RETURN u.id, u.username, matchWatchCount;
```

### 5.4 Get achievement progress for user

```cypher
MATCH (u:User {id: $userId})
MATCH (a:Achievement)
OPTIONAL MATCH (u)-[r:COMPLETED]->(a)
OPTIONAL MATCH (u)-[p:PROGRESS]->(a)
RETURN a.name, a.type, a.targetValue,
       coalesce(p.currentValue, 0) AS progress,
       CASE WHEN r IS NOT NULL THEN true ELSE false END AS completed;
```

### 5.5 Find users who predicted match correctly

```cypher
MATCH (u:User)-[r:PARTICIPATED]->(m:Match {id: $matchId})
WHERE r.isCorrect = true
RETURN u.id, u.username, r.pointsEarned
ORDER BY r.pointsEarned DESC;
```

### 5.6 Get team player roster

```cypher
MATCH (p:Player)-[r:PLAYS_FOR]->(t:Team {id: $teamId})
WHERE r.isActive = true
RETURN p.id, p.name, p.position, r.jerseyNumber
ORDER BY r.jerseyNumber;
```

### 5.7 Find trending matches (most watchers)

```cypher
MATCH (u:User)-[r:WATCHED]->(m:Match)
WHERE m.status = 'live'
WITH m, count(DISTINCT u) AS viewerCount
ORDER BY viewerCount DESC
LIMIT 10
RETURN m.id, m.homeTeamId, m.awayTeamId, viewerCount;
```

### 5.8 Get user's action timeline

```cypher
MATCH (u:User {id: $userId})-[r:WATCHED|PARTICIPATED]->(m:Match)
RETURN m.id, m.scheduledAt, type(r) AS actionType, r.pointsEarned
ORDER BY m.scheduledAt DESC
LIMIT 50;
```

---

## 6. Architecture Integration Notes

### 6.1 Brain & Muscle Separation

The Knowledge Graph schema supports the CQRS architecture:

- **Brain (Strategic)**: Neo4j + LLM handles complex queries, recommendations, and analytics
  - Complex pattern matching queries
  - User behavior analysis
  - Achievement progress tracking
  - Team/user relationship analysis

- **Muscle (Execution)**: Go + Redis + Kafka handles real-time operations
  - Action recording (Kafka → Neo4j)
  - Real-time leaderboards (Redis)
  - Instant badge awarding
  - Live match updates

### 6.2 Event-Driven Updates

```cypher
// Example: Recording user action via Kafka consumer
// 1. Action created in Redis for instant feedback
// 2. Kafka message triggers Neo4j write
// 3. LLM analyzes patterns asynchronously
```

### 6.3 Query Patterns

| Query Type | Use Case | Recommended Approach |
|------------|----------|---------------------|
| Real-time | User watching match | Redis + Kafka → Neo4j |
| Analytics | Team engagement stats | Neo4j parallel queries |
| ML/LLM | User behavior patterns | Neo4j export → Python |
| Recommendations | Next match to watch | Neo4j + LLM hybrid |

---

## 7. Schema Version

- **Version**: 1.0.0
- **Created**: 2026-03-23
- **Last Updated**: 2026-03-23
- **Author**: Architecture Team
