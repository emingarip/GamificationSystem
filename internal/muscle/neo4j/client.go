package neo4j

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gamification/config"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Client wraps the Neo4j connection
type Client struct {
	driver  neo4j.Driver
	config  *config.Neo4jConfig
	session neo4j.Session
}

// NewClient creates a new Neo4j client
func NewClient(cfg *config.Neo4jConfig) (*Client, error) {
	driver, err := neo4j.NewDriver(cfg.URI, neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
		func(c *neo4j.Config) {
			c.MaxConnectionPoolSize = cfg.MaxConnPool
			c.MaxConnectionLifetime = cfg.MaxConnLife
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	if err := driver.VerifyConnectivity(); err != nil {
		return nil, fmt.Errorf("failed to connect to neo4j: %w", err)
	}

	return &Client{
		driver: driver,
		config: cfg,
	}, nil
}

// Close closes the Neo4j connection
func (c *Client) Close() error {
	return c.driver.Close()
}

// Ping checks the Neo4j connection
func (c *Client) Ping(ctx context.Context) error {
	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	_, err := session.Run("RETURN 1", nil)
	return err
}

// QueryResult represents a Neo4j query result
type QueryResult struct {
	UserIDs []string
	Data    []map[string]any
}

// QueryAffectedUsers finds users based on rule conditions
// Uses relationship to teams in match, previous actions, and achievement progress
func (c *Client) QueryAffectedUsers(ctx context.Context, matchID, teamID, playerID string, queryPattern string, params map[string]string) (*QueryResult, error) {
	var cypher string
	var neo4jParams map[string]any

	switch queryPattern {
	case "team_supporters":
		cypher = TeamSupportersQuery
		neo4jParams = map[string]any{"matchId": matchID, "teamId": teamID}
	case "match_participants":
		cypher = MatchParticipantsQuery
		neo4jParams = map[string]any{"matchId": matchID}
	case "active_players":
		cypher = ActivePlayersInMatchQuery
		neo4jParams = map[string]any{"matchId": matchID, "teamId": teamID}
	case "player_followers":
		cypher = PlayerFollowersQuery
		neo4jParams = map[string]any{"playerId": playerID}
	case "achievement_progress":
		badgeID, _ := params["badge_id"]
		progress, _ := params["progress"]
		cypher = UsersWithAchievementProgressQuery
		neo4jParams = map[string]any{"badgeId": badgeID, "progress": progress}
	case "team_followers":
		cypher = TeamFollowersQuery
		neo4jParams = map[string]any{"teamId": teamID}
	default:
		return nil, fmt.Errorf("unknown query pattern: %s", queryPattern)
	}

	return c.runQuery(ctx, cypher, neo4jParams)
}

// runQuery executes a Cypher query
func (c *Client) runQuery(ctx context.Context, cypher string, params map[string]any) (*QueryResult, error) {
	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to run query: %w", err)
	}

	var userIDs []string
	var data []map[string]any

	for result.Next() {
		record := result.Record()
		if userID, ok := record.Get("userId"); ok {
			userIDs = append(userIDs, userID.(string))
		}
		// Collect all data
		data = append(data, record.AsMap())
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("error processing results: %w", err)
	}

	return &QueryResult{
		UserIDs: userIDs,
		Data:    data,
	}, nil
}

// GetUserMatchStats retrieves user statistics for a specific match.
func (c *Client) GetUserMatchStats(ctx context.Context, userID, matchID string) (map[string]any, error) {
	cypher := `
		MATCH (u:User {userId: $userId})-[r:PARTICIPATED_IN]->(m:Match {matchId: $matchId})
		RETURN r.stats as stats
	`
	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, map[string]any{"userId": userID, "matchId": matchID})
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	if result.Next() {
		record := result.Record()
		if stats, ok := record.Get("stats"); ok {
			if statsMap, ok := stats.(map[string]any); ok {
				return statsMap, nil
			}
		}
	}

	return nil, nil
}

