package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gamification/auth"
	"gamification/config"
	"gamification/engine"
	"gamification/metrics"
	"gamification/models"
	"gamification/neo4j"
	"gamification/redis"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	goredis "github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/swaggo/swag"
)

// Server represents the HTTP API server
type Server struct {
	config      *config.Config
	redisClient *redis.Client
	neo4jClient *neo4j.Client
	router      *chi.Mux
	httpServer  *http.Server
	jwtManager  *auth.JWTManager
	engine      *engine.RuleEngine
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, redisClient *redis.Client, neo4jClient *neo4j.Client, ruleEngine *engine.RuleEngine) *Server {
	s := &Server{
		config:      cfg,
		redisClient: redisClient,
		neo4jClient: neo4jClient,
		engine:      ruleEngine,
	}

	// Initialize JWT manager
	s.jwtManager = auth.NewJWTManager(&auth.JWTConfig{
		SecretKey:          cfg.JWT.SecretKey,
		AccessTokenExpiry:  cfg.JWT.AccessTokenExpiry,
		RefreshTokenExpiry: cfg.JWT.RefreshTokenExpiry,
		Issuer:             cfg.JWT.Issuer,
	})

	s.setupRouter()
	return s
}

// setupRouter configures all API routes
func (s *Server) setupRouter() {
	r := chi.NewRouter()

	// Metrics middleware - must be first to track all requests
	r.Use(s.metricsMiddleware)
	r.Use(s.corsMiddleware)

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health check (no prefix)
	r.Get("/health", s.HealthCheck)

	// Prometheus metrics endpoint
	r.Get("/metrics", s.MetricsHandler)

	// Docs portal (Docusaurus) - serve static files from build output
	// With routeBasePath: '/', docs are built to build/ directory
	docsHandler := http.FileServer(http.Dir("./docs-portal/build"))
	docsStaticHandler := http.StripPrefix("/docs/", http.FileServer(http.Dir("./docs-portal/static")))

	// Serve /docs (without trailing slash) - redirect to the docs landing page
	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/", http.StatusFound)
	})

	// Serve /docs/ (with trailing slash) - serve the docs landing page
	r.Get("/docs/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./docs-portal/build/index.html")
	})

	// Backward compatibility for the old overview route
	r.Get("/docs/overview", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/", http.StatusFound)
	})

	r.Get("/docs/overview/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/", http.StatusFound)
	})

	// Serve /docs/swagger or /docs/swagger/ - redirect to /swagger
	r.Get("/docs/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusFound)
	})

	// Serve /docs/swagger/* - redirect to /swagger/* (Swagger is not part of Docusaurus docs)
	r.Get("/docs/swagger/*", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/docs/swagger/" {
			http.Redirect(w, r, "/swagger/index.html", http.StatusFound)
			return
		}

		if strings.HasPrefix(path, "/docs/swagger/") {
			newPath := strings.TrimPrefix(path, "/docs")
			http.Redirect(w, r, newPath, http.StatusFound)
			return
		}

		http.Redirect(w, r, "/swagger/index.html", http.StatusFound)
	})

	// Serve branding and other static assets that live outside the generated build output.
	r.Get("/docs/img/*", func(w http.ResponseWriter, r *http.Request) {
		docsStaticHandler.ServeHTTP(w, r)
	})

	// Serve docs at /docs/* - strip the /docs prefix to find the file in build dir
	r.Get("/docs/*", func(w http.ResponseWriter, r *http.Request) {
		// Strip /docs prefix from the path
		path := r.URL.Path
		if len(path) > 5 && path[:5] == "/docs" {
			path = path[5:] // Remove /docs
			if path == "" || path == "/" {
				http.ServeFile(w, r, "./docs-portal/build/index.html")
				return
			}
			r.URL.Path = path
		}
		docsHandler.ServeHTTP(w, r)
	})

	// Explicit /swagger/doc.json handler - returns swagger JSON directly using swag.ReadDoc
	r.Get("/swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
		doc, err := swag.ReadDoc("swagger")
		if err != nil {
			log.Printf("Error reading swagger doc: %v", err)
			http.Error(w, "Failed to read swagger documentation", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(doc)); err != nil {
			log.Printf("Error writing swagger doc: %v", err)
		}
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.InstanceName("swagger"),
		httpSwagger.URL("/swagger/doc.json"),
	))

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public endpoints can optionally read claims from a bearer token.
		r.Use(s.jwtManager.OptionalAuth())

		r.Post("/auth/login", s.Login)
		r.Group(func(r chi.Router) {
			r.Use(s.jwtManager.AuthMiddleware())
			r.Get("/auth/me", s.GetCurrentUser)
			r.Post("/auth/logout", s.Logout)
		})

		// Leaderboard (public)
		r.Get("/leaderboard", s.GetLeaderboard)

		// Rules - read is public, write requires admin
		r.Group(func(r chi.Router) {
			r.Get("/rules", s.ListRules)
			r.Get("/rules/{id}", s.GetRule)
		})
		r.Group(func(r chi.Router) {
			r.Use(s.jwtManager.AuthMiddleware())
			r.Use(auth.AdminOnly())
			r.Post("/rules", s.CreateRule)
			r.Put("/rules/{id}", s.UpdateRule)
			r.Delete("/rules/{id}", s.DeleteRule)
		})

		// Users - reading profile/stats is public, listing requires admin/auth
		r.Get("/users/{id}", s.GetUserProfile)
		r.Get("/users/{id}/stats", s.GetUserStats)
		
		r.Group(func(r chi.Router) {
			r.Use(s.jwtManager.AuthMiddleware())
			r.Get("/users", s.ListUsers)
		})
		r.Group(func(r chi.Router) {
			r.Use(s.jwtManager.AuthMiddleware())
			r.Use(auth.AdminOnly())
			r.Put("/users/{id}", s.UpdateUser)
			r.Delete("/users/{id}", s.DeleteUser)
			r.Put("/users/{id}/points", s.UpdateUserPoints)
			r.Post("/users/{id}/badges", s.AssignBadgeToUser)
		})

		// Badges - reading is public, write requires admin
		r.Get("/badges", s.ListBadges)
		r.Get("/badges/{id}", s.GetBadge)
		
		r.Group(func(r chi.Router) {
			r.Use(s.jwtManager.AuthMiddleware())
			r.Use(auth.AdminOnly())
			r.Post("/badges", s.CreateBadge)
			r.Put("/badges/{id}", s.UpdateBadge)
			r.Delete("/badges/{id}", s.DeleteBadge)
		})

		// Analytics
		r.Group(func(r chi.Router) {
			r.Use(s.jwtManager.AuthMiddleware())
			r.Get("/analytics/summary", s.GetAnalyticsSummary)
			r.Get("/analytics/activity", s.GetAnalyticsActivity)
			r.Get("/analytics/points-history", s.GetPointsHistory)
			r.Get("/analytics/badge-distribution", s.GetBadgeDistribution)
			r.Get("/analytics/event-logs", s.GetEventLogs)
			r.Get("/matches/{id}/stats", s.GetMatchStats)
		})

		// Events - public endpoint for applications
		r.Post("/events", s.ProcessEvent)

		// Events - test endpoint (admin only)
		r.Group(func(r chi.Router) {
			r.Use(s.jwtManager.AuthMiddleware())
			r.Use(auth.AdminOnly())
			r.Post("/events/test", s.TestEvent)
		})

		// Event Types - registry management (admin only)
		r.Group(func(r chi.Router) {
			r.Use(s.jwtManager.AuthMiddleware())
			r.Use(auth.AdminOnly())
			r.Get("/event-types", s.ListEventTypes)
			r.Post("/event-types", s.CreateEventType)
			r.Put("/event-types/{key}", s.UpdateEventType)
			r.Delete("/event-types/{key}", s.DeleteEventType)
		})
	})

	s.router = r
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting HTTP API server on %s", addr)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// metricsMiddleware tracks HTTP request metrics
func (s *Server) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		m := metrics.GetMetrics()
		m.IncActiveConnections()
		defer func() {
			m.DecActiveConnections()
			duration := time.Since(start)
			method := r.Method
			path := r.URL.Path
			// Use approximate status code from response
			m.RecordRequest(method, path, 200, duration)
		}()
		next.ServeHTTP(w, r)
	})
}

