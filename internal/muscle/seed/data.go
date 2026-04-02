package seed

// SeedData contains all sample data for populating Neo4j
type SeedData struct {
	Teams   []Team
	Players []Player
	Users   []User
	Matches []Match
	Badges  []Badge
}

// Team represents a football team
type Team struct {
	ID          string
	Name        string
	City        string
	Stadium     string
	Founded     int
	Description string
}

// Player represents a football player
type Player struct {
	ID          string
	Name        string
	Number      int
	Position    string
	Nationality string
	Age         int
	TeamID      string
}

// User represents a fan user
type User struct {
	ID       string
	Username string
	Email    string
	Age      int
	City     string
}

// Match represents a football match
type Match struct {
	ID        string
	HomeTeam  string
	AwayTeam  string
	HomeScore int
	AwayScore int
	Date      string
	Stadium   string
	Completed bool
}

// Badge represents an achievement badge
type Badge struct {
	ID          string
	Name        string
	Description string
	Category    string
	Points      int
	Icon        string
}

// GetSeedData returns all sample data for seeding
func GetSeedData() SeedData {
	return SeedData{
		Teams:   getTeams(),
		Players: getPlayers(),
		Users:   getUsers(),
		Matches: getMatches(),
		Badges:  getBadges(),
	}
}

func getTeams() []Team {
	return []Team{
		{
			ID:          "team_galatasaray",
			Name:        "Galatasaray",
			City:        "Istanbul",
			Stadium:     "RAMS Park",
			Founded:     1905,
			Description: "Galatasaray Sports Club, popularly known as Galatasaray, is a Turkish professional football club based in Istanbul.",
		},
		{
			ID:          "team_fenerbahce",
			Name:        "Fenerbahçe",
			City:        "Istanbul",
			Stadium:     "Ülker Stadyumu",
			Founded:     1907,
			Description: "Fenerbahçe S.K. is a Turkish professional football club based in Istanbul, founded in 1907.",
		},
		{
			ID:          "team_besiktas",
			Name:        "Beşiktaş",
			City:        "Istanbul",
			Stadium:     "Vodafone Park",
			Founded:     1903,
			Description: "Beşiktaş Jimnastik Kulübü is a Turkish sports club based in the Beşiktaş district of Istanbul.",
		},
		{
			ID:          "team_trabzonspor",
			Name:        "Trabzonspor",
			City:        "Trabzon",
			Stadium:     "Stadyum",
			Founded:     1967,
			Description: "Trabzonspor is a Turkish professional football club based in the city of Trabzon.",
		},
	}
}

func getPlayers() []Player {
	return []Player{
		// Galatasaray Players
		{
			ID:          "player_gs_1",
			Name:        "Icardi",
			Number:      9,
			Position:    "Forward",
			Nationality: "Argentina",
			Age:         31,
			TeamID:      "team_galatasaray",
		},
		{
			ID:          "player_gs_2",
			Name:        "Mert Günok",
			Number:      1,
			Position:    "Goalkeeper",
			Nationality: "Turkey",
			Age:         35,
			TeamID:      "team_galatasaray",
		},
		// Fenerbahçe Players
		{
			ID:          "player_fb_1",
			Name:        "Dzeko",
			Number:      9,
			Position:    "Forward",
			Nationality: "Bosnia",
			Age:         38,
			TeamID:      "team_fenerbahce",
		},
		{
			ID:          "player_fb_2",
			Name:        "Altay Bayındır",
			Number:      1,
			Position:    "Goalkeeper",
			Nationality: "Turkey",
			Age:         26,
			TeamID:      "team_fenerbahce",
		},
		// Beşiktaş Players
		{
			ID:          "player_bjk_1",
			Name:        "Gheorghe Gheorghe",
			Number:      10,
			Position:    "Midfielder",
			Nationality: "Romania",
			Age:         39,
			TeamID:      "team_besiktas",
		},
		{
			ID:          "player_bjk_2",
			Name:        "Ersin Destanoğlu",
			Number:      1,
			Position:    "Goalkeeper",
			Nationality: "Turkey",
			Age:         23,
			TeamID:      "team_besiktas",
		},
		// Trabzonspor Players
		{
			ID:          "player_ts_1",
			Name:        "Edin Višća",
			Number:      10,
			Position:    "Midfielder",
			Nationality: "Bosnia",
			Age:         34,
			TeamID:      "team_trabzonspor",
		},
		{
			ID:          "player_ts_2",
			Name:        "Uğurcan Çakır",
			Number:      1,
			Position:    "Goalkeeper",
			Nationality: "Turkey",
			Age:         28,
			TeamID:      "team_trabzonspor",
		},
	}
}

func getUsers() []User {
	return []User{
		{
			ID:       "user_1",
			Username: "cimen_ultra",
			Email:    "cimen@example.com",
			Age:      28,
			City:     "Istanbul",
		},
		{
			ID:       "user_2",
			Username: "sarı_kanaryalar",
			Email:    "kanarya@example.com",
			Age:      32,
			City:     "Ankara",
		},
		{
			ID:       "user_3",
			Username: "karadeniz_fırtınası",
			Email:    "firtina@example.com",
			Age:      25,
			City:     "Trabzon",
		},
		{
			ID:       "user_4",
			Username: "boğaziçi_Aslanları",
			Email:    "aslan@example.com",
			Age:      35,
			City:     "Istanbul",
		},
		{
			ID:       "user_5",
			Username: "futbol_sever_01",
			Email:    "futbol01@example.com",
			Age:      22,
			City:     "Izmir",
		},
		{
			ID:       "user_6",
			Username: "spor_abel",
			Email:    "sporabel@example.com",
			Age:      40,
			City:     "Bursa",
		},
	}
}

func getMatches() []Match {
	return []Match{
		{
			ID:        "match_1",
			HomeTeam:  "team_galatasaray",
			AwayTeam:  "team_fenerbahce",
			HomeScore: 3,
			AwayScore: 2,
			Date:      "2024-03-10T19:00:00Z",
			Stadium:   "RAMS Park",
			Completed: true,
		},
		{
			ID:        "match_2",
			HomeTeam:  "team_besiktas",
			AwayTeam:  "team_trabzonspor",
			HomeScore: 1,
			AwayScore: 1,
			Date:      "2024-03-17T19:00:00Z",
			Stadium:   "Vodafone Park",
			Completed: true,
		},
	}
}

func getBadges() []Badge {
	return []Badge{
		{
			ID:          "badge_first_match",
			Name:        "First Match Watched",
			Description: "Watch your first match live",
			Category:    "Milestone",
			Points:      10,
			Icon:        "stadium",
		},
		{
			ID:          "badge_loyal_supporter",
			Name:        "Loyal Supporter",
			Description: "Watch 10 matches of your team",
			Category:    "Engagement",
			Points:      50,
			Icon:        "heart",
		},
		{
			ID:          "badge_derby_master",
			Name:        "Derby Master",
			Description: "Watch 5 Istanbul derbies",
			Category:    "Special",
			Points:      100,
			Icon:        "trophy",
		},
		{
			ID:          "badge_early_bird",
			Name:        "Early Bird",
			Description: "Watch a match 1 hour before kickoff",
			Category:    "Activity",
			Points:      15,
			Icon:        "clock",
		},
		{
			ID:          "badge_social_butterfly",
			Name:        "Social Butterfly",
			Description: "Watch matches with 3 different friends",
			Category:    "Social",
			Points:      30,
			Icon:        "users",
		},
	}
}
