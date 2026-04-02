// =============================================================================
// Knowledge Graph Schema for Sports Gamification Platform
// Neo4j Cypher Migration Script
// Version: 1.0.0
// Created: 2026-03-24
// =============================================================================

// =============================================================================
// SECTION 1: CONSTRAINTS (Unique IDs)
// =============================================================================

// User Constraints
CREATE CONSTRAINT user_id_unique IF NOT EXISTS
FOR (u:User) REQUIRE u.id IS UNIQUE;

CREATE CONSTRAINT user_email_unique IF NOT EXISTS
FOR (u:User) REQUIRE u.email IS UNIQUE;

// Team Constraints
CREATE CONSTRAINT team_id_unique IF NOT EXISTS
FOR (t:Team) REQUIRE t.id IS UNIQUE;

CREATE CONSTRAINT team_name_unique IF NOT EXISTS
FOR (t:Team) REQUIRE t.name IS UNIQUE;

// Match Constraints
CREATE CONSTRAINT match_id_unique IF NOT EXISTS
FOR (m:Match) REQUIRE m.id IS UNIQUE;

// Player Constraints
CREATE CONSTRAINT player_id_unique IF NOT EXISTS
FOR (p:Player) REQUIRE p.id IS UNIQUE;

// Badge Constraints
CREATE CONSTRAINT badge_id_unique IF NOT EXISTS
FOR (b:Badge) REQUIRE b.id IS UNIQUE;

// Achievement Constraints
CREATE CONSTRAINT achievement_id_unique IF NOT EXISTS
FOR (a:Achievement) REQUIRE a.id IS UNIQUE;

// Action Constraints
CREATE CONSTRAINT action_id_unique IF NOT EXISTS
FOR (a:Action) REQUIRE a.id IS UNIQUE;

// =============================================================================
// SECTION 2: INDEXES FOR PERFORMANCE
// =============================================================================

// User Indexes
CREATE INDEX user_username_idx IF NOT EXISTS
FOR (u:User) ON (u.username);

CREATE INDEX user_created_at_idx IF NOT EXISTS
FOR (u:User) ON (u.createdAt);

CREATE INDEX user_points_idx IF NOT EXISTS
FOR (u:User) ON (u.points);

CREATE INDEX user_level_idx IF NOT EXISTS
FOR (u:User) ON (u.level);

// Team Indexes
CREATE INDEX team_sport_idx IF NOT EXISTS
FOR (t:Team) ON (t.sport);

CREATE INDEX team_country_idx IF NOT EXISTS
FOR (t:Team) ON (t.country);

// Match Indexes
CREATE INDEX match_scheduled_at_idx IF NOT EXISTS
FOR (m:Match) ON (m.scheduledAt);

CREATE INDEX match_status_idx IF NOT EXISTS
FOR (m:Match) ON (m.status);

CREATE INDEX match_league_idx IF NOT EXISTS
FOR (m:Match) ON (m.league);

// Player Indexes
CREATE INDEX player_name_idx IF NOT EXISTS
FOR (p:Player) ON (p.name);

CREATE INDEX player_position_idx IF NOT EXISTS
FOR (p:Player) ON (p.position);

CREATE INDEX player_nationality_idx IF NOT EXISTS
FOR (p:Player) ON (p.nationality);

// Badge Indexes
CREATE INDEX badge_category_idx IF NOT EXISTS
FOR (b:Badge) ON (b.category);

// Achievement Indexes
CREATE INDEX achievement_type_idx IF NOT EXISTS
FOR (a:Achievement) ON (a.type);

// Action Indexes
CREATE INDEX action_type_idx IF NOT EXISTS
FOR (a:Action) ON (a.type);

CREATE INDEX action_timestamp_idx IF NOT EXISTS
FOR (a:Action) ON (a.timestamp);

// Relationship Indexes
CREATE INDEX supports_since_idx IF NOT EXISTS
FOR ()-[r:SUPPORTS]->() ON (r.since);

CREATE INDEX earned_at_idx IF NOT EXISTS
FOR ()-[r:EARNED]->() ON (r.earnedAt);

CREATE INDEX watched_timestamp_idx IF NOT EXISTS
FOR ()-[r:WATCHED]->() ON (r.timestamp);

CREATE INDEX participated_timestamp_idx IF NOT EXISTS
FOR ()-[r:PARTICIPATED]->() ON (r.timestamp);

// =============================================================================
// SECTION 3: SAMPLE DATA - TEAMS
// =============================================================================