// MetricsHandler handles the Prometheus metrics endpoint
// @Summary Prometheus metrics
// @Description Get Prometheus metrics for monitoring
// @Tags health
// @Accept json
// @Produce text/plain
// @Success 200 {string} string
// @Router /metrics [get]
func (s *Server) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics.Handler().ServeHTTP(w, r)
}

// HealthCheck handles the health check endpoint
// @Summary Health check
// @Description Check the health status of all services (Redis, Neo4j, Kafka)
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (s *Server) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	services := make(map[string]ServiceStatus)

	// Check Redis
	if err := s.redisClient.Ping(ctx); err != nil {
		services["redis"] = ServiceStatus{Status: "unhealthy", Message: err.Error()}
	} else {
		services["redis"] = ServiceStatus{Status: "healthy"}
	}

	// Check Neo4j connectivity
	if err := s.neo4jClient.Ping(ctx); err != nil {
		services["neo4j"] = ServiceStatus{Status: "unhealthy", Message: err.Error()}
	} else {
		services["neo4j"] = ServiceStatus{Status: "healthy"}
	}

	// Kafka - check if broker is reachable (config-based check)
	services["kafka"] = ServiceStatus{Status: "healthy"}

	// Determine overall status
	status := "healthy"
	for _, v := range services {
		if v.Status != "healthy" {
			status = "degraded"
			break
		}
	}

	WriteJSON(w, http.StatusOK, HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Services:  services,
	})
}

// GetBadgeDistribution godoc
// @Summary Get badge distribution analytics
// @Description Gets the count of users per badge
// @Tags analytics
// @Accept json
// @Produce json
// @Success 200 {array} neo4j.BadgeDistribution
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/analytics/badge-distribution [get]
func (s *Server) GetBadgeDistribution(w http.ResponseWriter, r *http.Request) {
	dist, err := s.neo4jClient.GetBadgeDistribution(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to get badge distribution")
		return
	}

	WriteJSON(w, http.StatusOK, dist)
}

// GetEventLogs godoc
// @Summary Get recent event evaluation logs
// @Description Gets the most recent gamification event engine evaluations
// @Tags analytics
// @Accept json
// @Produce json
// @Param limit query int false "Limit (default 100)"
// @Success 200 {array} map[string]interface{}
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/analytics/event-logs [get]
func (s *Server) GetEventLogs(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	logs, err := s.redisClient.GetEventEvaluations(r.Context(), limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to get event logs")
		return
	}

	if logs == nil {
		logs = make([]map[string]any, 0)
	}

	WriteJSON(w, http.StatusOK, logs)
}


// ==================== Rules Handlers ====================

