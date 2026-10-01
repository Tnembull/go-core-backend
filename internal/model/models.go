package model

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// RBAC Roles
const (
	RoleSuperAdmin = "superadmin"
	RoleAdmin      = "admin"
	RoleEditor     = "editor"
	RoleViewer     = "viewer"
)

// RBAC Permissions
const (
	PermissionWildcard       = "*"
	PermissionUsersRead      = "users:read"
	PermissionUsersWrite     = "users:write"
	PermissionSettingsManage = "settings:manage"
	PermissionAuditRead      = "audit:read"
)

var RolePermissions = map[string][]string{
	RoleSuperAdmin: {PermissionWildcard},
	RoleAdmin:      {PermissionUsersRead, PermissionUsersWrite, PermissionAuditRead},
	RoleEditor:     {PermissionUsersRead, PermissionUsersWrite},
	RoleViewer:     {PermissionUsersRead},
}

func HasPermission(role, permission string) bool {
	perms, exists := RolePermissions[role]
	if !exists {
		return false
	}
	for _, p := range perms {
		if p == PermissionWildcard || p == permission {
			return true
		}
	}
	return false
}

type User struct {
	ID                   string    `json:"id"`
	Email                string    `json:"email"`
	Password             string    `json:"-"` // never serialized
	Role                 string    `json:"role"`
	TwoFactorEnabled     bool      `json:"two_factor_enabled"`
	TwoFactorSecret      string    `json:"-"` // TOTP secret key (base32)
	TwoFactorRecoveryCodes []string `json:"-"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type JWTClaims struct {
	UserID    string   `json:"user_id"`
	Email     string   `json:"email"`
	Role      string   `json:"role"`
	TokenType string   `json:"token_type"` // "access", "refresh", "2fa_preauth"
	jwt.RegisteredClaims
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type Enable2FARequest struct {
	Passcode string `json:"passcode"`
}

type Verify2FARequest struct {
	TempToken string `json:"temp_token"`
	Passcode  string `json:"passcode"`
}

type HealthStatus struct {
	Status      string            `json:"status"`
	Timestamp   time.Time         `json:"timestamp"`
	Uptime      string            `json:"uptime"`
	Environment string            `json:"environment"`
	Version     string            `json:"version"`
	Runtime     map[string]string `json:"runtime"`
}
