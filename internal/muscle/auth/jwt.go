package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken indicates the token is invalid
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken indicates the token has expired
	ErrExpiredToken = errors.New("token expired")
	// ErrMissingToken indicates no token was provided
	ErrMissingToken = errors.New("missing token")
	// ErrForbidden indicates access is forbidden
	ErrForbidden = errors.New("forbidden")
	// ErrUnauthorized indicates not authenticated
	ErrUnauthorized = errors.New("unauthorized")
)

// JWTConfig holds JWT configuration
type JWTConfig struct {
	SecretKey          string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	Issuer             string
}

// DefaultJWTConfig returns default JWT configuration
func DefaultJWTConfig() *JWTConfig {
	return &JWTConfig{
		SecretKey:          "muscle-gamification-secret-key-change-in-production",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "muscle-gamification",
	}
}

// JWTManager handles JWT token operations
type JWTManager struct {
	config *JWTConfig
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(cfg *JWTConfig) *JWTManager {
	if cfg == nil {
		cfg = DefaultJWTConfig()
	}
	// If secret key is too short, generate a secure one
	if len(cfg.SecretKey) < 32 {
		cfg.SecretKey = generateSecureKey()
	}
	return &JWTManager{config: cfg}
}

// generateSecureKey generates a secure random key
func generateSecureKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// HashToken creates a SHA256 hash of a token for storage
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// GenerateTokenPair creates access and refresh tokens
func (m *JWTManager) GenerateTokenPair(userID, username, email string, role Role) (*TokenPair, error) {
	now := time.Now()

	// Generate access token
	accessClaims := jwt.MapClaims{
		"sub":         userID,
		"user_id":     userID,
		"username":    username,
		"email":       email,
		"role":        string(role),
		"permissions": getPermissionsForRole(role),
		"iat":         now.Unix(),
		"exp":         now.Add(m.config.AccessTokenExpiry).Unix(),
		"iss":         m.config.Issuer,
		"type":        "access",
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(m.config.SecretKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token
	refreshClaims := jwt.MapClaims{
		"sub":     userID,
		"user_id": userID,
		"type":    "refresh",
		"iat":     now.Unix(),
		"exp":     now.Add(m.config.RefreshTokenExpiry).Unix(),
		"iss":     m.config.Issuer,
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(m.config.SecretKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(m.config.AccessTokenExpiry.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// getPermissionsForRole returns permissions for a given role
func getPermissionsForRole(role Role) []string {
	perms, ok := RolePermissions[role]
	if !ok {
		return []string{}
	}
	result := make([]string, len(perms))
	for i, p := range perms {
		result[i] = string(p)
	}
	return result
}

// ValidateToken validates a JWT token and returns claims
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.SecretKey), nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	// Verify token type
	tokenType, _ := claims["type"].(string)
	if tokenType != "access" {
		return nil, ErrInvalidToken
	}

	// Extract claims
	userID, _ := claims["user_id"].(string)
	username, _ := claims["username"].(string)
	email, _ := claims["email"].(string)
	roleStr, _ := claims["role"].(string)

	if userID == "" {
		return nil, ErrInvalidToken
	}

	// Parse role
	role := Role(roleStr)
	if role == "" {
		role = RoleUser
	}

	// Parse times
	issuedAt := time.Now()
	if iat, ok := claims["iat"].(float64); ok {
		issuedAt = time.Unix(int64(iat), 0)
	}

	expiresAt := time.Now()
	if exp, ok := claims["exp"].(float64); ok {
		expiresAt = time.Unix(int64(exp), 0)
	}

	return &Claims{
		UserID:    userID,
		Username:  username,
		Email:     email,
		Role:      role,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}, nil
}

// RefreshTokens refreshes access and refresh tokens
func (m *JWTManager) RefreshTokens(refreshToken string) (*TokenPair, error) {
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.SecretKey), nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	// Verify token type
	tokenType, _ := claims["type"].(string)
	if tokenType != "refresh" {
		return nil, ErrInvalidToken
	}

	userID, _ := claims["user_id"].(string)
	if userID == "" {
		return nil, ErrInvalidToken
	}

	// Generate new tokens (in a real system, you'd fetch user data from DB)
	return m.GenerateTokenPair(userID, "", "", RoleUser)
}

// contextKey is used for context values
type contextKey string

const (
	// ClaimsContextKey is the key for storing claims in context
	ClaimsContextKey contextKey = "jwt_claims"
)

// ContextWithClaims adds claims to context
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, ClaimsContextKey, claims)
}

// ClaimsFromContext retrieves claims from context
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(*Claims)
	return claims, ok
}

// AuthMiddleware creates JWT authentication middleware
func (m *JWTManager) AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// Try to get token from query parameter (for WebSocket)
				authHeader = r.URL.Query().Get("token")
				if authHeader == "" {
					WriteAuthError(w, http.StatusUnauthorized, "Missing authorization token")
					return
				}
			}

			// Handle "Bearer <token>" format
			parts := strings.SplitN(authHeader, " ", 2)
			var token string
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				token = parts[1]
			} else {
				token = authHeader
			}

			// Validate token
			claims, err := m.ValidateToken(token)
			if err != nil {
				if errors.Is(err, ErrExpiredToken) {
					WriteAuthError(w, http.StatusUnauthorized, "Token expired")
				} else {
					WriteAuthError(w, http.StatusUnauthorized, "Invalid token")
				}
				return
			}

			// Add claims to context
			ctx := ContextWithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole creates middleware that requires specific role