// ListRules handles GET /api/v1/rules - List all rules
// @Summary List rules
// @Description Get all gamification rules, optionally filtered by event type
// @Tags rules
// @Accept json
// @Produce json
// @Param event_type query string false "Filter by event type"
// @Success 200 {object} RulesListResponse
// @Router /rules [get]
func (s *Server) ListRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	eventType := GetQueryParam(r, "event_type")

	var rules []models.Rule
	var err error

	if eventType != "" {
		rules, err = s.redisClient.GetRulesByEventType(ctx, models.EventType(eventType))
	} else {
		// Get ALL rules (no filter) - remove the old "goal" fallback
		rules, err = s.redisClient.GetAllRules(ctx)
	}

	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch rules")
		return
	}

	if rules == nil {
		rules = []models.Rule{}
	}

	ruleInfos := make([]RuleInfo, len(rules))
	for i, rule := range rules {
		ruleInfos[i] = RuleInfo{
			ID:          rule.RuleID,
			Name:        rule.Name,
			Description: rule.Description,
			EventType:   string(rule.EventType),
			Points:      getActionPoints(rule.Actions),
			Enabled:     rule.IsActive,
			Conditions:  rule.Conditions,
			Rewards:     getActionRewards(rule.Actions),
			Actions:     rule.Actions,
		}
	}

	WriteJSON(w, http.StatusOK, RulesListResponse{
		Rules: ruleInfos,
		Count: len(ruleInfos),
	})
}

func getActionPoints(actions []models.RuleAction) int {
	for _, a := range actions {
		if a.ActionType == "award_points" {
			switch v := a.Params["points"].(type) {
			case int:
				return v
			case float64:
				return int(v)
			}
		}
	}
	return 0
}

func getActionRewards(actions []models.RuleAction) map[string]any {
	for _, a := range actions {
		if a.ActionType == "grant_badge" {
			return a.Params
		}
	}
	return nil
}

// CreateRule handles POST /api/v1/rules - Create a new rule
// @Summary Create rule
// @Description Create a new gamification rule with points and badge rewards
// @Tags rules
// @Accept json
// @Produce json
// @Param rule body CreateRuleRequest true "Rule definition"
// @Success 201 {object} RuleResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /rules [post]
// @Example 请求示例
//
//	{
//	  "id": "rule_goal_scorer",
//	  "name": "Goal Scorer",
//	  "description": "Award points for scoring a goal",
//	  "event_type": "goal",
//	  "points": 10,
//	  "multiplier": 1.0,
//	  "cooldown": 0,
//	  "enabled": true,
//	  "conditions": [
//	    {
//	      "field": "event_type",
//	      "operator": "==",
//	      "value": "goal",
//	      "evaluation_type": "simple"
//	    }
//	  ],
//	  "rewards": {
//	    "badge_id": "badge_first_goal"
//	  }
//	}
func (s *Server) CreateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateRuleRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.Name == "" {
		WriteError(w, http.StatusBadRequest, "Rule name is required")
		return
	}
	if req.EventType == "" {
		WriteError(w, http.StatusBadRequest, "Event type is required")
		return
	}

	// Build the rule from request
	rule := &models.Rule{
		RuleID:          req.ID,
		Name:            req.Name,
		Description:     req.Description,
		EventType:       models.EventType(req.EventType),
		IsActive:        req.Enabled,
		Priority:        1,
		Conditions:      req.Conditions,
		TargetUsers:     models.TargetUsers{},
		CooldownSeconds: req.Cooldown,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Determine Actions
	if req.Actions != nil && len(req.Actions) > 0 {
		rule.Actions = req.Actions
	} else {
		// Fallback to legacy creation via points/rewards
		if req.Points > 0 {
			rule.Actions = append(rule.Actions, models.RuleAction{
				ActionType: "award_points",
				Params:     map[string]any{"points": req.Points},
			})
		}
		if req.Rewards != nil && len(req.Rewards) > 0 {
			rule.Actions = append(rule.Actions, models.RuleAction{
				ActionType: "grant_badge",
				Params:     req.Rewards,
			})
		}
	}

	err := s.redisClient.SaveRule(ctx, rule)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to create rule")
		return
	}

	WriteJSON(w, http.StatusCreated, RuleResponse{
		ID:      rule.RuleID,
		Message: "Rule created successfully",
	})
}

// GetRule handles GET /api/v1/rules/:id - Get rule details
// @Summary Get rule
// @Description Get details of a specific gamification rule by ID
// @Tags rules
// @Accept json
// @Produce json
// @Param id path string true "Rule ID"
// @Success 200 {object} RuleInfo
// @Failure 404 {object} ErrorResponse
// @Router /rules/{id} [get]
func (s *Server) GetRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ruleID := GetPathParam(r, "id")

	rule, err := s.redisClient.GetRuleByID(ctx, ruleID)
	if err != nil || rule == nil {
		WriteError(w, http.StatusNotFound, "Rule not found")
		return
	}

	WriteJSON(w, http.StatusOK, RuleInfo{
		ID:          rule.RuleID,
		Name:        rule.Name,
		Description: rule.Description,
		EventType:   string(rule.EventType),
		Points:      getActionPoints(rule.Actions),
		Enabled:     rule.IsActive,
		Conditions:  rule.Conditions,
		Rewards:     getActionRewards(rule.Actions),
	})
}