// Create Teams
CREATE (t1:Team {
    id: 'team-001',
    name: 'Galatasaray',
    shortName: 'GS',
    sport: 'soccer',
    league: 'Super Lig',
    logoUrl: 'https://example.com/logos/galatasaray.png',
    primaryColor: 'red',
    secondaryColor: 'yellow',
    foundedYear: 1905,
    city: 'Istanbul',
    country: 'Turkey',
    stadium: 'RAMS Park',
    stats: {wins: 23, draws: 8, losses: 3, goalsFor: 72},
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (t2:Team {
    id: 'team-002',
    name: 'Fenerbahçe',
    shortName: 'FB',
    sport: 'soccer',
    league: 'Super Lig',
    logoUrl: 'https://example.com/logos/fenerbahce.png',
    primaryColor: 'yellow',
    secondaryColor: 'navy',
    foundedYear: 1907,
    city: 'Istanbul',
    country: 'Turkey',
    stadium: 'Şükrü Saracoğlu',
    stats: {wins: 21, draws: 9, losses: 4, goalsFor: 68},
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (t3:Team {
    id: 'team-003',
    name: 'Beşiktaş',
    shortName: 'BJK',
    sport: 'soccer',
    league: 'Super Lig',
    logoUrl: 'https://example.com/logos/besiktas.png',
    primaryColor: 'black',
    secondaryColor: 'white',
    foundedYear: 1903,
    city: 'Istanbul',
    country: 'Turkey',
    stadium: 'Tüpraş Stadyumu',
    stats: {wins: 19, draws: 10, losses: 5, goalsFor: 61},
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (t4:Team {
    id: 'team-004',
    name: 'Los Angeles Lakers',
    shortName: 'LAL',
    sport: 'basketball',
    league: 'NBA',
    logoUrl: 'https://example.com/logos/lakers.png',
    primaryColor: 'purple',
    secondaryColor: 'gold',
    foundedYear: 1947,
    city: 'Los Angeles',
    country: 'USA',
    stadium: 'Crypto.com Arena',
    stats: {wins: 35, losses: 15, pointsFor: 7890},
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (t5:Team {
    id: 'team-005',
    name: 'Boston Celtics',
    shortName: 'BOS',
    sport: 'basketball',
    league: 'NBA',
    logoUrl: 'https://example.com/logos/celtics.png',
    primaryColor: 'green',
    secondaryColor: 'white',
    foundedYear: 1946,
    city: 'Boston',
    country: 'USA',
    stadium: 'TD Garden',
    stats: {wins: 38, losses: 12, pointsFor: 8120},
    createdAt: datetime('2024-01-01T00:00:00Z')
});

// =============================================================================
// SECTION 4: SAMPLE DATA - PLAYERS
// =============================================================================

// Create Players
CREATE (p1:Player {
    id: 'player-001',
    name: 'Victor Osimhen',
    firstName: 'Victor',
    lastName: 'Osimhen',
    position: 'Forward',
    jerseyNumber: 9,
    sport: 'soccer',
    nationality: 'Nigeria',
    dateOfBirth: date('1998-12-29'),
    height: 186,
    weight: 78,
    photoUrl: 'https://example.com/players/osimhen.png',
    stats: {goals: 22, assists: 8, matches: 30},
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (p2:Player {
    id: 'player-002',
    name: 'Hakan Çalhanoğlu',
    firstName: 'Hakan',
    lastName: 'Çalhanoğlu',
    position: 'Midfielder',
    jerseyNumber: 10,
    sport: 'soccer',
    nationality: 'Turkey',
    dateOfBirth: date('1994-02-08'),
    height: 178,
    weight: 75,
    photoUrl: 'https://example.com/players/calhanoglu.png',
    stats: {goals: 12, assists: 15, matches: 32},
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (p3:Player {
    id: 'player-003',
    name: 'LeBron James',
    firstName: 'LeBron',
    lastName: 'James',
    position: 'Small Forward',
    jerseyNumber: 23,
    sport: 'basketball',
    nationality: 'USA',
    dateOfBirth: date('1984-12-30'),
    height: 206,
    weight: 113,
    photoUrl: 'https://example.com/players/lebron.png',
    stats: {points: 2450, assists: 620, rebounds: 780, matches: 55},
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (p4:Player {
    id: 'player-004',
    name: 'Stephen Curry',
    firstName: 'Stephen',
    lastName: 'Curry',
    position: 'Point Guard',
    jerseyNumber: 30,
    sport: 'basketball',
    nationality: 'USA',
    dateOfBirth: date('1988-03-14'),
    height: 188,
    weight: 86,
    photoUrl: 'https://example.com/players/curry.png',
    stats: {points: 2180, assists: 540, rebounds: 340, matches: 52},
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (p5:Player {
    id: 'player-005',
    name: 'Giannis Antetokounmpo',
    firstName: 'Giannis',
    lastName: 'Antetokounmpo',
    position: 'Power Forward',
    jerseyNumber: 34,
    sport: 'basketball',
    nationality: 'Greece',
    dateOfBirth: date('1994-12-06'),
    height: 211,
    weight: 110,
    photoUrl: 'https://example.com/players/giannis.png',
    stats: {points: 2680, assists: 480, rebounds: 920, matches: 58},
    createdAt: datetime('2024-01-01T00:00:00Z')
});

// =============================================================================
// SECTION 5: SAMPLE DATA - MATCHES
// =============================================================================

// Create Matches
CREATE (m1:Match {
    id: 'match-001',
    homeTeamId: 'team-001',
    awayTeamId: 'team-002',
    sport: 'soccer',
    league: 'Super Lig',
    scheduledAt: datetime('2026-03-28T19:00:00Z'),
    endedAt: null,
    status: 'scheduled',
    homeScore: 0,
    awayScore: 0,
    venue: 'RAMS Park',
    round: 'Week 30',
    stats: {},
    highlights: [],
    viewerCount: 0,
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (m2:Match {
    id: 'match-002',
    homeTeamId: 'team-003',
    awayTeamId: 'team-001',
    sport: 'soccer',
    league: 'Super Lig',
    scheduledAt: datetime('2026-03-21T19:00:00Z'),
    endedAt: datetime('2026-03-21T21:00:00Z'),
    status: 'completed',
    homeScore: 2,
    awayScore: 3,
    venue: 'Tüpraş Stadyumu',
    round: 'Week 29',
    stats: {possession: {home: 45, away: 55}, shots: {home: 12, away: 15}},
    highlights: ['https://example.com/highlights/match-002-goal1.mp4'],
    viewerCount: 5200000,
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (m3:Match {
    id: 'match-003',
    homeTeamId: 'team-004',
    awayTeamId: 'team-005',
    sport: 'basketball',
    league: 'NBA',
    scheduledAt: datetime('2026-03-25T03:30:00Z'),
    endedAt: null,
    status: 'scheduled',
    homeScore: 0,
    awayScore: 0,
    venue: 'Crypto.com Arena',
    round: 'Regular Season',
    stats: {},
    highlights: [],
    viewerCount: 0,
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (m4:Match {
    id: 'match-004',
    homeTeamId: 'team-002',
    awayTeamId: 'team-003',
    sport: 'soccer',
    league: 'Super Lig',
    scheduledAt: datetime('2026-03-15T15:00:00Z'),
    endedAt: datetime('2026-03-15T17:00:00Z'),
    status: 'completed',
    homeScore: 1,
    awayScore: 1,
    venue: 'Şükrü Saracoğlu',
    round: 'Week 28',
    stats: {possession: {home: 52, away: 48}, shots: {home: 8, away: 10}},
    highlights: [],
    viewerCount: 4800000,
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (m5:Match {
    id: 'match-005',
    homeTeamId: 'team-005',
    awayTeamId: 'team-004',
    sport: 'basketball',
    league: 'NBA',
    scheduledAt: datetime('2026-03-24T00:00:00Z'),
    endedAt: datetime('2026-03-24T02:30:00Z'),
    status: 'completed',
    homeScore: 118,
    awayScore: 112,
    venue: 'TD Garden',
    round: 'Regular Season',
    stats: {quarters: [28, 32, 30, 28], leadChanges: 12},
    highlights: ['https://example.com/highlights/match-005.mp4'],
    viewerCount: 18500000,
    createdAt: datetime('2024-01-01T00:00:00Z')
});

// =============================================================================
// SECTION 6: SAMPLE DATA - BADGES
// =============================================================================

// Create Badges
CREATE (b1:Badge {
    id: 'badge-001',
    name: 'First Match Watch',
    description: 'Watch your first match',
    iconUrl: 'https://example.com/badges/first-watch.png',
    category: 'fan',
    rarity: 'common',
    pointsValue: 10,
    criteria: {action: 'watch', count: 1},
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (b2:Badge {
    id: 'badge-002',
    name: 'Super Fan',
    description: 'Watch 50 matches',
    iconUrl: 'https://example.com/badges/super-fan.png',
    category: 'fan',
    rarity: 'rare',
    pointsValue: 500,
    criteria: {action: 'watch', count: 50},
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (b3:Badge {
    id: 'badge-003',
    name: 'Prediction Master',
    description: 'Get 10 predictions correct',
    iconUrl: 'https://example.com/badges/prediction-master.png',
    category: 'fan',
    rarity: 'epic',
    pointsValue: 1000,
    criteria: {action: 'predict', correctCount: 10},
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (b4:Badge {
    id: 'badge-004',
    name: 'Team Legend',
    description: 'Support a team for 1 year',
    iconUrl: 'https://example.com/badges/team-legend.png',
    category: 'fan',
    rarity: 'legendary',
    pointsValue: 2500,
    criteria: {action: 'support', duration: '1 year'},
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (b5:Badge {
    id: 'badge-005',
    name: 'Social Butterfly',
    description: 'Share 25 match reactions',
    iconUrl: 'https://example.com/badges/social-butterfly.png',
    category: 'social',
    rarity: 'rare',
    pointsValue: 300,
    criteria: {action: 'share', count: 25},
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (b6:Badge {
    id: 'badge-006',
    name: 'Top Scorer',
    description: 'Earn the most points in a week',
    iconUrl: 'https://example.com/badges/top-scorer.png',
    category: 'player',
    rarity: 'legendary',
    pointsValue: 5000,
    criteria: {action: 'points', period: 'week', rank: 1},
    createdAt: datetime('2024-01-01T00:00:00Z')
});

// =============================================================================
// SECTION 7: SAMPLE DATA - ACHIEVEMENTS
// =============================================================================

// Create Achievements
CREATE (a1:Achievement {
    id: 'achievement-001',
    name: 'Match Watcher',
    description: 'Watch 10 matches',
    type: 'watch',
    targetValue: 10,
    currentValue: 0,
    pointsReward: 100,
    isActive: true,
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (a2:Achievement {
    id: 'achievement-002',
    name: 'Prediction Pro',
    description: 'Get 5 predictions correct in a row',
    type: 'milestone',
    targetValue: 5,
    currentValue: 0,
    pointsReward: 250,
    isActive: true,
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (a3:Achievement {
    id: 'achievement-003',
    name: 'Social Star',
    description: 'Share 50 match reactions',
    type: 'social',
    targetValue: 50,
    currentValue: 0,
    pointsReward: 500,
    isActive: true,
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (a4:Achievement {
    id: 'achievement-004',
    name: 'Loyal Supporter',
    description: 'Support a team for 6 months',
    type: 'milestone',
    targetValue: 180,
    currentValue: 0,
    pointsReward: 750,
    isActive: true,
    createdAt: datetime('2024-01-01T00:00:00Z')
})

CREATE (a5:Achievement {
    id: 'achievement-005',
    name: 'Point Collector',
    description: 'Earn 10000 total points',
    type: 'milestone',
    targetValue: 10000,
    currentValue: 0,
    pointsReward: 2000,
    isActive: true,
    createdAt: datetime('2024-01-01T00:00:00Z')
});

// =============================================================================
// SECTION 8: SAMPLE DATA - USERS
// =============================================================================

// Create Users
CREATE (u1:User {
    id: 'user-001',
    email: 'john.doe@example.com',
    username: 'johndoe',
    passwordHash: '$2a$10$hashedpassword123',
    role: 'fan',
    avatarUrl: 'https://example.com/avatars/user001.png',
    points: 1250,
    level: 5,
    createdAt: datetime('2024-06-15T10:30:00Z'),
    updatedAt: datetime('2026-03-20T14:22:00Z'),
    lastActiveAt: datetime('2026-03-24T10:15:00Z'),
    isActive: true,
    preferences: {notifications: true, language: 'en'}
})

CREATE (u2:User {
    id: 'user-002',
    email: 'jane.smith@example.com',
    username: 'janesmith',
    passwordHash: '$2a$10$hashedpassword456',
    role: 'fan',
    avatarUrl: 'https://example.com/avatars/user002.png',
    points: 3450,
    level: 8,
    createdAt: datetime('2024-03-10T08:00:00Z'),
    updatedAt: datetime('2026-03-23T16:45:00Z'),
    lastActiveAt: datetime('2026-03-24T09:30:00Z'),
    isActive: true,
    preferences: {notifications: true, language: 'tr'}
})

CREATE (u3:User {
    id: 'user-003',
    email: 'alex.football@example.com',
    username: 'alexfan',
    passwordHash: '$2a$10$hashedpassword789',
    role: 'fan',
    avatarUrl: 'https://example.com/avatars/user003.png',
    points: 890,
    level: 3,
    createdAt: datetime('2025-01-20T12:00:00Z'),
    updatedAt: datetime('2026-03-22T11:30:00Z'),
    lastActiveAt: datetime('2026-03-23T20:00:00Z'),
    isActive: true,
    preferences: {notifications: false, language: 'en'}
})

CREATE (u4:User {
    id: 'user-004',
    email: 'player.one@example.com',
    username: 'playerone',
    passwordHash: '$2a$10$hashedpasswordabc',
    role: 'player',
    avatarUrl: 'https://example.com/avatars/user004.png',
    points: 5200,
    level: 12,
    createdAt: datetime('2024-02-01T09:00:00Z'),
    updatedAt: datetime('2026-03-24T08:00:00Z'),
    lastActiveAt: datetime('2026-03-24T12:00:00Z'),
    isActive: true,
    preferences: {notifications: true, language: 'en'}
})

CREATE (u5:User {
    id: 'user-005',
    email: 'admin@gamification.com',
    username: 'admin',
    passwordHash: '$2a$10$hashedpasswordxyz',
    role: 'admin',
    avatarUrl: 'https://example.com/avatars/admin.png',
    points: 0,
    level: 1,
    createdAt: datetime('2024-01-01T00:00:00Z'),
    updatedAt: datetime('2026-03-24T00:00:00Z'),
    lastActiveAt: datetime('2026-03-24T00:00:00Z'),
    isActive: true,
    preferences: {notifications: true, language: 'en'}
});

// =============================================================================
// SECTION 9: SAMPLE DATA - ACTIONS
// =============================================================================

// Create Actions
CREATE (act1:Action {
    id: 'action-001',
    type: 'watch',
    userId: 'user-001',
    matchId: 'match-002',
    teamId: null,
    pointsEarned: 10,
    metadata: {duration: 7200, device: 'mobile'},
    timestamp: datetime('2026-03-21T19:05:00Z'),
    ipAddress: '192.168.1.100',
    userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0)'
})

CREATE (act2:Action {
    id: 'action-002',
    type: 'cheer',
    userId: 'user-001',
    matchId: 'match-002',
    teamId: 'team-001',
    pointsEarned: 5,
    metadata: {reaction: 'goal'},
    timestamp: datetime('2026-03-21T20:15:00Z'),
    ipAddress: '192.168.1.100',
    userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0)'
})

CREATE (act3:Action {
    id: 'action-003',
    type: 'predict',
    userId: 'user-002',
    matchId: 'match-002',
    teamId: null,
    pointsEarned: 25,
    metadata: {prediction: '3-2', isCorrect: true},
    timestamp: datetime('2026-03-21T18:00:00Z'),
    ipAddress: '192.168.1.101',
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)'
})

CREATE (act4:Action {
    id: 'action-004',
    type: 'share',
    userId: 'user-002',
    matchId: 'match-002',
    teamId: null,
    pointsEarned: 15,
    metadata: {platform: 'twitter', content: 'Great match!'},
    timestamp: datetime('2026-03-21T21:30:00Z'),
    ipAddress: '192.168.1.101',
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)'
})

CREATE (act5:Action {
    id: 'action-005',
    type: 'watch',
    userId: 'user-003',
    matchId: 'match-004',
    teamId: null,
    pointsEarned: 10,
    metadata: {duration: 5400, device: 'web'},
    timestamp: datetime('2026-03-15T15:10:00Z'),
    ipAddress: '192.168.1.102',
    userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)'
})

CREATE (act6:Action {
    id: 'action-006',
    type: 'predict',
    userId: 'user-003',
    matchId: 'match-004',
    teamId: null,
    pointsEarned: 0,
    metadata: {prediction: '2-0', isCorrect: false},
    timestamp: datetime('2026-03-15T14:00:00Z'),
    ipAddress: '192.168.1.102',
    userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)'
})

CREATE (act7:Action {
    id: 'action-007',
    type: 'watch',
    userId: 'user-001',
    matchId: 'match-005',
    teamId: null,
    pointsEarned: 10,
    metadata: {duration: 9000, device: 'tv'},
    timestamp: datetime('2026-03-24T00:05:00Z'),
    ipAddress: '192.168.1.100',
    userAgent: 'Mozilla/5.0 (Smart TV)'
})

CREATE (act8:Action {
    id: 'action-008',
    type: 'cheer',
    userId: 'user-002',
    matchId: 'match-005',
    teamId: 'team-005',
    pointsEarned: 5,
    metadata: {reaction: 'three-pointer'},
    timestamp: datetime('2026-03-24T01:00:00Z'),
    ipAddress: '192.168.1.101',
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)'
});

// =============================================================================
// SECTION 10: RELATIONSHIPS - PLAYER-PLAYER_FOR-TEAM
// =============================================================================

MATCH (p:Player {id: 'player-001'}), (t:Team {id: 'team-001'})
CREATE (p)-[r:PLAYS_FOR {
    jerseyNumber: 9,
    position: 'Forward',
    startDate: date('2023-07-01'),
    endDate: date('2025-06-30'),
    isActive: true,
    createdAt: datetime('2023-07-01T00:00:00Z')
}]->(t);

MATCH (p:Player {id: 'player-002'}), (t:Team {id: 'team-001'})
CREATE (p)-[r:PLAYS_FOR {
    jerseyNumber: 10,
    position: 'Midfielder',
    startDate: date('2021-07-01'),
    endDate: date('2025-06-30'),
    isActive: true,
    createdAt: datetime('2021-07-01T00:00:00Z')
}]->(t);

MATCH (p:Player {id: 'player-003'}), (t:Team {id: 'team-004'})
CREATE (p)-[r:PLAYS_FOR {
    jerseyNumber: 23,
    position: 'Small Forward',
    startDate: date('2018-07-01'),
    endDate: date('2025-06-30'),
    isActive: true,
    createdAt: datetime('2018-07-01T00:00:00Z')
}]->(t);

MATCH (p:Player {id: 'player-004'}), (t:Team {id: 'team-004'})
CREATE (p)-[r:PLAYS_FOR {
    jerseyNumber: 30,
    position: 'Point Guard',
    startDate: date('2009-07-01'),
    endDate: date('2025-06-30'),
    isActive: true,
    createdAt: datetime('2009-07-01T00:00:00Z')
}]->(t);

MATCH (p:Player {id: 'player-005'}), (t:Team {id: 'team-005'})
CREATE (p)-[r:PLAYS_FOR {
    jerseyNumber: 34,
    position: 'Power Forward',
    startDate: date('2013-07-01'),
    endDate: date('2025-06-30'),
    isActive: true,
    createdAt: datetime('2013-07-01T00:00:00Z')
}]->(t);

// =============================================================================
// SECTION 11: RELATIONSHIPS - MATCH-HOME_TEAM-AWAY_TEAM
// =============================================================================

// Match 1: Galatasaray vs Fenerbahçe
MATCH (m:Match {id: 'match-001'}), (t:Team {id: 'team-001'})
CREATE (m)-[r:HOME_TEAM {
    score: 0,
    isWinner: null,
    createdAt: datetime('2024-01-01T00:00:00Z')
}]->(t);

MATCH (m:Match {id: 'match-001'}), (t:Team {id: 'team-002'})
CREATE (m)-[r:AWAY_TEAM {
    score: 0,
    isWinner: null,
    createdAt: datetime('2024-01-01T00:00:00Z')
}]->(t);

// Match 2: Beşiktaş vs Galatasaray (completed)
MATCH (m:Match {id: 'match-002'}), (t:Team {id: 'team-003'})
CREATE (m)-[r:HOME_TEAM {
    score: 2,
    isWinner: false,
    createdAt: datetime('2024-01-01T00:00:00Z')
}]->(t);

MATCH (m:Match {id: 'match-002'}), (t:Team {id: 'team-001'})
CREATE (m)-[r:AWAY_TEAM {
    score: 3,
    isWinner: true,
    createdAt: datetime('2024-01-01T00:00:00Z')
}]->(t);

// Match 3: Lakers vs Celtics
MATCH (m:Match {id: 'match-003'}), (t:Team {id: 'team-004'})
CREATE (m)-[r:HOME_TEAM {
    score: 0,
    isWinner: null,
    createdAt: datetime('2024-01-01T00:00:00Z')
}]->(t);

MATCH (m:Match {id: 'match-003'}), (t:Team {id: 'team-005'})
CREATE (m)-[r:AWAY_TEAM {
    score: 0,
    isWinner: null,
    createdAt: datetime('2024-01-01T00:00:00Z')
}]->(t);

// Match 4: Fenerbahçe vs Beşiktaş (completed)
MATCH (m:Match {id: 'match-004'}), (t:Team {id: 'team-002'})
CREATE (m)-[r:HOME_TEAM {
    score: 1,
    isWinner: null,
    createdAt: datetime('2024-01-01T00:00:00Z')
}]->(t);

MATCH (m:Match {id: 'match-004'}), (t:Team {id: 'team-003'})
CREATE (m)-[r:AWAY_TEAM {
    score: 1,
    isWinner: null,
    createdAt: datetime('2024-01-01T00:00:00Z')
}]->(t);

// Match 5: Celtics vs Lakers (completed)
MATCH (m:Match {id: 'match-005'}), (t:Team {id: 'team-005'})
CREATE (m)-[r:HOME_TEAM {
    score: 118,
    isWinner: true,
    createdAt: datetime('2024-01-01T00:00:00Z')
}]->(t);

MATCH (m:Match {id: 'match-005'}), (t:Team {id: 'team-004'})
CREATE (m)-[r:AWAY_TEAM {
    score: 112,
    isWinner: false,
    createdAt: datetime('2024-01-01T00:00:00Z')
}]->(t);

// =============================================================================
// SECTION 12: RELATIONSHIPS - USER-SUPPORTS-TEAM
// =============================================================================

MATCH (u:User {id: 'user-001'}), (t:Team {id: 'team-001'})
CREATE (u)-[r:SUPPORTS {
    since: datetime('2024-06-20T00:00:00Z'),
    isPrimary: true,
    notificationsEnabled: true,
    createdAt: datetime('2024-06-20T00:00:00Z')
}]->(t);

MATCH (u:User {id: 'user-001'}), (t:Team {id: 'team-004'})
CREATE (u)-[r:SUPPORTS {
    since: datetime('2025-01-15T00:00:00Z'),
    isPrimary: false,
    notificationsEnabled: false,
    createdAt: datetime('2025-01-15T00:00:00Z')
}]->(t);

MATCH (u:User {id: 'user-002'}), (t:Team {id: 'team-001'})
CREATE (u)-[r:SUPPORTS {
    since: datetime('2024-03-15T00:00:00Z'),
    isPrimary: true,
    notificationsEnabled: true,
    createdAt: datetime('2024-03-15T00:00:00Z')
}]->(t);

MATCH (u:User {id: 'user-002'}), (t:Team {id: 'team-005'})
CREATE (u)-[r:SUPPORTS {
    since: datetime('2024-08-01T00:00:00Z'),
    isPrimary: false,
    notificationsEnabled: true,
    createdAt: datetime('2024-08-01T00:00:00Z')
}]->(t);

MATCH (u:User {id: 'user-003'}), (t:Team {id: 'team-002'})
CREATE (u)-[r:SUPPORTS {
    since: datetime('2025-02-01T00:00:00Z'),
    isPrimary: true,
    notificationsEnabled: true,
    createdAt: datetime('2025-02-01T00:00:00Z')
}]->(t);

MATCH (u:User {id: 'user-004'}), (t:Team {id: 'team-001'})
CREATE (u)-[r:SUPPORTS {
    since: datetime('2024-02-05T00:00:00Z'),
    isPrimary: true,
    notificationsEnabled: true,
    createdAt: datetime('2024-02-05T00:00:00Z')
}]->(t);

// =============================================================================
// SECTION 13: RELATIONSHIPS - USER-WATCHED-MATCH
// =============================================================================

MATCH (u:User {id: 'user-001'}), (m:Match {id: 'match-002'})
CREATE (u)-[r:WATCHED {
    watchDuration: 7200,
    watchPercentage: 100.0,
    timestamp: datetime('2026-03-21T19:05:00Z'),
    device: 'mobile',
    createdAt: datetime('2026-03-21T19:05:00Z')
}]->(m);

MATCH (u:User {id: 'user-002'}), (m:Match {id: 'match-002'})
CREATE (u)-[r:WATCHED {
    watchDuration: 7200,
    watchPercentage: 100.0,
    timestamp: datetime('2026-03-21T19:00:00Z'),
    device: 'web',
    createdAt: datetime('2026-03-21T19:00:00Z')
}]->(m);

MATCH (u:User {id: 'user-003'}), (m:Match {id: 'match-004'})
CREATE (u)-[r:WATCHED {
    watchDuration: 5400,
    watchPercentage: 75.0,
    timestamp: datetime('2026-03-15T15:10:00Z'),
    device: 'web',
    createdAt: datetime('2026-03-15T15:10:00Z')
}]->(m);

MATCH (u:User {id: 'user-001'}), (m:Match {id: 'match-005'})
CREATE (u)-[r:WATCHED {
    watchDuration: 9000,
    watchPercentage: 100.0,
    timestamp: datetime('2026-03-24T00:05:00Z'),
    device: 'tv',
    createdAt: datetime('2026-03-24T00:05:00Z')
}]->(m);

MATCH (u:User {id: 'user-002'}), (m:Match {id: 'match-005'})
CREATE (u)-[r:WATCHED {
    watchDuration: 9000,
    watchPercentage: 100.0,
    timestamp: datetime('2026-03-24T00:00:00Z'),
    device: 'web',
    createdAt: datetime('2026-03-24T00:00:00Z')
}]->(m);

// =============================================================================
// SECTION 14: RELATIONSHIPS - USER-PARTICIPATED-MATCH
// =============================================================================

MATCH (u:User {id: 'user-002'}), (m:Match {id: 'match-002'})
CREATE (u)-[r:PARTICIPATED {
    prediction: '3-2',
    isCorrect: true,
    pointsEarned: 25,
    createdAt: datetime('2026-03-21T18:00:00Z')
}]->(m);

MATCH (u:User {id: 'user-003'}), (m:Match {id: 'match-004'})
CREATE (u)-[r:PARTICIPATED {
    prediction: '2-0',
    isCorrect: false,
    pointsEarned: 0,
    createdAt: datetime('2026-03-15T14:00:00Z')
}]->(m);

MATCH (u:User {id: 'user-001'}), (m:Match {id: 'match-001'})
CREATE (u)-[r:PARTICIPATED {
    prediction: '2-1',
    isCorrect: null,
    pointsEarned: 0,
    createdAt: datetime('2026-03-27T10:00:00Z')
}]->(m);

MATCH (u:User {id: 'user-002'}), (m:Match {id: 'match-003'})
CREATE (u)-[r:PARTICIPATED {
    prediction: 'LAL',
    isCorrect: null,
    pointsEarned: 0,
    createdAt: datetime('2026-03-24T12:00:00Z')
}]->(m);

// =============================================================================
// SECTION 15: RELATIONSHIPS - USER-EARNED-BADGE
// =============================================================================

MATCH (u:User {id: 'user-001'}), (b:Badge {id: 'badge-001'})
CREATE (u)-[r:EARNED {
    earnedAt: datetime('2024-06-16T10:30:00Z'),
    pointsAwarded: 10,
    source: 'first_match_watch',
    createdAt: datetime('2024-06-16T10:30:00Z')
}]->(b);

MATCH (u:User {id: 'user-002'}), (b:Badge {id: 'badge-001'})
CREATE (u)-[r:EARNED {
    earnedAt: datetime('2024-03-12T08:00:00Z'),
    pointsAwarded: 10,
    source: 'first_match_watch',
    createdAt: datetime('2024-03-12T08:00:00Z')
}]->(b);

MATCH (u:User {id: 'user-002'}), (b:Badge {id: 'badge-002'})
CREATE (u)-[r:EARNED {
    earnedAt: datetime('2025-12-01T12:00:00Z'),
    pointsAwarded: 500,
    source: 'match_watch_milestone',
    createdAt: datetime('2025-12-01T12:00:00Z')
}]->(b);

MATCH (u:User {id: 'user-002'}), (b:Badge {id: 'badge-003'})
CREATE (u)-[r:EARNED {
    earnedAt: datetime('2026-02-15T16:00:00Z'),
    pointsAwarded: 1000,
    source: 'prediction_master',
    createdAt: datetime('2026-02-15T16:00:00Z')
}]->(b);

MATCH (u:User {id: 'user-004'}), (b:Badge {id: 'badge-006'})
CREATE (u)-[r:EARNED {
    earnedAt: datetime('2025-11-30T23:59:00Z'),
    pointsAwarded: 5000,
    source: 'weekly_leaderboard',
    createdAt: datetime('2025-11-30T23:59:00Z')
}]->(b);

// =============================================================================
// SECTION 16: RELATIONSHIPS - ACTION-PERFORMED_BY-USER
// =============================================================================

MATCH (a:Action {id: 'action-001'}), (u:User {id: 'user-001'})
CREATE (a)-[r:PERFORMED_BY {
    userId: 'user-001',
    createdAt: datetime('2026-03-21T19:05:00Z')
}]->(u);

MATCH (a:Action {id: 'action-002'}), (u:User {id: 'user-001'})
CREATE (a)-[r:PERFORMED_BY {
    userId: 'user-001',
    createdAt: datetime('2026-03-21T20:15:00Z')
}]->(u);

MATCH (a:Action {id: 'action-003'}), (u:User {id: 'user-002'})
CREATE (a)-[r:PERFORMED_BY {
    userId: 'user-002',
    createdAt: datetime('2026-03-21T18:00:00Z')
}]->(u);

MATCH (a:Action {id: 'action-004'}), (u:User {id: 'user-002'})
CREATE (a)-[r:PERFORMED_BY {
    userId: 'user-002',
    createdAt: datetime('2026-03-21T21:30:00Z')
}]->(u);

MATCH (a:Action {id: 'action-005'}), (u:User {id: 'user-003'})
CREATE (a)-[r:PERFORMED_BY {
    userId: 'user-003',
    createdAt: datetime('2026-03-15T15:10:00Z')
}]->(u);

MATCH (a:Action {id: 'action-006'}), (u:User {id: 'user-003'})
CREATE (a)-[r:PERFORMED_BY {
    userId: 'user-003',
    createdAt: datetime('2026-03-15T14:00:00Z')
}]->(u);

MATCH (a:Action {id: 'action-007'}), (u:User {id: 'user-001'})
CREATE (a)-[r:PERFORMED_BY {
    userId: 'user-001',
    createdAt: datetime('2026-03-24T00:05:00Z')
}]->(u);

MATCH (a:Action {id: 'action-008'}), (u:User {id: 'user-002'})
CREATE (a)-[r:PERFORMED_BY {
    userId: 'user-002',
    createdAt: datetime('2026-03-24T01:00:00Z')
}]->(u);

// =============================================================================
// SECTION 17: RELATIONSHIPS - ACTION-OCCURRED-IN-MATCH
// =============================================================================

MATCH (a:Action {id: 'action-001'}), (m:Match {id: 'match-002'})
CREATE (a)-[r:OCCURRED_IN {
    matchId: 'match-002',
    timestamp: datetime('2026-03-21T19:05:00Z'),
    createdAt: datetime('2026-03-21T19:05:00Z')
}]->(m);

MATCH (a:Action {id: 'action-002'}), (m:Match {id: 'match-002'})
CREATE (a)-[r:OCCURRED_IN {
    matchId: 'match-002',
    timestamp: datetime('2026-03-21T20:15:00Z'),
    createdAt: datetime('2026-03-21T20:15:00Z')
}]->(m);

MATCH (a:Action {id: 'action-003'}), (m:Match {id: 'match-002'})
CREATE (a)-[r:OCCURRED_IN {
    matchId: 'match-002',
    timestamp: datetime('2026-03-21T18:00:00Z'),
    createdAt: datetime('2026-03-21T18:00:00Z')
}]->(m);

MATCH (a:Action {id: 'action-004'}), (m:Match {id: 'match-002'})
CREATE (a)-[r:OCCURRED_IN {
    matchId: 'match-002',
    timestamp: datetime('2026-03-21T21:30:00Z'),
    createdAt: datetime('2026-03-21T21:30:00Z')
}]->(m);

MATCH (a:Action {id: 'action-005'}), (m:Match {id: 'match-004'})
CREATE (a)-[r:OCCURRED_IN {
    matchId: 'match-004',
    timestamp: datetime('2026-03-15T15:10:00Z'),
    createdAt: datetime('2026-03-15T15:10:00Z')
}]->(m);

MATCH (a:Action {id: 'action-006'}), (m:Match {id: 'match-004'})
CREATE (a)-[r:OCCURRED_IN {
    matchId: 'match-004',
    timestamp: datetime('2026-03-15T14:00:00Z'),
    createdAt: datetime('2026-03-15T14:00:00Z')
}]->(m);

MATCH (a:Action {id: 'action-007'}), (m:Match {id: 'match-005'})
CREATE (a)-[r:OCCURRED_IN {
    matchId: 'match-005',
    timestamp: datetime('2026-03-24T00:05:00Z'),
    createdAt: datetime('2026-03-24T00:05:00Z')
}]->(m);

MATCH (a:Action {id: 'action-008'}), (m:Match {id: 'match-005'})
CREATE (a)-[r:OCCURRED_IN {
    matchId: 'match-005',
    timestamp: datetime('2026-03-24T01:00:00Z'),
    createdAt: datetime('2026-03-24T01:00:00Z')
}]->(m);

// =============================================================================
// SECTION 18: RELATIONSHIPS - ACHIEVEMENT-UNLOCKS-BADGE
// =============================================================================

MATCH (a:Achievement {id: 'achievement-001'}), (b:Badge {id: 'badge-001'})
CREATE (a)-[r:UNLOCKS {
    badgeId: 'badge-001',
    createdAt: datetime('2024-01-01T00:00:00Z')
}]->(b);

MATCH (a:Achievement {id: 'achievement-001'}), (b:Badge {id: 'badge-002'})
CREATE (a)-[r:UNLOCKS {
    badgeId: 'badge-002',
    createdAt: datetime('2024-01-01T00:00:00Z')
}]->(b);

MATCH (a:Achievement {id: 'achievement-002'}), (b:Badge {id: 'badge-003'})
CREATE (a)-[r:UNLOCKS {
    badgeId: 'badge-003',
    createdAt: datetime('2024-01-01T00:00:00Z')
}]->(b);

MATCH (a:Achievement {id: 'achievement-003'}), (b:Badge {id: 'badge-005'})
CREATE (a)-[r:UNLOCKS {
    badgeId: 'badge-005',
    createdAt: datetime('2024-01-01T00:00:00Z')
}]->(b);

MATCH (a:Achievement {id: 'achievement-004'}), (b:Badge {id: 'badge-004'})
CREATE (a)-[r:UNLOCKS {
    badgeId: 'badge-004',
    createdAt: datetime('2024-01-01T00:00:00Z')
}]->(b);

// =============================================================================
// SECTION 19: RELATIONSHIPS - USER-PROGRESS-ACHIEVEMENT
// =============================================================================

MATCH (u:User {id: 'user-001'}), (a:Achievement {id: 'achievement-001'})
CREATE (u)-[r:PROGRESS {
    currentValue: 2,
    startedAt: datetime('2024-06-16T10:30:00Z'),
    lastUpdated: datetime('2026-03-21T19:05:00Z'),
    createdAt: datetime('2024-06-16T10:30:00Z')
}]->(a);

MATCH (u:User {id: 'user-002'}), (a:Achievement {id: 'achievement-001'})
CREATE (u)-[r:PROGRESS {
    currentValue: 52,
    startedAt: datetime('2024-03-12T08:00:00Z'),
    lastUpdated: datetime('2026-03-24T00:00:00Z'),
    createdAt: datetime('2024-03-12T08:00:00Z')
}]->(a);

MATCH (u:User {id: 'user-002'}), (a:Achievement {id: 'achievement-002'})
CREATE (u)-[r:PROGRESS {
    currentValue: 3,
    startedAt: datetime('2025-06-01T00:00:00Z'),
    lastUpdated: datetime('2026-02-15T16:00:00Z'),
    createdAt: datetime('2025-06-01T00:00:00Z')
}]->(a);

MATCH (u:User {id: 'user-003'}), (a:Achievement {id: 'achievement-001'})
CREATE (u)-[r:PROGRESS {
    currentValue: 1,
    startedAt: datetime('2025-01-20T12:00:00Z'),
    lastUpdated: datetime('2026-03-15T15:10:00Z'),
    createdAt: datetime('2025-01-20T12:00:00Z')
}]->(a);

MATCH (u:User {id: 'user-004'}), (a:Achievement {id: 'achievement-005'})
CREATE (u)-[r:PROGRESS {
    currentValue: 5200,
    startedAt: datetime('2024-02-01T09:00:00Z'),
    lastUpdated: datetime('2026-03-24T08:00:00Z'),
    createdAt: datetime('2024-02-01T09:00:00Z')
}]->(a);

// =============================================================================
// SECTION 20: RELATIONSHIPS - USER-COMPLETED-ACHIEVEMENT
// =============================================================================

MATCH (u:User {id: 'user-002'}), (a:Achievement {id: 'achievement-001'})
CREATE (u)-[r:COMPLETED {
    completedAt: datetime('2025-12-01T12:00:00Z'),
    pointsAwarded: 100,
    createdAt: datetime('2025-12-01T12:00:00Z')
}]->(a);

MATCH (u:User {id: 'user-002'}), (a:Achievement {id: 'achievement-002'})
CREATE (u)-[r:COMPLETED {
    completedAt: datetime('2026-02-15T16:00:00Z'),
    pointsAwarded: 250,
    createdAt: datetime('2026-02-15T16:00:00Z')
}]->(a);

MATCH (u:User {id: 'user-004'}), (a:Achievement {id: 'achievement-005'})
CREATE (u)-[r:COMPLETED {
    completedAt: datetime('2025-11-30T23:59:00Z'),
    pointsAwarded: 2000,
    createdAt: datetime('2025-11-30T23:59:00Z')
}]->(a);

// =============================================================================
// SECTION 21: SAMPLE QUERIES FOR TESTING
// =============================================================================

// Query 1: Get user's gamification dashboard
// MATCH (u:User {id: 'user-002'})
// OPTIONAL MATCH (u)-[r:EARNED]->(b:Badge)
// OPTIONAL MATCH (u)-[:SUPPORTS]->(t:Team)
// OPTIONAL MATCH (u)-[:WATCHED]->(m:Match)
// WHERE m.scheduledAt > datetime() - duration({days: 30})
// RETURN {
//     user: u,
//     badges: collect(DISTINCT b),
//     teams: collect(DISTINCT t),
//     watchedMatches: count(m),
//     totalPoints: u.points,
//     level: u.level
// } AS dashboard;

// Query 2: Find top supporters for a team
// MATCH (u:User)-[r:SUPPORTS]->(t:Team {id: 'team-001'})
// MATCH (u)-[:WATCHED]->(m:Match)
// WHERE (m)-[:HOME_TEAM|AWAY_TEAM]->(t)
// WITH u, count(m) AS matchWatchCount
// ORDER BY matchWatchCount DESC
// LIMIT 10
// RETURN u.id, u.username, matchWatchCount;

// Query 3: Get achievement progress for user
// MATCH (u:User {id: 'user-002'})
// MATCH (a:Achievement)
// OPTIONAL MATCH (u)-[r:COMPLETED]->(a)
// OPTIONAL MATCH (u)-[p:PROGRESS]->(a)
// RETURN a.name, a.type, a.targetValue,
//        coalesce(p.currentValue, 0) AS progress,
//        CASE WHEN r IS NOT NULL THEN true ELSE false END AS completed;

// Query 4: Find users who predicted match correctly
// MATCH (u:User)-[r:PARTICIPATED]->(m:Match {id: 'match-002'})
// WHERE r.isCorrect = true
// RETURN u.id, u.username, r.pointsEarned
// ORDER BY r.pointsEarned DESC;

// Query 5: Get team player roster
// MATCH (p:Player)-[r:PLAYS_FOR]->(t:Team {id: 'team-001'})
// WHERE r.isActive = true
// RETURN p.id, p.name, p.position, r.jerseyNumber
// ORDER BY r.jerseyNumber;

// Query 6: Find trending matches (most watchers)
// MATCH (u:User)-[r:WATCHED]->(m:Match)
// WHERE m.status = 'live'
// WITH m, count(DISTINCT u) AS viewerCount
// ORDER BY viewerCount DESC
// LIMIT 10
// RETURN m.id, m.homeTeamId, m.awayTeamId, viewerCount;

// =============================================================================
// END OF MIGRATION SCRIPT
// =============================================================================