// UpdateUserAchievementProgress updates a user's achievement progress
func (c *Client) UpdateUserAchievementProgress(ctx context.Context, userID, badgeID, matchID string, progress int) error {
	cypher := `
		MERGE (u:User {userId: $userId})
		WITH u
		MATCH (a:Achievement {badgeId: $badgeId})
		MERGE (u)-[r:HAS_BADGE]->(a)
		SET r.progress = $progress, r.lastMatchId = $matchId, r.updatedAt = datetime()
		RETURN r
	`
	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	_, err := session.Run(cypher, map[string]any{
		"userId":   userID,
		"badgeId":  badgeID,
		"matchId":  matchID,
		"progress": progress,
	})
	return err
}

// RecordUserAction records a user action in the knowledge graph
func (c *Client) RecordUserAction(ctx context.Context, userID, actionType, matchID, eventID string) error {
	cypher := `
		MERGE (u:User {userId: $userId})
		WITH u
		MATCH (m:Match {matchId: $matchId})
		MERGE (u)-[r:PERFORMED]->(m)
		SET r.actionType = $actionType, r.eventId = $eventId, r.timestamp = datetime()
		RETURN r
	`
	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	_, err := session.Run(cypher, map[string]any{
		"userId":     userID,
		"actionType": actionType,
		"matchId":    matchID,
		"eventId":    eventID,
	})
	return err
}

// GetMatchParticipants retrieves all participants in a match
func (c *Client) GetMatchParticipants(ctx context.Context, matchID string) (*QueryResult, error) {
	cypher := MatchParticipantsQuery
	return c.runQuery(ctx, cypher, map[string]any{"matchId": matchID})
}

// Cypher queries
const (
	// TeamSupportersQuery finds supporters of a team in a match context
	TeamSupportersQuery = `
		MATCH (u:User)-[r:SUPPORTS]->(t:Team {teamId: $teamId})
		WHERE r.isActive = true
		OPTIONAL MATCH (u)-[p:PARTICIPATED_IN]->(m:Match {matchId: $matchId})
		RETURN DISTINCT u.userId as userId, 
		       CASE WHEN p IS NOT NULL THEN true ELSE false END as inMatch
	`

	// MatchParticipantsQuery finds all users who participated in a match
	MatchParticipantsQuery = `
		MATCH (u:User)-[r:PARTICIPATED_IN]->(m:Match {matchId: $matchId})
		RETURN u.userId as userId, r.role as role, r.stats as stats
	`

	// ActivePlayersInMatchQuery finds active players from a team in a match
	ActivePlayersInMatchQuery = `
		MATCH (p:Player)-[r:PLAYED_IN]->(m:Match {matchId: $matchId})
		WHERE r.teamId = $teamId AND r.isStarter = true
		OPTIONAL MATCH (u:User)-[f:FOLLOWS]->(p)
		RETURN DISTINCT u.userId as userId
		LIMIT 100
	`

	// PlayerFollowersQuery finds followers of a player
	PlayerFollowersQuery = `
		MATCH (u:User)-[r:FOLLOWS]->(p:Player {playerId: $playerId})
		WHERE r.isActive = true
		RETURN u.userId as userId
	`

	// TeamFollowersQuery finds followers of a team
	TeamFollowersQuery = `
		MATCH (u:User)-[r:SUPPORTS]->(t:Team {teamId: $teamId})
		WHERE r.isActive = true
		RETURN u.userId as userId
	`

	// UsersWithAchievementProgressQuery finds users with progress toward a badge
	UsersWithAchievementProgressQuery = `
		MATCH (u:User)-[r:HAS_BADGE]->(a:Achievement {badgeId: $badgeId})
		WHERE r.progress >= $progress
		RETURN u.userId as userId, r.progress as progress
		ORDER BY r.progress DESC
	`
)

// MarshalJSON implements custom JSON marshaling
func (r *QueryResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"user_ids": r.UserIDs,
		"data":     r.Data,
	})
}