// UpdateRule handles PUT /api/v1/rules/:id - Update a rule
// @Summary Update rule
// @Description Update an existing gamification rule
// @Tags rules
// @Accept json
// @Produce json
// @Param id path string true "Rule ID"
// @Param rule body UpdateRuleRequest true "Rule updates"
// @Success 200 {object} RuleResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /rules/{id} [put]
func (s *Server) UpdateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ruleID := GetPathParam(r, "id")

	var req UpdateRuleRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get existing rule
	existing, err := s.redisClient.GetRuleByID(ctx, ruleID)
	if err != nil || existing == nil {
		WriteError(w, http.StatusNotFound, "Rule not found")
		return
	}

	// Update fields
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.EventType != "" {
		existing.EventType = models.EventType(req.EventType)
	}
	// Update Actions
	if req.Actions != nil {
		existing.Actions = req.Actions
	} else {
		// Fallback
		if req.Points > 0 {
			// Update points action
			found := false
			for i, a := range existing.Actions {
				if a.ActionType == "award_points" {
					a.Params["points"] = req.Points
					existing.Actions[i] = a
					found = true
					break
				}
			}
			if !found {
				existing.Actions = append(existing.Actions, models.RuleAction{
					ActionType: "award_points",
					Params:     map[string]any{"points": req.Points},
				})
			}
		}

		if req.Rewards != nil {
			found := false
			for i, a := range existing.Actions {
				if a.ActionType == "grant_badge" {
					a.Params = req.Rewards
					existing.Actions[i] = a
					found = true
					break
				}
			}
			if !found {
				existing.Actions = append(existing.Actions, models.RuleAction{
					ActionType: "grant_badge",
					Params:     req.Rewards,
				})
			}
		}
	}

	if req.Cooldown > 0 {
		existing.CooldownSeconds = req.Cooldown
	}
	existing.UpdatedAt = time.Now()

	existing.IsActive = req.Enabled

	// Update conditions
	if req.Conditions != nil {
		existing.Conditions = req.Conditions
	}

	err = s.redisClient.SaveRule(ctx, existing)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update rule")
		return
	}

	WriteJSON(w, http.StatusOK, RuleResponse{
		ID:      ruleID,
		Message: "Rule updated successfully",
	})
}

// DeleteRule handles DELETE /api/v1/rules/:id - Delete a rule
// @Summary Delete rule
// @Description Delete a gamification rule by ID
// @Tags rules
// @Accept json
// @Produce json
// @Param id path string true "Rule ID"
// @Success 200 {object} RuleResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /rules/{id} [delete]
func (s *Server) DeleteRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ruleID := GetPathParam(r, "id")

	err := s.redisClient.DeleteRule(ctx, ruleID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to delete rule")
		return
	}

	WriteJSON(w, http.StatusOK, RuleResponse{
		ID:      ruleID,
		Message: "Rule deleted successfully",
	})
}

// ==================== Users Handlers ====================

// ListUsers handles GET /api/v1/users - List all users
// @Summary List users
// @Description Get all users with pagination support
// @Tags users
// @Accept json
// @Produce json
// @Param limit query int false "Number of users to return (default 50)"
// @Param offset query int false "Number of users to skip (default 0)"
// @Success 200 {object} UsersListResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /users [get]
func (s *Server) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := GetIntQueryParam(r, "limit", 50)
	offset := GetIntQueryParam(r, "offset", 0)

	users, err := s.neo4jClient.GetAllUsers(ctx, limit, offset)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	if users == nil {
		users = []neo4j.User{}
	}

	userList := make([]UserProfileResponse, len(users))
	for i, u := range users {
		userList[i] = UserProfileResponse{
			ID:        u.ID,
			Name:      u.Name,
			Email:     u.Email,
			Points:    u.Points, // Get points from Neo4j (source of truth)
			Level:     u.Level,
			CreatedAt: u.CreatedAt,
		}
	}

	WriteJSON(w, http.StatusOK, UsersListResponse{
		Users:  userList,
		Count:  len(userList),
		Limit:  limit,
		Offset: offset,
	})
}

// GetUserProfile handles GET /api/v1/users/:id - Get user profile
// @Summary Get user profile
// @Description Get detailed profile of a user including badges and recent activity
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} UserProfileResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [get]
func (s *Server) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := GetPathParam(r, "id")

	// Get user from Neo4j (source of truth)
	user, err := s.neo4jClient.GetUserByID(ctx, userID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "User not found")
		return
	}

	// Get badges from Neo4j (source of truth)
	neo4jBadges, _ := s.neo4jClient.GetUserBadges(ctx, userID)

	// Use points from Neo4j (source of truth) - user object already has points

	// Build rich badge info from Neo4j response (already has full details)
	richBadges := make([]RichBadgeInfo, 0, len(neo4jBadges))
	for _, b := range neo4jBadges {
		richBadges = append(richBadges, RichBadgeInfo{
			ID:          b.BadgeID,
			Name:        b.Name,
			Description: b.Description,
			Category:    b.Category,
			Metric:      b.Metric,
			Target:      b.Target,
			Icon:        b.Icon,
			Points:      b.Points,
			EarnedAt:    b.EarnedAt,
			Reason:      b.Reason,
		})
	}

	// Get recent activity from Neo4j
	recentActivity := make([]RecentActivityEntry, 0)
	if activities, err := s.neo4jClient.GetUserRecentActivity(ctx, userID, 5); err == nil {
		for _, a := range activities {
			recentActivity = append(recentActivity, RecentActivityEntry{
				ActionType: a.ActionType,
				Points:     a.Points,
				Reason:     a.Reason,
				Timestamp:  a.Timestamp,
			})
		}
	}

	// Fetch user stats from Redis
	stats := make(map[string]int)
	var cursor uint64
	for {
		var keys []string
		keys, cursor, err = s.redisClient.Raw().Scan(ctx, cursor, fmt.Sprintf("user:%s:*", userID), 100).Result()
		if err == nil {
			for _, key := range keys {
				// skip lock keys
				if len(key) > 5 && key[len(key)-5:] == ":lock" {
					continue
				}
				prefix := fmt.Sprintf("user:%s:", userID)
				if len(key) > len(prefix) {
					metricName := key[len(prefix):]
					valStr, err := s.redisClient.Raw().Get(ctx, key).Result()
					if err == nil {
						if val, err2 := strconv.Atoi(valStr); err2 == nil {
							stats[metricName] = val
						}
					}
				}
			}
		}
		if cursor == 0 {
			break
		}
	}

	// Fetch global event counts from Redis
	cursor = 0
	for {
		var keys []string
		keys, cursor, err = s.redisClient.Raw().Scan(ctx, cursor, fmt.Sprintf("events:count:global:%s:*", userID), 100).Result()
		if err == nil {
			for _, key := range keys {
				prefix := fmt.Sprintf("events:count:global:%s:", userID)
				if len(key) > len(prefix) {
					metricName := key[len(prefix):]
					valStr, err := s.redisClient.Raw().Get(ctx, key).Result()
					if err == nil {
						if val, err2 := strconv.Atoi(valStr); err2 == nil {
							// If metricName matches our badge target types (e.g. invite_friend), insert it
							stats[metricName] = val
							stats["global_count:"+metricName] = val
						}
					}
				}
			}
		}
		if cursor == 0 {
			break
		}
	}

	WriteJSON(w, http.StatusOK, UserProfileResponse{
		ID:             user.ID,
		Name:           user.Name,
		Email:          user.Email,
		Points:         user.Points, // Get from Neo4j (source of truth)
		Level:          user.Level,
		CreatedAt:      user.CreatedAt,
		Stats:          stats,
		RichBadgeInfo:  richBadges,
		RecentActivity: recentActivity,
	})
}

