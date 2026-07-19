package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nexuscore/identity-platform/shared-auth-library"
)

var (
	ErrRoleNotFound       = errors.New("specified role does not exist")
	ErrPermissionNotFound = errors.New("specified permission does not exist")
	ErrDuplicateRole      = errors.New("role with this identifier already exists")
)

type Role struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"` // Global roles have tenant_id = "global"
	Name        string    `json:"name"`
	ParentRole  string    `json:"parent_role,omitempty"` // For hierarchy: e.g., "TENANT_ADMIN" inherits from "MEMBER"
	Permissions []string  `json:"permissions"`           // Permission codes
	CreatedAt   time.Time `json:"created_at"`
}

type Permission struct {
	Code        string    `json:"code"` // e.g., "users:create", "billing:write"
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type RBACService struct {
	mu          sync.RWMutex
	roles       map[string]*Role       // tenant_id:role_name -> Role
	permissions map[string]*Permission // code -> Permission
}

func NewRBACService() *RBACService {
	s := &RBACService{
		roles:       make(map[string]*Role),
		permissions: make(map[string]*Permission),
	}
	s.seedDefaultRBAC()
	return s
}

func (s *RBACService) seedDefaultRBAC() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Seed Standard Permissions
	defaultPerms := []*Permission{
		{Code: "tenant:read", Description: "Read tenant metadata"},
		{Code: "tenant:write", Description: "Modify tenant configuration"},
		{Code: "users:create", Description: "Create new tenant users"},
		{Code: "users:read", Description: "Query tenant users profiles"},
		{Code: "users:write", Description: "Modify user profile attributes"},
		{Code: "users:delete", Description: "Remove users from tenant bounds"},
		{Code: "billing:read", Description: "View subscription billing records"},
		{Code: "billing:write", Description: "Modify billing subscriptions"},
		{Code: "audit:read", Description: "Query tenant audit trails"},
	}

	for _, p := range defaultPerms {
		s.permissions[p.Code] = p
	}

	// Seed Hierarchical Roles
	// GUEST -> MEMBER -> ORG_ADMIN -> TENANT_ADMIN -> SYSTEM_ADMIN
	s.roles["global:GUEST"] = &Role{
		ID:          "role-guest",
		TenantID:    "global",
		Name:        "GUEST",
		Permissions: []string{"tenant:read"},
		CreatedAt:   time.Now(),
	}

	s.roles["global:MEMBER"] = &Role{
		ID:          "role-member",
		TenantID:    "global",
		Name:        "MEMBER",
		ParentRole:  "GUEST",
		Permissions: []string{"users:read"},
		CreatedAt:   time.Now(),
	}

	s.roles["global:ORG_ADMIN"] = &Role{
		ID:          "role-org-admin",
		TenantID:    "global",
		Name:        "ORG_ADMIN",
		ParentRole:  "MEMBER",
		Permissions: []string{"users:create", "users:write"},
		CreatedAt:   time.Now(),
	}

	s.roles["global:TENANT_ADMIN"] = &Role{
		ID:          "role-tenant-admin",
		TenantID:    "global",
		Name:        "TENANT_ADMIN",
		ParentRole:  "ORG_ADMIN",
		Permissions: []string{"users:delete", "billing:read", "billing:write", "audit:read"},
		CreatedAt:   time.Now(),
	}

	s.roles["global:SYSTEM_ADMIN"] = &Role{
		ID:          "role-system-admin",
		TenantID:    "global",
		Name:        "SYSTEM_ADMIN",
		Permissions: []string{"*"}, // Wildcard administrative rights
		CreatedAt:   time.Now(),
	}
}

// GetEffectivePermissions traverses the role hierarchy to resolve all inherited permissions
func (s *RBACService) GetEffectivePermissions(ctx context.Context, tenantID, roleName string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var allPerms []string
	visited := make(map[string]bool)

	currentRoleName := roleName
	for currentRoleName != "" {
		if visited[currentRoleName] {
			break // Guard against circular hierarchy structures
		}
		visited[currentRoleName] = true

		// Check tenant-specific role first, then fall back to global role
		key := tenantID + ":" + currentRoleName
		r, exists := s.roles[key]
		if !exists {
			key = "global:" + currentRoleName
			r, exists = s.roles[key]
		}

		if !exists {
			return nil, ErrRoleNotFound
		}

		allPerms = append(allPerms, r.Permissions...)
		currentRoleName = r.ParentRole
	}

	return allPerms, nil
}

// CreateRole adds a tenant-scoped or global role configuration
func (s *RBACService) CreateRole(ctx context.Context, r *Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r.TenantID == "" {
		r.TenantID = "global"
	}

	key := r.TenantID + ":" + r.Name
	if _, exists := s.roles[key]; exists {
		return ErrDuplicateRole
	}

	r.CreatedAt = time.Now()
	s.roles[key] = r
	return nil
}

// EvaluateAccess assesses standard RBAC permission matches using wildcard structures
func (s *RBACService) EvaluateAccess(ctx context.Context, tenantID, roleName string, requiredPermission string) (bool, error) {
	perms, err := s.GetEffectivePermissions(ctx, tenantID, roleName)
	if err != nil {
		return false, err
	}

	return auth.CheckPermission(perms, requiredPermission), nil
}