// User represents a user in the system
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Points    int       `json:"points"`
	Level     int       `json:"level"`
	CreatedAt time.Time `json:"created_at"`
}

// GetAllUsers retrieves all users with pagination
func (c *Client) GetAllUsers(ctx context.Context, limit, offset int) ([]User, error) {
	cypher := `
	MATCH (u:User)
	RETURN u.userId as id, COALESCE(u.name, u.username, u.email) as name, u.email as email, 
	       COALESCE(u.points, 0) as points, COALESCE(u.level, 1) as level,
	       COALESCE(u.createdAt, datetime()) as createdAt
	ORDER BY u.createdAt DESC
	SKIP $offset LIMIT $limit
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, map[string]any{"offset": offset, "limit": limit})
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	var users []User
	for result.Next() {
		record := result.Record()
		user := User{
			ID:        getString(record, "id"),
			Name:      getString(record, "name"),
			Email:     getString(record, "email"),
			Points:    getInt(record, "points"),
			Level:     getInt(record, "level"),
			CreatedAt: getTime(record, "createdAt"),
		}
		users = append(users, user)
	}

	return users, nil
}

// GetUserByID retrieves a single user by ID
func (c *Client) GetUserByID(ctx context.Context, userID string) (*User, error) {
	cypher := `
	MATCH (u:User {userId: $userId})
	RETURN u.userId as id, COALESCE(u.name, u.username, u.email) as name, u.email as email, 
	       COALESCE(u.points, 0) as points, COALESCE(u.level, 1) as level,
	       COALESCE(u.createdAt, datetime()) as createdAt
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, map[string]any{"userId": userID})
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if result.Next() {
		record := result.Record()
		user := &User{
			ID:        getString(record, "id"),
			Name:      getString(record, "name"),
			Email:     getString(record, "email"),
			Points:    getInt(record, "points"),
			Level:     getInt(record, "level"),
			CreatedAt: getTime(record, "createdAt"),
		}
		return user, nil
	}

	return nil, fmt.Errorf("user not found")
}

// UpdateUserPoints updates a user's points in Neo4j
func (c *Client) UpdateUserPoints(ctx context.Context, userID string, points int, operation string) error {
	var cypher string

	switch operation {
	case "add":
		cypher = `
		MERGE (u:User {userId: $userId})
		SET u.points = COALESCE(u.points, 0) + $points
		RETURN u.points
		`
	case "subtract":
		cypher = `
		MERGE (u:User {userId: $userId})
		SET u.points = COALESCE(u.points, 0) - $points
		RETURN u.points
		`
	case "set":
		fallthrough
	default:
		cypher = `
		MERGE (u:User {userId: $userId})
		SET u.points = $points
		RETURN u.points
		`
	}

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	_, err := session.Run(cypher, map[string]any{"userId": userID, "points": points})
	if err != nil {
		return fmt.Errorf("failed to update user points: %w", err)
	}

	return nil
}

// UpdateUserProfile updates editable user fields in Neo4j.
func (c *Client) UpdateUserProfile(ctx context.Context, userID, name, email string, level, points int) error {
	cypher := `
	MATCH (u:User {userId: $userId})
	SET u.name = $name,
	    u.email = $email,
	    u.level = $level,
	    u.points = $points,
	    u.updatedAt = datetime()
	RETURN u.userId
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	result, err := session.Run(cypher, map[string]any{
		"userId": userID,
		"name":   name,
		"email":  email,
		"level":  level,
		"points": points,
	})
	if err != nil {
		return fmt.Errorf("failed to update user profile: %w", err)
	}

	if !result.Next() {
		return fmt.Errorf("user not found")
	}

	return result.Err()
}

// DeleteUser removes a user and all attached relationships from Neo4j.
func (c *Client) DeleteUser(ctx context.Context, userID string) error {
	cypher := `
	MATCH (u:User {userId: $userId})
	DETACH DELETE u
	RETURN count(u) as deleted
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	result, err := session.Run(cypher, map[string]any{"userId": userID})
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.Next() {
		if getInt(result.Record(), "deleted") == 0 {
			return fmt.Errorf("user not found")
		}
	}

	return result.Err()
}