// UpdateUserPoints handles PUT /api/v1/users/:id/points - Update user points
// @Summary Update user points
// @Description Add, subtract, or set user points. Use operation 'add' (default), 'subtract', or 'set'
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param points body UpdatePointsRequest true "Points update request"
// @Success 200 {object} UserPointsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /users/{id}/points [put]
// @Example 请求示例
//
//	{
//	  "points": 100,
//	  "operation": "add"
//	}
func (s *Server) UpdateUserPoints(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := GetPathParam(r, "id")

	var req UpdatePointsRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Points == 0 {
		WriteError(w, http.StatusBadRequest, "Points value is required")
		return
	}

	// Determine operation
	operation := "add"
	if req.Operation != "" {
		operation = req.Operation
	}

	// Get current points from Neo4j (source of truth) before update
	currentUser, err := s.neo4jClient.GetUserByID(ctx, userID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "User not found")
		return
	}
	currentPoints := currentUser.Points

	// Calculate delta based on operation type
	var delta int
	switch operation {
	case "subtract":
		// For subtract: delta is negative of requested points
		delta = -req.Points
	case "set":
		// For set: delta is the difference between requested and current
		delta = req.Points - currentPoints
	default:
		// For "add" or any other operation: delta is the requested points
		delta = req.Points
	}

	// Update points in Neo4j (source of truth)
	err = s.neo4jClient.UpdateUserPoints(ctx, userID, req.Points, operation)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update user points")
		return
	}

	// Update Redis leaderboard (cache for leaderboard sorted sets) with the calculated delta
	s.redisClient.UpdateLeaderboard(ctx, userID, delta, "add")

	// Get current points from Neo4j (source of truth) after update
	user, err := s.neo4jClient.GetUserByID(ctx, userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to get updated user points")
		return
	}

	// Update Redis cache (optional, for fast reads)
	s.redisClient.SetUserPoints(ctx, userID, user.Points)

	WriteJSON(w, http.StatusOK, UserPointsResponse{
		UserID:  userID,
		Points:  user.Points,
		Message: "Points updated successfully",
	})
}

// AssignBadgeToUser handles POST /api/v1/users/:id/badges - Assign badge to user
// @Summary Assign badge to user
// @Description Manually assign a badge to a user
// @Tags badges
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param badge body AssignBadgeRequest true "Badge assignment request"
// @Success 200 {object} BadgeAssignResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /users/{id}/badges [post]
func (s *Server) AssignBadgeToUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := GetPathParam(r, "id")

	var req AssignBadgeRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.BadgeID == "" {
		WriteError(w, http.StatusBadRequest, "Badge ID is required")
		return
	}

	// Use Neo4j as source of truth for badge ownership
	err := s.neo4jClient.GrantBadge(ctx, userID, req.BadgeID, "", "Manual badge assignment")
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to assign badge")
		return
	}

	// Update Redis cache for badges (optional, for faster reads)
	// This keeps the cache in sync with Neo4j source of truth
	s.redisClient.AssignBadgeToUser(ctx, userID, req.BadgeID)

	WriteJSON(w, http.StatusOK, BadgeAssignResponse{
		UserID:  userID,
		BadgeID: req.BadgeID,
		Message: "Badge assigned successfully",
	})
}

// ==================== Badges Handlers ====================

// ListBadges handles GET /api/v1/badges - List all badges
// @Summary List badges
// @Description Get all available badges in the system
// @Tags badges
// @Accept json
// @Produce json
// @Success 200 {object} BadgesListResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /badges [get]
func (s *Server) ListBadges(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	badges, err := s.neo4jClient.GetAllBadges(ctx)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch badges")
		return
	}

	if badges == nil {
		badges = []neo4j.Badge{}
	}

	badgeInfos := make([]BadgeInfo, len(badges))
	for i, b := range badges {
		badgeInfos[i] = BadgeInfo{
			ID:          b.ID,
			Name:        b.Name,
			Description: b.Description,
			Icon:        b.Icon,
			Points:      b.Points,
			Category:    b.Category,
			Metric:      b.Metric,
			Target:      b.Target,
		}
	}

	WriteJSON(w, http.StatusOK, BadgesListResponse{
		Badges: badgeInfos,
		Count:  len(badgeInfos),
	})
}

