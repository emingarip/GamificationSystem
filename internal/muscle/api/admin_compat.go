package api

import (
	"context"
	"net/http"
	"time"

	"gamification/auth"
	"gamification/config"
	"gamification/models"
	"gamification/neo4j"

	"golang.org/x/crypto/bcrypt"
)

type adminLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type adminUserResponse struct {
	ID        string         `json:"id"`
	Email     string         `json:"email"`
	Name      string         `json:"name"`
	Points    int            `json:"points"`
	Level     int            `json:"level"`
	Badges    []BadgeSummary `json:"badges"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

type updateUserRequest struct {
	Name   *string `json:"name"`
	Email  *string `json:"email"`
	Points *int    `json:"points"`
	Level  *int    `json:"level"`
}

type updateBadgeRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Icon         string `json:"icon"`
	Color        string `json:"color"`
	Points       int    `json:"points"`
	Rarity       string `json:"rarity"`
	Requirements string `json:"requirements"`
}

// corsMiddleware enables local frontend development against the Go API.
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Login issues a local-development admin token.
// @Summary Login
// @Description Authenticate admin user and receive JWT access and refresh tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body adminLoginRequest true "Admin credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/login [post]
// @Example 请求示例
//
//	{
//	  "email": "admin",
//	  "password": "admin123"
//	}
func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	var req adminLoginRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate credentials against configured admin user
	if err := s.validateAdminCredentials(r.Context(), req.Email, req.Password); err != nil {
		WriteError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	tokenPair, err := s.jwtManager.GenerateTokenPair("admin_local", s.config.Admin.Username, req.Email, auth.RoleAdmin)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	now := time.Now().UTC()
	WriteJSON(w, http.StatusOK, LoginResponse{
		Token:        tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User: CurrentUserResponse{
			ID:        "admin_local",
			Email:     req.Email,
			Name:      s.config.Admin.Username,
			Points:    0,
			Level:     99,
			Badges:    []BadgeSummary{},
			CreatedAt: now,
			UpdatedAt: now,
		},
	})
}

// validateAdminCredentials validates the admin credentials against the configured hash
func (s *Server) validateAdminCredentials(ctx context.Context, email, password string) error {
	// Check username matches
	if email != s.config.Admin.Username {
		return config.ErrInvalidCredentials
	}

	// Verify password against bcrypt hash
	return bcrypt.CompareHashAndPassword([]byte(s.config.Admin.PasswordHash), []byte(password))
}

// GetCurrentUser returns the authenticated admin identity.
// @Summary Get current user
// @Description Get the currently authenticated user's information
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} CurrentUserResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /auth/me [get]
func (s *Server) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetUserClaims(r)
	if !ok {
		WriteError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	now := time.Now().UTC()
	WriteJSON(w, http.StatusOK, CurrentUserResponse{
		ID:        claims.UserID,
		Email:     claims.Email,
		Name:      claims.Username,
		Points:    0,
		Level:     99,
		Badges:    []BadgeSummary{},
		CreatedAt: now,
		UpdatedAt: now,
	})
}

// Logout exists for frontend compatibility.
// @Summary Logout
// @Description Log out the current user (client should discard token)
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Security BearerAuth
// @Router /auth/logout [post]
func (s *Server) Logout(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Logged out"})
}

// UpdateUser updates user profile fields used by the admin UI.
// @Summary Update user
// @Description Update user profile fields (name, email, points, level)
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body updateUserRequest true "User update request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [put]
func (s *Server) UpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := GetPathParam(r, "id")

	var req updateUserRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	existing, err := s.neo4jClient.GetUserByID(ctx, userID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "User not found")
		return
	}

	// Use existing values as defaults for nil fields
	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}
	email := existing.Email
	if req.Email != nil {
		email = *req.Email
	}
	level := existing.Level
	if req.Level != nil {
		level = *req.Level
	}
	points := existing.Points
	if req.Points != nil {
		points = *req.Points
	}

	if err := s.neo4jClient.UpdateUserProfile(ctx, userID, name, email, level, points); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	// Update Redis cache and leaderboard if points were updated
	if req.Points != nil {
		delta := points - existing.Points
		if delta != 0 {
			if err := s.redisClient.SetUserPoints(ctx, userID, points); err == nil {
				_ = s.redisClient.UpdateLeaderboard(ctx, userID, delta, "add")
			}
		}
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "User updated successfully"})
}

// DeleteUser deletes a user from Neo4j and Redis.
// @Summary Delete user
// @Description Delete a user and all their associated data
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [delete]
func (s *Server) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := GetPathParam(r, "id")

	if err := s.neo4jClient.DeleteUser(ctx, userID); err != nil {
		WriteError(w, http.StatusNotFound, "User not found")
		return
	}
	_ = s.redisClient.DeleteUserData(ctx, userID)

	WriteJSON(w, http.StatusOK, map[string]string{"message": "User deleted successfully"})
}

// GetBadge returns a single badge for edit flows.
// @Summary Get badge
// @Description Get details of a specific badge by ID
// @Tags badges
// @Accept json
// @Produce json
// @Param id path string true "Badge ID"
// @Success 200 {object} BadgeInfo
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /badges/{id} [get]
func (s *Server) GetBadge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	badgeID := GetPathParam(r, "id")

	badge, err := s.neo4jClient.GetBadgeByID(ctx, badgeID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "Badge not found")
		return
	}

	WriteJSON(w, http.StatusOK, BadgeInfo{
		ID:          badge.ID,
		Name:        badge.Name,
		Description: badge.Description,
		Icon:        badge.Icon,
		Points:      badge.Points,
		Category:    badge.Category,
		Metric:      badge.Metric,
		Target:      badge.Target,
	})
}

// UpdateBadge updates an existing badge.
// @Summary Update badge
// @Description Update an existing badge's properties
// @Tags badges
// @Accept json
// @Produce json
// @Param id path string true "Badge ID"
// @Param badge body CreateBadgeRequest true "Badge update request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /badges/{id} [put]
func (s *Server) UpdateBadge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	badgeID := GetPathParam(r, "id")

	var req CreateBadgeRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Check if badge exists
	_, err := s.neo4jClient.GetBadgeByID(ctx, badgeID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "Badge not found")
		return
	}

	// Check for duplicate name (excluding current badge)
	exists, err := s.neo4jClient.CheckBadgeExists(ctx, "", req.Name, badgeID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to check badge existence")
		return
	}
	if exists {
		WriteError(w, http.StatusConflict, "Badge with this name already exists")
		return
	}

	badge := &neo4j.Badge{
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		Points:      req.Points,
		Category:    req.Rarity,
	}

	if err := s.neo4jClient.UpdateBadge(ctx, badgeID, badge); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update badge")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Badge updated successfully"})
}

// DeleteBadge deletes a badge from Neo4j.
// @Summary Delete badge
// @Description Delete a badge from the system
// @Tags badges
// @Accept json
// @Produce json
// @Param id path string true "Badge ID"
// @Success 200 {object} map[string]string
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /badges/{id} [delete]
func (s *Server) DeleteBadge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	badgeID := GetPathParam(r, "id")

	if err := s.neo4jClient.DeleteBadge(ctx, badgeID); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to delete badge")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Badge deleted successfully"})
}

func inferRuleType(rule RuleInfo) string {
	if rule.Rewards != nil {
		if _, ok := rule.Rewards["badge_id"]; ok {
			return "badge"
		}
	}
	if rule.Points > 0 {
		return "points"
	}
	return "streak"
}

func badgeToSummary(badge models.UserBadge) map[string]string {
	return map[string]string{
		"id":   badge.BadgeID,
		"name": badge.BadgeID,
	}
}