// GetUserStats retrieves user statistics from Neo4j
func (c *Client) GetUserStats(ctx context.Context, userID string) (map[string]any, error) {
	cypher := `
	MATCH (u:User {userId: $userId})
	OPTIONAL MATCH (u)-[r:HAS_BADGE]->(a:Achievement)
	OPTIONAL MATCH (u)-[p:PERFORMED]->(m:Match)
	RETURN COALESCE(u.points, 0) as points, 
	       COALESCE(u.level, 1) as level,
	       count(DISTINCT a) as badges,
	       count(DISTINCT m) as matches
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, map[string]any{"userId": userID})
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	if result.Next() {
		record := result.Record()
		stats := map[string]any{
			"points":  getInt(record, "points"),
			"level":   getInt(record, "level"),
			"badges":  getInt(record, "badges"),
			"matches": getInt(record, "matches"),
		}
		return stats, nil
	}

	return nil, fmt.Errorf("user not found")
}

// Helper functions for extracting values from Neo4j records
func getString(record *neo4j.Record, key string) string {
	if val, ok := record.Get(key); ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getInt(record *neo4j.Record, key string) int {
	if val, ok := record.Get(key); ok {
		switch v := val.(type) {
		case int64:
			return int(v)
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return 0
}

func getTime(record *neo4j.Record, key string) time.Time {
	if val, ok := record.Get(key); ok {
		if t, ok := val.(time.Time); ok {
			return t
		}
	}
	return time.Now()
}

// AwardPoints awards points to a user in Neo4j and updates their points property
func (c *Client) AwardPoints(ctx context.Context, userID string, points int, eventID, reason string) error {
	cypher := `
	MERGE (u:User {userId: $userId})
	SET u.points = COALESCE(u.points, 0) + $points,
	    u.updatedAt = datetime()
	RETURN u.points as totalPoints
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	result, err := session.Run(cypher, map[string]any{
		"userId":  userID,
		"points":  points,
		"eventId": eventID,
	})
	if err != nil {
		return fmt.Errorf("failed to award points: %w", err)
	}

	if !result.Next() {
		return fmt.Errorf("user not found: %s", userID)
	}

	return result.Err()
}

// Badge represents an achievement badge
type Badge struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Points      int       `json:"points"`
	Icon        string    `json:"icon"`
	Metric      string    `json:"metric"`
	Target      int       `json:"target"`
	CreatedAt   time.Time `json:"created_at"`
}

// GetAllBadges retrieves all badges from Neo4j
func (c *Client) GetAllBadges(ctx context.Context) ([]Badge, error) {
	cypher := `
	MATCH (b:Achievement)
	RETURN b.badgeId as id, b.name as name, COALESCE(b.description, '') as description,
	       COALESCE(b.category, 'common') as category, COALESCE(b.points, 0) as points,
	       COALESCE(b.icon, 'award') as icon, COALESCE(b.metric, '') as metric, COALESCE(b.target, 1) as target, COALESCE(b.createdAt, datetime()) as createdAt
	ORDER BY b.name
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get badges: %w", err)
	}

	var badges []Badge
	for result.Next() {
		record := result.Record()
		badge := Badge{
			ID:          getString(record, "id"),
			Name:        getString(record, "name"),
			Description: getString(record, "description"),
			Category:    getString(record, "category"),
			Points:      getInt(record, "points"),
			Icon:        getString(record, "icon"),
			Metric:      getString(record, "metric"),
			Target:      getInt(record, "target"),
			CreatedAt:   getTime(record, "createdAt"),
		}
		badges = append(badges, badge)
	}

	return badges, nil
}

// GetBadgeByID retrieves a single badge by ID
func (c *Client) GetBadgeByID(ctx context.Context, badgeID string) (*Badge, error) {
	cypher := `
	MATCH (b:Achievement {badgeId: $badgeId})
	RETURN b.badgeId as id, b.name as name, COALESCE(b.description, '') as description,
	       COALESCE(b.category, 'common') as category, COALESCE(b.points, 0) as points,
	       COALESCE(b.icon, 'award') as icon, COALESCE(b.metric, '') as metric, COALESCE(b.target, 1) as target, COALESCE(b.createdAt, datetime()) as createdAt
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, map[string]any{"badgeId": badgeID})
	if err != nil {
		return nil, fmt.Errorf("failed to get badge: %w", err)
	}

	if result.Next() {
		record := result.Record()
		return &Badge{
			ID:          getString(record, "id"),
			Name:        getString(record, "name"),
			Description: getString(record, "description"),
			Category:    getString(record, "category"),
			Points:      getInt(record, "points"),
			Icon:        getString(record, "icon"),
			Metric:      getString(record, "metric"),
			Target:      getInt(record, "target"),
			CreatedAt:   getTime(record, "createdAt"),
		}, nil
	}

	return nil, fmt.Errorf("badge not found")
}

