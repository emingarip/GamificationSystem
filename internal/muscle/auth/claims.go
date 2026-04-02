package auth

import "time"

// Role represents user role in the system
type Role string

const (
	// RoleAdmin has full access to all endpoints
	RoleAdmin Role = "admin"
	// RoleModerator has access to moderate content and users
	RoleModerator Role = "moderator"
	// RoleUser has basic access to view and interact
	RoleUser Role = "user"
)

// Permission represents a specific permission
type Permission string

const (
	// PermissionRead allows reading data
	PermissionRead Permission = "read"
	// PermissionWrite allows creating/updating data
	PermissionWrite Permission = "write"
	// PermissionDelete allows deleting data
	PermissionDelete Permission = "delete"
	// PermissionAdmin allows admin-level operations
	PermissionAdmin Permission = "admin"
)

// RolePermissions maps roles to their permissions
var RolePermissions = map[Role][]Permission{
	RoleAdmin:     {PermissionRead, PermissionWrite, PermissionDelete, PermissionAdmin},
	RoleModerator: {PermissionRead, PermissionWrite},
	RoleUser:      {PermissionRead},
}

// HasPermission checks if a role has a specific permission
func (r Role) HasPermission(permission Permission) bool {
	perms, ok := RolePermissions[r]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}

// Claims represents JWT claims for authenticated user
type Claims struct {
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Role        Role      `json:"role"`
	Permissions []string  `json:"permissions,omitempty"`
	IssuedAt    time.Time `json:"iat"`
	ExpiresAt   time.Time `json:"exp"`
	RefreshAt   time.Time `json:"refresh_at,omitempty"`
}

// IsAdmin checks if the user has admin role
func (c *Claims) IsAdmin() bool {
	return c.Role == RoleAdmin
}

// IsModerator checks if the user has moderator role or above
func (c *Claims) IsModerator() bool {
	return c.Role == RoleAdmin || c.Role == RoleModerator
}

// HasPermission checks if the claims have a specific permission
func (c *Claims) HasPermission(permission Permission) bool {
	return c.Role.HasPermission(permission)
}

// CanAccessResource checks if the user can access a specific resource
// userID is the owner of the resource (for ownership checks)
func (c *Claims) CanAccessResource(resourceOwner string) bool {
	// Admins can access everything
	if c.IsAdmin() {
		return true
	}
	// Moderators can access any user resource
	if c.IsModerator() {
		return true
	}
	// Regular users can only access their own resources
	return c.UserID == resourceOwner
}

// TokenPair contains access and refresh tokens
// @Description JWT token pair for authentication
type TokenPair struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiresIn    int64  `json:"expires_in" example:3600`
	TokenType    string `json:"token_type" example:"Bearer"`
}

// LoginRequest represents a login request
// @Description Admin login credentials
type LoginRequest struct {
	Username string `json:"username" example:"admin"`
	Password string `json:"password" example:"securepassword123"`
}

// LoginResponse represents a login response
// @Description Login response with access token and user info
type LoginResponse struct {
	TokenPair TokenPair `json:"token_pair"`
	User      UserInfo  `json:"user"`
}

// UserInfo represents user information in responses
// @Description Basic user information
type UserInfo struct {
	ID       string `json:"id" example:"user_123"`
	Username string `json:"username" example:"admin"`
	Email    string `json:"email" example:"admin@example.com"`
	Role     Role   `json:"role" example:"admin"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ErrorResponse represents an auth error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}