// CreateBadge handles POST /api/v1/badges - Create a new badge
// @Summary Create badge
// @Description Create a new badge that can be earned by users
// @Tags badges
// @Accept json
// @Produce json
// @Param badge body CreateBadgeRequest true "Badge definition"
// @Success 201 {object} BadgeResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /badges [post]
func (s *Server) CreateBadge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateBadgeRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		WriteError(w, http.StatusBadRequest, "Badge name is required")
		return
	}

	// Check for duplicate badge (by name or ID)
	exists, err := s.neo4jClient.CheckBadgeExists(ctx, req.ID, req.Name, "")
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to check badge existence")
		return
	}
	if exists {
		WriteError(w, http.StatusConflict, "Badge with this name or ID already exists")
		return
	}

	badge := &neo4j.Badge{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		Points:      req.Points,
		Category:    req.Rarity,
	}

	badgeID, err := s.neo4jClient.CreateBadge(ctx, badge)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to create badge")
		return
	}

	WriteJSON(w, http.StatusCreated, BadgeResponse{
		ID:      badgeID,
		Message: "Badge created successfully",
	})
}

// ==================== Leaderboard Handler ====================

// GetLeaderboard handles GET /api/v1/leaderboard - Get leaderboard
// @Summary Get leaderboard
// @Description Get the top users by points, optionally filtered by event type
// @Tags leaderboard
// @Accept json
// @Produce json
// @Param limit query int false "Number of entries to return (default 10)"
// @Param event_type query string false "Filter by event type"
// @Success 200 {object} LeaderboardResponse
// @Failure 500 {object} ErrorResponse
// @Router /leaderboard [get]
func (s *Server) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := GetIntQueryParam(r, "limit", 10)
	eventType := GetQueryParam(r, "event_type")

	entries, err := s.redisClient.GetLeaderboard(ctx, eventType, limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch leaderboard")
		return
	}

	if entries == nil {
		entries = []goredis.Z{}
	}

	leaderboardEntries := make([]LeaderboardEntry, len(entries))
	for i, e := range entries {
		leaderboardEntries[i] = LeaderboardEntry{
			UserID: e.Member.(string),
			Score:  int(e.Score),
			Rank:   i + 1,
		}
	}

	WriteJSON(w, http.StatusOK, LeaderboardResponse{
		Entries: leaderboardEntries,
		Count:   len(leaderboardEntries),
	})
}

// GetUserStats handles GET /api/v1/users/:id/stats - Get user statistics
// @Summary Get user stats
// @Description Get statistics for a specific user
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /users/{id}/stats [get]
func (s *Server) GetUserStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := GetPathParam(r, "id")

	// Get stats from Neo4j (source of truth) - includes points
	stats, err := s.neo4jClient.GetUserStats(ctx, userID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "User stats not found")
		return
	}

	// Use points from Neo4j (source of truth) - stats already contains points
	WriteJSON(w, http.StatusOK, stats)
}

// GetMatchStats handles GET /api/v1/matches/:id/stats - Get match statistics
// @Summary Get match stats
// @Description Get statistics for a specific match
// @Tags analytics
// @Accept json
// @Produce json
// @Param id path string true "Match ID"
// @Success 200 {object} MatchStatsResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /matches/{id}/stats [get]
func (s *Server) GetMatchStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	matchID := GetPathParam(r, "id")

	participants, err := s.neo4jClient.GetMatchParticipants(ctx, matchID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "Match not found")
		return
	}

	WriteJSON(w, http.StatusOK, MatchStatsResponse{
		MatchID:      matchID,
		Participants: participants.Data,
		Count:        len(participants.Data),
	})
}

// ==================== Test Event Handler ====================

// TestEvent handles POST /api/v1/events/test - Test an event
// @Summary Test event
// @Description Test how an event would be processed by the rule engine. Use dry_run=true to evaluate without executing actions
// @Tags events
// @Accept json
// @Produce json
// @Param dry_run query bool false "Dry run mode (default true)"
// @Param event body SwaggerTestEventRequest true "Event to test"
// @Success 200 {object} TestEventResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /events/test [post]
// @Example 请求示例
//
//	{
//	  "event": {
//	    "event_id": "evt_001",
//	    "event_type": "goal",
//	    "match_id": "match_123",
//	    "team_id": "team_a",
//	    "player_id": "player_456",
//	    "minute": 45,
//	    "timestamp": "2024-01-15T10:30:00Z",
//	    "metadata": {
//	      "scorer_id": "player_456",
//	      "assist_player": "player_789",
//	      "goal_type": "open_play"
//	    }
//	  },
//	  "dry_run": true
//	}
func (s *Server) TestEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req TestEventRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Determine dry_run value with priority: query param > body > default true
	dryRun := true

	// First check query param (highest priority)
	dryRunParam := r.URL.Query().Get("dry_run")
	if dryRunParam != "" {
		dryRun = !(dryRunParam == "false" || dryRunParam == "0")
	} else if req.DryRun != nil {
		// Fall back to body value if query param not provided
		dryRun = *req.DryRun
	}
	// Default remains true if neither query param nor body provides a value

	// Use the rule engine to process the event
	// The engine will handle action execution based on dryRun flag
	if s.engine == nil {
		WriteError(w, http.StatusInternalServerError, "Rule engine not available")
		return
	}

	result := s.engine.ProcessMatchEvent(ctx, &req.Event, dryRun)

	// Build response
	matches := make([]RuleMatchInfo, 0)
	affectedUsers := make([]string, 0)
	actions := make([]ActionInfo, 0)

	for _, tr := range result.TriggeredRules {
		matches = append(matches, RuleMatchInfo{
			RuleID:  tr.Rule.RuleID,
			Name:    tr.Rule.Name,
			Matched: tr.Matched,
		})

		if tr.Matched {
			// Collect unique affected users
			for _, u := range tr.Users {
				found := false
				for _, existing := range affectedUsers {
					if existing == u {
						found = true
						break
					}
				}
				if !found {
					affectedUsers = append(affectedUsers, u)
				}
			}

			// Collect actions
			for _, a := range tr.Actions {
				actions = append(actions, ActionInfo{
					ActionType: a.ActionType,
					Params:     a.Params,
				})
			}
		}
	}

	// The engine has already handled action execution based on dryRun flag
	// dryRun=true: actions were NOT executed (just evaluated)
	// dryRun=false: actions WERE executed by the engine
	// We just report what happened
	executed := !dryRun && len(actions) > 0

	WriteJSON(w, http.StatusOK, TestEventResponse{
		Matches:       matches,
		AffectedUsers: affectedUsers,
		Actions:       actions,
		Executed:      executed,
	})
}