// CreateBadge creates a new badge in Neo4j
func (c *Client) CreateBadge(ctx context.Context, badge *Badge) (string, error) {
	// Generate ID if not provided
	badgeID := badge.ID
	if badgeID == "" {
		badgeID = "badge_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	cypher := `
	MERGE (b:Achievement {badgeId: $badgeId})
	SET b.name = $name,
		b.description = $description,
		b.category = $category,
		b.points = $points,
		b.icon = $icon,
		b.metric = $metric,
		b.target = $target,
		b.createdAt = COALESCE(b.createdAt, datetime())
	RETURN b.badgeId
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	result, err := session.Run(cypher, map[string]any{
		"badgeId":     badgeID,
		"name":        badge.Name,
		"description": badge.Description,
		"category":    badge.Category,
		"points":      badge.Points,
		"icon":        badge.Icon,
		"metric":      badge.Metric,
		"target":      badge.Target,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create badge: %w", err)
	}

	if !result.Next() {
		return "", fmt.Errorf("failed to create badge")
	}

	return badgeID, result.Err()
}

// UpdateBadge updates an existing badge in Neo4j
func (c *Client) UpdateBadge(ctx context.Context, badgeID string, badge *Badge) error {
	cypher := `
	MATCH (b:Achievement {badgeId: $badgeId})
	SET b.name = $name,
	    b.description = $description,
	    b.category = $category,
	    b.points = $points,
	    b.icon = $icon,
	    b.updatedAt = datetime()
	RETURN b.badgeId
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	result, err := session.Run(cypher, map[string]any{
		"badgeId":     badgeID,
		"name":        badge.Name,
		"description": badge.Description,
		"category":    badge.Category,
		"points":      badge.Points,
		"icon":        badge.Icon,
	})
	if err != nil {
		return fmt.Errorf("failed to update badge: %w", err)
	}

	if !result.Next() {
		return fmt.Errorf("badge not found")
	}

	return result.Err()
}

// DeleteBadge deletes a badge from Neo4j
func (c *Client) DeleteBadge(ctx context.Context, badgeID string) error {
	cypher := `
	MATCH (b:Achievement {badgeId: $badgeId})
	DETACH DELETE b
	RETURN count(b) as deleted
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	result, err := session.Run(cypher, map[string]any{"badgeId": badgeID})
	if err != nil {
		return fmt.Errorf("failed to delete badge: %w", err)
	}

	if result.Next() {
		if getInt(result.Record(), "deleted") == 0 {
			return fmt.Errorf("badge not found")
		}
	}

	return result.Err()
}

// CheckBadgeExists checks if a badge with the given name or ID already exists
// If excludeBadgeID is provided, the badge with that ID will be excluded from the check
func (c *Client) CheckBadgeExists(ctx context.Context, badgeID, name, excludeBadgeID string) (bool, error) {
	var cypher string
	var params map[string]any

	if excludeBadgeID != "" {
		cypher = `
		MATCH (b:Achievement)
		WHERE b.badgeId <> $excludeBadgeId AND b.name = $name
		RETURN b
		LIMIT 1
		`
		params = map[string]any{
			"excludeBadgeId": excludeBadgeID,
			"name":           name,
		}
	} else {
		cypher = `
		MATCH (b:Achievement)
		WHERE b.badgeId = $badgeId OR b.name = $name
		RETURN b
		LIMIT 1
		`
		params = map[string]any{
			"badgeId": badgeID,
			"name":    name,
		}
	}

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, params)
	if err != nil {
		return false, fmt.Errorf("failed to check badge existence: %w", err)
	}

	return result.Next(), nil
}

// GrantBadge grants a badge to a user by creating a HAS_BADGE relationship
func (c *Client) GrantBadge(ctx context.Context, userID, badgeID, eventID, reason string) error {
	cypher := `
	MERGE (u:User {userId: $userId})
	WITH u
	MATCH (b:Achievement {badgeId: $badgeId})
	MERGE (u)-[r:HAS_BADGE]->(b)
	SET r.earnedAt = datetime(),
	    r.eventId = $eventId,
	    r.reason = $reason
	RETURN r
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	result, err := session.Run(cypher, map[string]any{
		"userId":  userID,
		"badgeId": badgeID,
		"eventId": eventID,
		"reason":  reason,
	})
	if err != nil {
		return fmt.Errorf("failed to grant badge: %w", err)
	}

	if !result.Next() {
		return fmt.Errorf("failed to create badge relationship")
	}

	return result.Err()
}

// CheckBadgeOwnership checks if a user already has a specific badge
func (c *Client) CheckBadgeOwnership(ctx context.Context, userID, badgeID string) (bool, error) {
	cypher := `
	MATCH (u:User {userId: $userId})-[r:HAS_BADGE]->(b:Achievement {badgeId: $badgeId})
	RETURN r
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, map[string]any{
		"userId":  userID,
		"badgeId": badgeID,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check badge ownership: %w", err)
	}

	// If there's a record, the user has the badge
	hasBadge := result.Next()
	return hasBadge, nil
}

// RecordRewardAction creates an append-only action record for reward history
func (c *Client) RecordRewardAction(ctx context.Context, userID, actionType string, points int, eventID, reason string) error {
	cypher := `
	MATCH (u:User {userId: $userId})
	CREATE (a:Action {
		actionType: $actionType,
		points: $points,
		eventId: $eventId,
		reason: $reason,
		timestamp: datetime()
	})
	CREATE (u)-[:PERFORMED]->(a)
	RETURN a
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	result, err := session.Run(cypher, map[string]any{
		"userId":     userID,
		"actionType": actionType,
		"points":     points,
		"eventId":    eventID,
		"reason":     reason,
	})
	if err != nil {
		return fmt.Errorf("failed to record reward action: %w", err)
	}

	_ = result.Next()
	return result.Err()
}

// ActionRecord represents a reward action record
type ActionRecord struct {
	UserID     string    `json:"user_id"`
	ActionType string    `json:"action_type"`
	Points     int       `json:"points"`
	Reason     string    `json:"reason"`
	Timestamp  time.Time `json:"timestamp"`
}

// GetRecentActions retrieves recent reward actions
func (c *Client) GetRecentActions(ctx context.Context, limit int) ([]ActionRecord, error) {
	cypher := `
	MATCH (u:User)-[:PERFORMED]->(a:Action)
	WHERE a.actionType IN ["award_points", "grant_badge"]
	RETURN u.userId as userId, a.actionType as actionType, COALESCE(a.points, 0) as points, 
	       COALESCE(a.reason, "") as reason, a.timestamp as timestamp
	ORDER BY a.timestamp DESC
	LIMIT $limit
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, map[string]any{"limit": limit})
	if err != nil {
		return nil, fmt.Errorf("failed to get recent actions: %w", err)
	}

	var actions []ActionRecord
	for result.Next() {
		record := result.Record()
		action := ActionRecord{
			UserID:     getString(record, "userId"),
			ActionType: getString(record, "actionType"),
			Points:     getInt(record, "points"),
			Reason:     getString(record, "reason"),
			Timestamp:  getTime(record, "timestamp"),
		}
		actions = append(actions, action)
	}

	return actions, nil
}

