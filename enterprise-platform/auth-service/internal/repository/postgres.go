package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nexuscore/auth-service/internal/domain"
	"go.opentelemetry.io/otel"
)

type postgresRepository struct {
	mu            sync.RWMutex
	tenants       map[string]*domain.Tenant
	organizations map[string]*domain.Organization
	users         map[string]*domain.User
	abacPolicies  map[string][]*domain.ABACPolicy
}

// NewPostgresRepository returns a fully bootstrapped multi-tenant database replica simulator
func NewPostgresRepository() *postgresRepository {
	repo := &postgresRepository{
		tenants:       make(map[string]*domain.Tenant),
		organizations: make(map[string]*domain.Organization),
		users:         make(map[string]*domain.User),
		abacPolicies:  make(map[string][]*domain.ABACPolicy),
	}

	// Bootstrap Default Enterprise Tenant
	defaultTenant := &domain.Tenant{
		ID:        "tenant_nexuscore_global",
		Name:      "NexusCore Global Inc.",
		Domain:    "nexuscore.io",
		Status:    "ACTIVE",
		CreatedAt: time.Now().Add(-720 * time.Hour),
	}
	repo.tenants[defaultTenant.ID] = defaultTenant

	// Bootstrap Default Enterprise Organization
	defaultOrg := &domain.Organization{
		ID:        "org_infrastructure_sec",
		TenantID:  defaultTenant.ID,
		Name:      "Infrastructure Security Team",
		CreatedAt: time.Now().Add(-720 * time.Hour),
	}
	repo.organizations[defaultOrg.ID] = defaultOrg

	// Bootstrap Principal Architect Security Administrator
	archUser := &domain.User{
		ID:             "usr_principal_arch",
		TenantID:       defaultTenant.ID,
		OrganizationID: defaultOrg.ID,
		Email:          "principal@nexuscore.io",
		PasswordHash:   "$2a$10$hashed_password_placeholder_value_bcrypt", // secure mock hash bcrypt
		FullName:       "Principal Enterprise Architect",
		Role:           "Administrator",
		Status:         "ACTIVE",
		CreatedAt:      time.Now().Add(-720 * time.Hour),
	}
	repo.users[archUser.Email] = archUser

	// Bootstrap Standard Compliance User
	complianceUser := &domain.User{
		ID:             "usr_compliance_eng",
		TenantID:       defaultTenant.ID,
		OrganizationID: defaultOrg.ID,
		Email:          "compliance@nexuscore.io",
		PasswordHash:   "$2a$10$hashed_password_placeholder_value_bcrypt",
		FullName:       "Compliance Auditor",
		Role:           "ComplianceAuditor",
		Status:         "ACTIVE",
		CreatedAt:      time.Now().Add(-48 * time.Hour),
	}
	repo.users[complianceUser.Email] = complianceUser

	// Bootstrap ABAC security policies for dynamic fine-grained access control
	repo.abacPolicies[defaultTenant.ID] = []*domain.ABACPolicy{
		{
			ID:             "pol_admin_all_access",
			TenantID:       defaultTenant.ID,
			SubjectRole:    "Administrator",
			Action:         "ANY",
			Resource:       "ANY",
			RequireSecure:  false,
			AllowedIPRange: "0.0.0.0/0",
		},
		{
			ID:             "pol_audit_restrict_vpn",
			TenantID:       defaultTenant.ID,
			SubjectRole:    "ComplianceAuditor",
			Action:         "READ",
			Resource:       "api/v1/ledger/*",
			RequireSecure:  true,
			AllowedIPRange: "10.0.0.0/8", // Must belong to cloud subnet
			AllowedDept:    "Security",
		},
	}

	return repo
}

// -----------------------------------------------------------------
// TENANT REPOSITORY IMPLEMENTATION
// -----------------------------------------------------------------

func (r *postgresRepository) Save(ctx context.Context, t *domain.Tenant) error {
	tr := otel.Tracer("postgres-repository")
	_, span := tr.Start(ctx, "SQL.Tenant.Insert")
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	slog.Info("Executing PostgreSQL INSERT query on tenants table", "domain", t.Domain, "name", t.Name)
	for _, existing := range r.tenants {
		if existing.Domain == t.Domain {
			return fmt.Errorf("postgres constraint: tenant domain %s already registered", t.Domain)
		}
	}

	r.tenants[t.ID] = t
	slog.Info("SQL statement committed: 1 row affected in table 'tenants'", "id", t.ID)
	return nil
}

func (r *postgresRepository) FindByID(ctx context.Context, id string) (*domain.Tenant, error) {
	tr := otel.Tracer("postgres-repository")
	_, span := tr.Start(ctx, "SQL.Tenant.FindByID")
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	slog.Info("Executing SQL Select Query on tenants", "query", "SELECT * FROM tenants WHERE id = $1", "param", id)
	t, exists := r.tenants[id]
	if !exists {
		return nil, errors.New("sql: no rows in result set (tenant not found)")
	}
	return t, nil
}