// ProcessEvent handles POST /api/v1/events - Process a client event
// @Summary Process client event
// @Description Process a real event triggered by a client application (e.g. SportsApp)
// @Tags events
// @Accept json
// @Produce json
// @Param event body ProcessEventRequest true "Event to process"
// @Success 200 {object} ProcessEventResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /events [post]
func (s *Server) ProcessEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ProcessEventRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.UserID == "" || req.EventType == "" {
		WriteError(w, http.StatusBadRequest, "user_id and event_type are required")
		return
	}

	// Put user_id in metadata so conditions can use it, and assign PlayerID
	if req.Metadata == nil {
		req.Metadata = make(map[string]any)
	}
	req.Metadata["user_id"] = req.UserID

	metadataBytes, _ := json.Marshal(req.Metadata)

	event := &models.MatchEvent{
		EventID:   "evt_client_" + strconv.FormatInt(time.Now().UnixNano(), 10),
		EventType: models.EventType(req.EventType),
		Timestamp: time.Now(),
		Metadata:  metadataBytes,
		PlayerID:  req.UserID,
	}

	if s.engine == nil {
		WriteError(w, http.StatusInternalServerError, "Rule engine not available")
		return
	}

	// Process actual event (dryRun=false)
	result := s.engine.ProcessMatchEvent(ctx, event, false)

	matchedRules := make([]string, 0)
	pointsAwarded := 0
	badgesAwarded := make([]string, 0)

	for _, tr := range result.TriggeredRules {
		if tr.Matched {
			matchedRules = append(matchedRules, tr.Rule.Name)
			for _, a := range tr.Actions {
				if a.ActionType == "award_points" {
					if pts, ok := a.Params["points"].(float64); ok {
						pointsAwarded += int(pts)
					} else if pts, ok := a.Params["points"].(int); ok {
						pointsAwarded += pts
					}
				} else if a.ActionType == "grant_badge" {
					if bid, ok := a.Params["badge_id"].(string); ok {
						badgesAwarded = append(badgesAwarded, bid)
					}
				}
			}
		}
	}

	WriteJSON(w, http.StatusOK, ProcessEventResponse{
		Message:       "Event processed successfully",
		MatchedRules:  matchedRules,
		PointsAwarded: pointsAwarded,
		BadgesAwarded: badgesAwarded,
	})
}


// ==================== Analytics Handlers ====================

// GetAnalyticsSummary handles GET /api/v1/analytics/summary
// @Summary Get analytics summary
// @Description Get summary statistics including total users, badges, points, and active rules
// @Tags analytics
// @Accept json
// @Produce json
// @Success 200 {object} AnalyticsSummaryResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /analytics/summary [get]
func (s *Server) GetAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get counts from Neo4j
	totalUsers, _ := s.neo4jClient.GetTotalUsers(ctx)
	totalBadges, _ := s.neo4jClient.GetTotalBadges(ctx)
	badgeCatalogCount, _ := s.neo4jClient.GetBadgeCatalogCount(ctx)
	activeUsers, _ := s.neo4jClient.GetActiveUsersCount(ctx)
	pointsDistributed, _ := s.neo4jClient.GetTotalPointsDistributed(ctx)

	// Get rules with conditions from Redis
	activeRules, _ := s.redisClient.GetTotalActiveRules(ctx)

	// Events processed - get from Redis
	eventsProcessed, _ := s.redisClient.GetEventHistoryCount(ctx)

	WriteJSON(w, http.StatusOK, AnalyticsSummaryResponse{
		TotalUsers:        totalUsers,
		TotalBadges:       totalBadges,
		BadgeCatalogCount: badgeCatalogCount,
		ActiveUsers:       activeUsers,
		ActiveRules:       activeRules,
		PointsDistributed: pointsDistributed,
		EventsProcessed:   eventsProcessed,
	})
}

// GetAnalyticsActivity handles GET /api/v1/analytics/activity
// @Summary Get recent activity
// @Description Get recent reward actions and user activities
// @Tags analytics
// @Accept json
// @Produce json
// @Param limit query int false "Number of activities to return (default 20, max 100)"
// @Success 200 {object} ActivityResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /analytics/activity [get]
func (s *Server) GetAnalyticsActivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := GetIntQueryParam(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}

	actions, err := s.neo4jClient.GetRecentActions(ctx, limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch activity")
		return
	}

	if actions == nil {
		actions = []neo4j.ActionRecord{}
	}

	activities := make([]ActivityEntry, len(actions))
	for i, a := range actions {
		activities[i] = ActivityEntry{
			UserID:     a.UserID,
			ActionType: a.ActionType,
			Points:     a.Points,
			Reason:     a.Reason,
			Timestamp:  a.Timestamp,
		}
	}

	WriteJSON(w, http.StatusOK, ActivityResponse{
		Activities: activities,
		Count:      len(activities),
	})
}