// GetPointsHistory retrieves points distributed over time with period filtering
func (c *Client) GetPointsHistory(ctx context.Context, period string) ([]map[string]any, error) {
	// Determine the date filter based on period
	var dateFilter string
	switch period {
	case "day":
		dateFilter = "-1 days"
	case "week":
		dateFilter = "-7 days"
	case "month":
		dateFilter = "-30 days"
	default:
		dateFilter = "-1 days" // Default to day
	}

	cypher := `
		MATCH (u:User)-[:PERFORMED]->(a:Action {actionType: "award_points"})
		WHERE a.timestamp >= datetime('now', '` + dateFilter + `') AND a.points > 0
		RETURN date(a.timestamp) as date, sum(a.points) as totalPoints
		ORDER BY date
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get points history: %w", err)
	}

	var history []map[string]any
	for result.Next() {
		record := result.Record()
		history = append(history, map[string]any{
			"date":   getString(record, "date"),
			"points": getInt(record, "totalPoints"),
		})
	}

	return history, nil
}

// GetTotalUsers returns total user count
func (c *Client) GetTotalUsers(ctx context.Context) (int, error) {
	cypher := `MATCH (u:User) RETURN count(u) as total`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get user count: %w", err)
	}

	if result.Next() {
		return getInt(result.Record(), "total"), nil
	}
	return 0, nil
}

// GetTotalBadges returns total earned badges count (badges users have earned)
func (c *Client) GetTotalBadges(ctx context.Context) (int, error) {
	// Count all HAS_BADGE relationships (each badge award), not distinct badge types
	cypher := `MATCH (u:User)-[r:HAS_BADGE]->(b:Achievement) RETURN count(r) as total`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get badge count: %w", err)
	}

	if result.Next() {
		return getInt(result.Record(), "total"), nil
	}
	return 0, nil
}

// GetBadgeCatalogCount returns total badge catalog count (unique badge types in the system)
func (c *Client) GetBadgeCatalogCount(ctx context.Context) (int, error) {
	cypher := `MATCH (b:Achievement) RETURN count(b) as total`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get badge catalog count: %w", err)
	}

	if result.Next() {
		return getInt(result.Record(), "total"), nil
	}
	return 0, nil
}

// GetActiveUsersCount returns count of users with recent activity (last 30 days)
func (c *Client) GetActiveUsersCount(ctx context.Context) (int, error) {
	cypher := `
		MATCH (u:User)-[:PERFORMED]->(a:Action)
		WHERE a.timestamp >= datetime() - duration('P30D')
		RETURN count(DISTINCT u) as total
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get active users count: %w", err)
	}

	if result.Next() {
		return getInt(result.Record(), "total"), nil
	}
	return 0, nil
}

