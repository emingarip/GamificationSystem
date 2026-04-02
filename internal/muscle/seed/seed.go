package seed

import (
	"context"
	"fmt"
	"time"

	"gamification/config"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Seeder handles the seeding of Neo4j database
type Seeder struct {
	driver  neo4j.Driver
	verbose bool
}

// NewSeeder creates a new seed instance
func NewSeeder(cfg *config.Neo4jConfig, verbose bool) (*Seeder, error) {
	driver, err := neo4j.NewDriver(cfg.URI, neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
		func(c *neo4j.Config) {
			c.MaxConnectionPoolSize = 10
			c.MaxConnectionLifetime = 30 * time.Minute
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	if err := driver.VerifyConnectivity(); err != nil {
		driver.Close()
		return nil, fmt.Errorf("failed to connect to neo4j: %w", err)
	}

	return &Seeder{
		driver:  driver,
		verbose: verbose,
	}, nil
}

// Close closes the Neo4j driver
func (s *Seeder) Close() error {
	return s.driver.Close()
}

// Seed populates the database with sample data
func (s *Seeder) Seed(ctx context.Context, clearExisting bool) error {
	data := GetSeedData()

	if clearExisting {
		s.print("Clearing existing data...")
		if err := s.clearAll(ctx); err != nil {
			return fmt.Errorf("failed to clear existing data: %w", err)
		}
		s.print("Existing data cleared.")
	}

	s.print("Creating teams...")
	if err := s.createTeams(ctx, data.Teams); err != nil {
		return fmt.Errorf("failed to create teams: %w", err)
	}
	s.print(fmt.Sprintf("Created %d teams", len(data.Teams)))

	s.print("Creating players...")
	if err := s.createPlayers(ctx, data.Players); err != nil {
		return fmt.Errorf("failed to create players: %w", err)
	}
	s.print(fmt.Sprintf("Created %d players", len(data.Players)))

	s.print("Creating users...")
	if err := s.createUsers(ctx, data.Users); err != nil {
		return fmt.Errorf("failed to create users: %w", err)
	}
	s.print(fmt.Sprintf("Created %d users", len(data.Users)))

	s.print("Creating badges...")
	if err := s.createBadges(ctx, data.Badges); err != nil {
		return fmt.Errorf("failed to create badges: %w", err)
	}
	s.print(fmt.Sprintf("Created %d badges", len(data.Badges)))

	s.print("Creating matches...")
	if err := s.createMatches(ctx, data.Matches); err != nil {
		return fmt.Errorf("failed to create matches: %w", err)
	}
	s.print(fmt.Sprintf("Created %d matches", len(data.Matches)))

	s.print("Creating relationships...")
	if err := s.createRelationships(ctx, data); err != nil {
		return fmt.Errorf("failed to create relationships: %w", err)
	}
	s.print("Relationships created successfully")

	s.print("Seed data population completed!")
	return nil
}

// clearAll removes all nodes and relationships from the database
func (s *Seeder) clearAll(ctx context.Context) error {
	session := s.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	_, err := session.Run("MATCH (n) DETACH DELETE n", nil)
	return err
}

// createTeams creates Team nodes
func (s *Seeder) createTeams(ctx context.Context, teams []Team) error {
	session := s.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	cypher := `
	CREATE (t:Team {
		teamId: $teamId,
		name: $name,
		city: $city,
		stadium: $stadium,
		founded: $founded,
		description: $description,
		createdAt: datetime()
	})
	`

	for _, team := range teams {
		_, err := session.Run(cypher, map[string]any{
			"teamId":      team.ID,
			"name":        team.Name,
			"city":        team.City,
			"stadium":     team.Stadium,
			"founded":     team.Founded,
			"description": team.Description,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// createPlayers creates Player nodes and relationships to teams
func (s *Seeder) createPlayers(ctx context.Context, players []Player) error {
	session := s.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	cypher := `
	CREATE (p:Player {
		playerId: $playerId,
		name: $name,
		number: $number,
		position: $position,
		nationality: $nationality,
		age: $age,
		createdAt: datetime()
	})
	WITH p
	MATCH (t:Team {teamId: $teamId})
	CREATE (p)-[r:PLAYS_FOR {since: datetime(), jerseyNumber: $number}]->(t)
	RETURN p.playerId, t.teamId
	`

	for _, player := range players {
		_, err := session.Run(cypher, map[string]any{
			"playerId":    player.ID,
			"name":        player.Name,
			"number":      player.Number,
			"position":    player.Position,
			"nationality": player.Nationality,
			"age":         player.Age,
			"teamId":      player.TeamID,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// createUsers creates User nodes
func (s *Seeder) createUsers(ctx context.Context, users []User) error {
	session := s.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	cypher := `
	CREATE (u:User {
		userId: $userId,
		username: $username,
		email: $email,
		age: $age,
		city: $city,
		createdAt: datetime(),
		points: 0,
		level: 1
	})
	`

	for _, user := range users {
		_, err := session.Run(cypher, map[string]any{
			"userId":   user.ID,
			"username": user.Username,
			"email":    user.Email,
			"age":      user.Age,
			"city":     user.City,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// createBadges creates Badge nodes
func (s *Seeder) createBadges(ctx context.Context, badges []Badge) error {
	session := s.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	cypher := `
	CREATE (b:Achievement {
		badgeId: $badgeId,
		name: $name,
		description: $description,
		category: $category,
		points: $points,
		icon: $icon,
		createdAt: datetime()
	})
	`

	for _, badge := range badges {
		_, err := session.Run(cypher, map[string]any{
			"badgeId":     badge.ID,
			"name":        badge.Name,
			"description": badge.Description,
			"category":    badge.Category,
			"points":      badge.Points,
			"icon":        badge.Icon,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// createMatches creates Match nodes and relationships to teams
func (s *Seeder) createMatches(ctx context.Context, matches []Match) error {
	session := s.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	cypher := `
	CREATE (m:Match {
		matchId: $matchId,
		date: $date,
		stadium: $stadium,
		homeScore: $homeScore,
		awayScore: $awayScore,
		completed: $completed,
		createdAt: datetime()
	})
	WITH m
	MATCH (ht:Team {teamId: $homeTeam})
	MATCH (at:Team {teamId: $awayTeam})
	CREATE (m)-[r1:HOME_TEAM]->(ht)
	CREATE (m)-[r2:AWAY_TEAM]->(at)
	RETURN m.matchId
	`

	for _, match := range matches {
		_, err := session.Run(cypher, map[string]any{
			"matchId":   match.ID,
			"homeTeam":  match.HomeTeam,
			"awayTeam":  match.AwayTeam,
			"homeScore": match.HomeScore,
			"awayScore": match.AwayScore,
			"date":      match.Date,
			"stadium":   match.Stadium,
			"completed": match.Completed,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// createRelationships creates user-team, user-match, and player-match relationships
func (s *Seeder) createRelationships(ctx context.Context, data SeedData) error {
	session := s.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	// User-Team supporter relationships
	// user_1 supports Galatasaray
	// user_2 supports Fenerbahçe
	// user_3 supports Trabzonspor
	// user_4 supports Galatasaray
	// user_5 supports Beşiktaş
	// user_6 supports multiple teams
	supporterCypher := `
	MATCH (u:User {userId: $userId})
	MATCH (t:Team {teamId: $teamId})
	CREATE (u)-[r:SUPPORTS {since: datetime(), isActive: true}]->(t)
	RETURN u.userId, t.teamId
	`

	supporterMappings := []struct {
		UserID string
		TeamID string
	}{
		{"user_1", "team_galatasaray"},
		{"user_2", "team_fenerbahce"},
		{"user_3", "team_trabzonspor"},
		{"user_4", "team_galatasaray"},
		{"user_5", "team_besiktas"},
		{"user_6", "team_galatasaray"},
		{"user_6", "team_fenerbahce"},
	}

	for _, m := range supporterMappings {
		_, err := session.Run(supporterCypher, map[string]any{
			"userId": m.UserID,
			"teamId": m.TeamID,
		})
		if err != nil {
			return err
		}
	}

	// User-Match watched relationships
	// Users who watched matches
	watchedCypher := `
	MATCH (u:User {userId: $userId})
	MATCH (m:Match {matchId: $matchId})
	CREATE (u)-[r:WATCHED {
		watchedAt: datetime(),
		duration: $duration,
		completion: $completion
	}]->(m)
	RETURN u.userId, m.matchId
	`

	watchedMappings := []struct {
		UserID     string
		MatchID    string
		Duration   int
		Completion float64
	}{
		{"user_1", "match_1", 95, 1.0},
		{"user_2", "match_1", 90, 0.95},
		{"user_4", "match_1", 95, 1.0},
		{"user_5", "match_1", 60, 0.63},
		{"user_3", "match_2", 92, 0.97},
		{"user_5", "match_2", 92, 0.97},
		{"user_6", "match_2", 45, 0.47},
	}

	for _, m := range watchedMappings {
		_, err := session.Run(watchedCypher, map[string]any{
			"userId":     m.UserID,
			"matchId":    m.MatchID,
			"duration":   m.Duration,
			"completion": m.Completion,
		})
		if err != nil {
			return err
		}
	}

	// Player-Team relationships are already created in createPlayers

	// Match-Team relationships are already created in createMatches

	// User-Badge achievement relationships
	// Grant some badges to users
	badgeCypher := `
	MATCH (u:User {userId: $userId})
	MATCH (b:Achievement {badgeId: $badgeId})
	CREATE (u)-[r:HAS_BADGE {
		progress: $progress,
		unlocked: $unlocked,
		earnedAt: datetime(),
		updatedAt: datetime()
	}]->(b)
	RETURN u.userId, b.badgeId
	`

	badgeMappings := []struct {
		UserID   string
		BadgeID  string
		Progress int
		Unlocked bool
	}{
		{"user_1", "badge_first_match", 100, true},
		{"user_2", "badge_first_match", 100, true},
		{"user_3", "badge_first_match", 100, true},
		{"user_4", "badge_first_match", 100, true},
		{"user_1", "badge_early_bird", 100, true},
		{"user_2", "badge_social_butterfly", 60, false},
		{"user_4", "badge_loyal_supporter", 45, false},
	}

	for _, m := range badgeMappings {
		_, err := session.Run(badgeCypher, map[string]any{
			"userId":   m.UserID,
			"badgeId":  m.BadgeID,
			"progress": m.Progress,
			"unlocked": m.Unlocked,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Seeder) print(msg string) {
	if s.verbose {
		fmt.Println("[SEED]", msg)
	}
}

// Run executes the seed process
func Run(cfg *config.Config, verbose, clearExisting bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	seeder, err := NewSeeder(&cfg.Neo4j, verbose)
	if err != nil {
		return fmt.Errorf("failed to create seeder: %w", err)
	}
	defer seeder.Close()

	fmt.Println("Starting Neo4j seed data population...")
	fmt.Println("========================================")

	if err := seeder.Seed(ctx, clearExisting); err != nil {
		return fmt.Errorf("seed failed: %w", err)
	}

	fmt.Println("========================================")
	fmt.Println("Seed completed successfully!")
	return nil
}