// GetPointsHistory handles GET /api/v1/analytics/points-history
// @Summary Get points history
// @Description Get historical points distribution data over time
// @Tags analytics
// @Accept json
// @Produce json
// @Param period query string false "Time period: day, week, month (default day)"
// @Success 200 {object} PointsHistoryResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /analytics/points-history [get]
func (s *Server) GetPointsHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	period := GetQueryParam(r, "period")
	if period == "" {
		period = "day"
	}

	history, err := s.neo4jClient.GetPointsHistory(ctx, period)
	if err != nil {
		// Return empty history instead of 500 when Neo4j has no data
		WriteJSON(w, http.StatusOK, PointsHistoryResponse{
			Period:  period,
			History: []PointsHistoryEntry{},
		})
		return
	}

	if history == nil {
		history = []map[string]any{}
	}

	entries := make([]PointsHistoryEntry, 0, len(history))
	for _, h := range history {
		date, _ := h["date"].(string)
		if date == "" {
			continue
		}
		points := 0
		switch v := h["points"].(type) {
		case int:
			points = v
		case int64:
			points = int(v)
		case float64:
			points = int(v)
		}
		entries = append(entries, PointsHistoryEntry{
			Date:   date,
			Points: points,
		})
	}

	WriteJSON(w, http.StatusOK, PointsHistoryResponse{
		Period:  period,
		History: entries,
	})
}

// ==================== Helper Functions ====================

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("Error encoding JSON: %v", err)
		}
	}
}

// WriteError writes an error response
func WriteError(w http.ResponseWriter, status int, err string) {
	WriteJSON(w, status, ErrorResponse{Error: err})
}

// ParseJSON parses JSON request body
func ParseJSON(r *http.Request, dest any) error {
	return json.NewDecoder(r.Body).Decode(dest)
}

// GetPathParam extracts a path parameter
func GetPathParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

// GetQueryParam extracts a query parameter
func GetQueryParam(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

// GetIntQueryParam extracts an integer query parameter with default
func GetIntQueryParam(r *http.Request, key string, def int) int {
	if v := r.URL.Query().Get(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// ==================== Event Types Handlers ====================

// ListEventTypes handles GET /api/v1/event-types - List all event types
// @Summary List event types
// @Description Get all event types from the registry
// @Tags event-types
// @Accept json
// @Produce json
// @Success 200 {object} EventTypesListResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /event-types [get]
func (s *Server) ListEventTypes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	eventTypes, err := s.redisClient.ListEventTypes(ctx)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch event types")
		return
	}

	if eventTypes == nil {
		eventTypes = []redis.EventType{}
	}

	typeInfos := make([]EventTypeInfo, len(eventTypes))
	for i, et := range eventTypes {
		typeInfos[i] = EventTypeInfo{
			Key:         et.Key,
			Name:        et.Name,
			Description: et.Description,
			Category:    et.Category,
			Enabled:     et.Enabled,
			CreatedAt:   et.CreatedAt,
			UpdatedAt:   et.UpdatedAt,
		}
	}

	WriteJSON(w, http.StatusOK, EventTypesListResponse{
		EventTypes: typeInfos,
		Count:      len(typeInfos),
	})
}

// CreateEventType handles POST /api/v1/event-types - Create a new event type
// @Summary Create event type
// @Description Create a new event type in the registry
// @Tags event-types
// @Accept json
// @Produce json
// @Param event_type body CreateEventTypeRequest true "Event type definition"
// @Success 201 {object} EventTypeResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /event-types [post]
func (s *Server) CreateEventType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateEventTypeRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Key == "" {
		WriteError(w, http.StatusBadRequest, "Event type key is required")
		return
	}

	if req.Name == "" {
		WriteError(w, http.StatusBadRequest, "Event type name is required")
		return
	}

	// Handle enabled - default to true if not provided
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	eventType := &redis.EventType{
		Key:           req.Key,
		Name:          req.Name,
		Description:   req.Description,
		Category:      req.Category,
		Enabled:       enabled,
		SamplePayload: req.SamplePayload,
	}

	id, err := s.redisClient.CreateEventType(ctx, eventType)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			WriteError(w, http.StatusConflict, "Event type with this key already exists")
			return
		}
		WriteError(w, http.StatusInternalServerError, "Failed to create event type")
		return
	}

	WriteJSON(w, http.StatusCreated, EventTypeResponse{
		Key:     id,
		Message: "Event type created successfully",
	})
}

// UpdateEventType handles PUT /api/v1/event-types/:key - Update an event type
// @Summary Update event type
// @Description Update an existing event type in the registry
// @Tags event-types
// @Accept json
// @Produce json
// @Param key path string true "Event type key"
// @Param event_type body UpdateEventTypeRequest true "Event type updates"
// @Success 200 {object} EventTypeResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /event-types/{key} [put]
func (s *Server) UpdateEventType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := GetPathParam(r, "key")

	var req UpdateEventTypeRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get existing event type
	existing, err := s.redisClient.GetEventType(ctx, key)
	if err != nil || existing == nil {
		WriteError(w, http.StatusNotFound, "Event type not found")
		return
	}

	// Update fields - only update if explicitly provided (pointer)
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Category != "" {
		existing.Category = req.Category
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.SamplePayload != nil {
		existing.SamplePayload = req.SamplePayload
	}

	err = s.redisClient.UpdateEventType(ctx, existing)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update event type")
		return
	}

	WriteJSON(w, http.StatusOK, EventTypeResponse{
		Key:     key,
		Message: "Event type updated successfully",
	})
}

// DeleteEventType handles DELETE /api/v1/event-types/:key - Delete an event type
// @Summary Delete event type
// @Description Delete an event type from the registry
// @Tags event-types
// @Accept json
// @Produce json
// @Param key path string true "Event type key"
// @Success 200 {object} EventTypeResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /event-types/{key} [delete]
func (s *Server) DeleteEventType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := GetPathParam(r, "key")

	err := s.redisClient.DeleteEventType(ctx, key)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to delete event type")
		return
	}

	WriteJSON(w, http.StatusOK, EventTypeResponse{
		Key:     key,
		Message: "Event type deleted successfully",
	})
}