// GetTotalPointsDistributed returns total points distributed
func (c *Client) GetTotalPointsDistributed(ctx context.Context) (int, error) {
	cypher := `MATCH (u:User)-[:PERFORMED]->(a:Action {actionType: "award_points"}) RETURN sum(a.points) as total`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get total points: %w", err)
	}

	if result.Next() {
		return getInt(result.Record(), "total"), nil
	}
	return 0, nil
}

// GetUserRecentActivity retrieves recent activity for a user
func (c *Client) GetUserRecentActivity(ctx context.Context, userID string, limit int) ([]ActionRecord, error) {
	cypher := `
	MATCH (u:User {userId: $userId})-[:PERFORMED]->(a:Action)
	WHERE a.actionType IN ["award_points", "grant_badge"]
	RETURN a.actionType as actionType, COALESCE(a.points, 0) as points, 
	       COALESCE(a.reason, "") as reason, a.timestamp as timestamp
	ORDER BY a.timestamp DESC
	LIMIT $limit
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, map[string]any{"userId": userID, "limit": limit})
	if err != nil {
		return nil, fmt.Errorf("failed to get user activity: %w", err)
	}

	var actions []ActionRecord
	for result.Next() {
		record := result.Record()
		action := ActionRecord{
			ActionType: getString(record, "actionType"),
			Points:     getInt(record, "points"),
			Reason:     getString(record, "reason"),
			Timestamp:  getTime(record, "timestamp"),
		}
		actions = append(actions, action)
	}

	return actions, nil
}

// UserBadge represents a user's earned badge with relationship properties
type UserBadge struct {
	UserID      string    `json:"user_id"`
	BadgeID     string    `json:"badge_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Metric      string    `json:"metric"`
	Target      int       `json:"target"`
	Icon        string    `json:"icon"`
	Points      int       `json:"points"`
	EarnedAt    time.Time `json:"earned_at"`
	Reason      string    `json:"reason"`
}