func RequireRole(roles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				WriteAuthError(w, http.StatusUnauthorized, "Authentication required")
				return
			}

			// Check if user has required role
			for _, role := range roles {
				if claims.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Also allow if user has admin permission
			if claims.IsAdmin() {
				next.ServeHTTP(w, r)
				return
			}

			WriteAuthError(w, http.StatusForbidden, "Insufficient permissions")
		})
	}
}

// RequirePermission creates middleware that requires specific permission
func RequirePermission(permissions ...Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				WriteAuthError(w, http.StatusUnauthorized, "Authentication required")
				return
			}

			// Check if user has any of the required permissions
			for _, permission := range permissions {
				if claims.HasPermission(permission) {
					next.ServeHTTP(w, r)
					return
				}
			}

			WriteAuthError(w, http.StatusForbidden, "Insufficient permissions")
		})
	}
}

// OptionalAuth creates optional auth middleware (won't block if no token)
func (m *JWTManager) OptionalAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			parts := strings.SplitN(authHeader, " ", 2)
			var token string
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				token = parts[1]
			} else if len(parts) == 1 {
				token = parts[0]
			}

			if token != "" {
				claims, err := m.ValidateToken(token)
				if err == nil {
					ctx := ContextWithClaims(r.Context(), claims)
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// WriteAuthError writes authentication error response
func WriteAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":"` + message + `"}`))
}

// WebSocketAuth authenticates WebSocket connections
func (m *JWTManager) WebSocketAuth(r *http.Request) (*Claims, error) {
	// First try to get from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		// Try from Authorization header
		authHeader := r.Header.Get("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			token = parts[1]
		} else if len(parts) == 1 {
			token = parts[0]
		}
	}

	if token == "" {
		return nil, ErrMissingToken
	}

	return m.ValidateToken(token)
}

// GetUserID extracts user ID from request context
func GetUserID(r *http.Request) (string, bool) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		return "", false
	}
	return claims.UserID, true
}

// GetUserClaims extracts full claims from request
func GetUserClaims(r *http.Request) (*Claims, bool) {
	return ClaimsFromContext(r.Context())
}

// AdminOnly creates middleware that requires admin role
func AdminOnly() func(http.Handler) http.Handler {
	return RequireRole(RoleAdmin)
}

// UserIDParam extracts user ID from path parameter
func UserIDParam(r *http.Request) string {
	return chi.URLParam(r, "id")
}

// GetTokenFromRequest extracts token from request (header or query)
func GetTokenFromRequest(r *http.Request) string {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
		return authHeader
	}

	// Check query parameter
	return r.URL.Query().Get("token")
}