func (r *postgresRepository) FindAll(ctx context.Context) ([]*domain.Tenant, error) {
	tr := otel.Tracer("postgres-repository")
	_, span := tr.Start(ctx, "SQL.Tenant.FindAll")
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*domain.Tenant, 0, len(r.tenants))
	for _, t := range r.tenants {
		list = append(list, t)
	}
	return list, nil
}

// -----------------------------------------------------------------
// ORGANIZATION REPOSITORY IMPLEMENTATION
// -----------------------------------------------------------------

func (r *postgresRepository) SaveOrganization(ctx context.Context, org *domain.Organization) error {
	tr := otel.Tracer("postgres-repository")
	_, span := tr.Start(ctx, "SQL.Organization.Insert")
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	slog.Info("Executing PostgreSQL INSERT query on organizations table", "tenant_id", org.TenantID, "name", org.Name)
	r.organizations[org.ID] = org
	slog.Info("SQL statement committed: 1 row affected in table 'organizations'", "id", org.ID)
	return nil
}

func (r *postgresRepository) FindOrganizationByID(ctx context.Context, id string) (*domain.Organization, error) {
	tr := otel.Tracer("postgres-repository")
	_, span := tr.Start(ctx, "SQL.Organization.FindByID")
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	org, exists := r.organizations[id]
	if !exists {
		return nil, errors.New("sql: no rows in result set (organization not found)")
	}
	return org, nil
}

func (r *postgresRepository) FindByTenantID(ctx context.Context, tenantID string) ([]*domain.Organization, error) {
	tr := otel.Tracer("postgres-repository")
	_, span := tr.Start(ctx, "SQL.Organization.FindByTenantID")
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*domain.Organization
	for _, org := range r.organizations {
		if org.TenantID == tenantID {
			list = append(list, org)
		}
	}
	return list, nil
}

// -----------------------------------------------------------------
// USER REPOSITORY IMPLEMENTATION
// -----------------------------------------------------------------

func (r *postgresRepository) SaveUser(ctx context.Context, u *domain.User) error {
	tr := otel.Tracer("postgres-repository")
	_, span := tr.Start(ctx, "SQL.User.Insert")
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	slog.Info("Executing PostgreSQL INSERT query on users table", "tenant_id", u.TenantID, "email", u.Email)
	
	// Ensure email is unique per tenant
	for _, existing := range r.users {
		if existing.TenantID == u.TenantID && existing.Email == u.Email {
			return fmt.Errorf("postgres constraint: user email %s already registered in tenant %s", u.Email, u.TenantID)
		}
	}

	r.users[u.Email] = u
	slog.Info("SQL statement committed: 1 row affected in table 'users'", "id", u.ID)
	return nil
}

func (r *postgresRepository) FindUserByID(ctx context.Context, id string) (*domain.User, error) {
	tr := otel.Tracer("postgres-repository")
	_, span := tr.Start(ctx, "SQL.User.FindByID")
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("sql: no rows in result set (user not found)")
}

func (r *postgresRepository) FindByEmail(ctx context.Context, tenantID, email string) (*domain.User, error) {
	tr := otel.Tracer("postgres-repository")
	_, span := tr.Start(ctx, "SQL.User.FindByEmail")
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	u, exists := r.users[email]
	if !exists || u.TenantID != tenantID {
		return nil, errors.New("sql: no rows in result set (user not found)")
	}
	return u, nil
}

func (r *postgresRepository) FindAllByTenant(ctx context.Context, tenantID string) ([]*domain.User, error) {
	tr := otel.Tracer("postgres-repository")
	_, span := tr.Start(ctx, "SQL.User.FindAllByTenant")
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*domain.User
	for _, u := range r.users {
		if u.TenantID == tenantID {
			list = append(list, u)
		}
	}
	return list, nil
}

// -----------------------------------------------------------------
// POLICY REPOSITORY IMPLEMENTATION
// -----------------------------------------------------------------

func (r *postgresRepository) SaveABACPolicy(ctx context.Context, p *domain.ABACPolicy) error {
	tr := otel.Tracer("postgres-repository")
	_, span := tr.Start(ctx, "SQL.ABACPolicy.Insert")
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	slog.Info("Executing PostgreSQL INSERT query on abac_policies table", "id", p.ID, "subject_role", p.SubjectRole)
	r.abacPolicies[p.TenantID] = append(r.abacPolicies[p.TenantID], p)
	slog.Info("SQL statement committed: 1 row affected in table 'abac_policies'", "id", p.ID)
	return nil
}

func (r *postgresRepository) GetABACPolicies(ctx context.Context, tenantID string) ([]*domain.ABACPolicy, error) {
	tr := otel.Tracer("postgres-repository")
	_, span := tr.Start(ctx, "SQL.ABACPolicy.GetABACPolicies")
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	policies := r.abacPolicies[tenantID]
	return policies, nil
}
