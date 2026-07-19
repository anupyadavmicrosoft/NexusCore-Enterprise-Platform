package auth

import (
	"context"
	"errors"
	"strings"
)

// Keys for Context mapping
type contextKey string

const (
	ContextKeyTenantID    contextKey = "tenant_id"
	ContextKeyOrgID       contextKey = "org_id"
	ContextKeyUserID      contextKey = "user_id"
	ContextKeyRole        contextKey = "role"
	ContextKeyPermissions contextKey = "permissions"
)

// Common Error Definitions
var (
	ErrUnauthenticated = errors.New("authentication header missing or invalidly formatted")
	ErrForbidden       = errors.New("access denied: insufficient permissions to complete action")
)

// AuthContext holds validated security context variables inside running routines.
type AuthContext struct {
	TenantID    string
	OrgID       string
	UserID      string
	Role        string
	Permissions []string
}

// InjectIntoContext binds verified AuthContext variables onto a standard Go context.
func InjectIntoContext(ctx context.Context, actx *AuthContext) context.Context {
	ctx = context.WithValue(ctx, ContextKeyTenantID, actx.TenantID)
	ctx = context.WithValue(ctx, ContextKeyOrgID, actx.OrgID)
	ctx = context.WithValue(ctx, ContextKeyUserID, actx.UserID)
	ctx = context.WithValue(ctx, ContextKeyRole, actx.Role)
	ctx = context.WithValue(ctx, ContextKeyPermissions, actx.Permissions)
	return ctx
}

// ExtractFromContext retrieves active identity variables from running routines context.
func ExtractFromContext(ctx context.Context) (*AuthContext, error) {
	tenantID, _ := ctx.Value(ContextKeyTenantID).(string)
	orgID, _ := ctx.Value(ContextKeyOrgID).(string)
	userID, _ := ctx.Value(ContextKeyUserID).(string)
	role, _ := ctx.Value(ContextKeyRole).(string)
	permissions, _ := ctx.Value(ContextKeyPermissions).([]string)

	if tenantID == "" || userID == "" {
		return nil, ErrUnauthenticated
	}

	return &AuthContext{
		TenantID:    tenantID,
		OrgID:       orgID,
		UserID:      userID,
		Role:        role,
		Permissions: permissions,
	}, nil
}

// CheckPermission evaluates if the current context possesses the required permission scope.
func CheckPermission(permissions []string, required string) bool {
	for _, perm := range permissions {
		// Wildcard match (e.g. "tenant:billing:*" matches "tenant:billing:write")
		if perm == "*" {
			return true
		}
		if strings.HasSuffix(perm, ":*") {
			prefix := strings.TrimSuffix(perm, ":*")
			if strings.HasPrefix(required, prefix) {
				return true
			}
		}
		if perm == required {
			return true
		}
	}
	return false
}