// GetUserBadges retrieves all badges for a user from Neo4j
func (c *Client) GetUserBadges(ctx context.Context, userID string) ([]UserBadge, error) {
	cypher := `
	MATCH (u:User {userId: $userId})-[r:HAS_BADGE]->(b:Achievement)
	RETURN b.badgeId as badgeId, COALESCE(b.name, "") as name,
	       COALESCE(b.description, "") as description,
	       COALESCE(b.category, "common") as category,
	       COALESCE(b.metric, "") as metric,
	       COALESCE(b.target, 1) as target,
	       COALESCE(b.icon, "award") as icon, COALESCE(b.points, 0) as points,
	       COALESCE(r.earnedAt, datetime()) as earnedAt, COALESCE(r.reason, "") as reason
	ORDER BY r.earnedAt DESC
	`

	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypher, map[string]any{"userId": userID})
	if err != nil {
		return nil, fmt.Errorf("failed to get user badges: %w", err)
	}

	var badges []UserBadge
	for result.Next() {
		record := result.Record()
		badge := UserBadge{
			BadgeID:     getString(record, "badgeId"),
			Name:        getString(record, "name"),
			Description: getString(record, "description"),
			Category:    getString(record, "category"),
			Metric:      getString(record, "metric"),
			Target:      getInt(record, "target"),
			Icon:        getString(record, "icon"),
			Points:      getInt(record, "points"),
			EarnedAt:    getTime(record, "earnedAt"),
			Reason:      getString(record, "reason"),
		}
		badges = append(badges, badge)
	}

	return badges, nil
}

// BadgeDistribution represents the count of users who have earned a specific badge
type BadgeDistribution struct {
	BadgeID   string `json:"badge_id"`
	BadgeName string `json:"badge_name"`
	UserCount int    `json:"user_count"`
}

// GetBadgeDistribution returns the distribution of badges among users
func (c *Client) GetBadgeDistribution(ctx context.Context) ([]BadgeDistribution, error) {
	session := c.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	query := `
		MATCH (b:Achievement)
		OPTIONAL MATCH (u:User)-[r:HAS_BADGE]->(b)
		RETURN b.badgeId AS badgeId, b.name AS badgeName, count(r) AS userCount
		ORDER BY userCount DESC, badgeName ASC
	`

	result, err := session.Run(query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get badge distribution: %w", err)
	}

	var dist []BadgeDistribution
	for result.Next() {
		record := result.Record()
		dist = append(dist, BadgeDistribution{
			BadgeID:   getString(record, "badgeId"),
			BadgeName: getString(record, "badgeName"),
			UserCount: getInt(record, "userCount"),
		})
	}

	return dist, nil
}
