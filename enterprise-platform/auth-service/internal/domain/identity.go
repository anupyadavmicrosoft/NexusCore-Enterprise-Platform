package domain

import (
	"context"
	"time"
)

// -----------------------------------------------------------------
// 1. TENANT & ORGANIZATION DOMAINS (Multi-Tenant Architecture)
// -----------------------------------------------------------------

type Tenant struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Domain    string    `json:"domain" db:"domain"`
	Status    string    `json:"status" db:"status"` // ACTIVE, SUSPENDED
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Organization struct {
	ID        string    `json:"id" db:"id"`
	TenantID  string    `json:"tenant_id" db:"tenant_id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// -----------------------------------------------------------------
// 2. USER DOMAIN
// -----------------------------------------------------------------

type User struct {
	ID             string    `json:"id" db:"id"`
	TenantID       string    `json:"tenant_id" db:"tenant_id"`
	OrganizationID string    `json:"organization_id" db:"organization_id"`
	Email          string    `json:"email" db:"email"`
	PasswordHash   string    `json:"-" db:"password_hash"`
	FullName       string    `json:"full_name" db:"full_name"`
	Role           string    `json:"role" db:"role"` // RBAC Primary Role
	Status         string    `json:"status" db:"status"` // ACTIVE, LOCKED
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// -----------------------------------------------------------------
// 3. AUTHENTICATION & OIDC / OAUTH2 DOMAIN
// -----------------------------------------------------------------

type OAuthClient struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"-"`
	RedirectURIs []string `json:"redirect_uris"`
	AllowedScopes []string `json:"allowed_scopes"`
}

type TokenClaims struct {
	UserID         string   `json:"sub"`
	TenantID       string   `json:"tenant_id"`
	OrganizationID string   `json:"org_id"`
	Email          string   `json:"email"`
	Role           string   `json:"role"`
	Scopes         []string `json:"scopes"`
	Issuer         string   `json:"iss"`
	ExpiresAt      int64    `json:"exp"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	IDToken      string `json:"id_token,omitempty"` // OpenID Connect compatibility
}

// -----------------------------------------------------------------
// 4. AUTHORIZATION DOMAIN (RBAC & ABAC Policy Definitions)
// -----------------------------------------------------------------

// RBACPermission maps an action to a secure target resource
type RBACPermission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"` // READ, WRITE, DELETE, EXECUTE
}

// ABACContext represents environmental and attribute context for dynamic rules evaluation
type ABACContext struct {
	ClientIP     string    `json:"client_ip"`
	RequestTime  time.Time `json:"request_time"`
	ResourceDept string    `json:"resource_dept"`
	UserDept     string    `json:"user_dept"`
	IsSecureVPN  bool      `json:"is_secure_vpn"`
}

// ABACPolicy establishes dynamic conditional constraints
type ABACPolicy struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenant_id"`
	SubjectRole    string `json:"subject_role"`
	Action         string `json:"action"`
	Resource       string `json:"resource"`
	RequireSecure  bool   `json:"require_secure"`
	AllowedIPRange string `json:"allowed_ip_range,omitempty"` // CIDR block or subnet matching
	AllowedDept    string `json:"allowed_dept,omitempty"`
}

type AuthzCheckRequest struct {
	UserID   string      `json:"user_id" binding:"required"`
	Resource string      `json:"resource" binding:"required"`
	Action   string      `json:"action" binding:"required"`
	Context  ABACContext `json:"context"`
}

type AuthzCheckResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason"`
}

// -----------------------------------------------------------------
// 5. INFRASTRUCTURE & REPOSITORY PORTS
// -----------------------------------------------------------------

type TenantRepository interface {
	Save(ctx context.Context, t *Tenant) error
	FindByID(ctx context.Context, id string) (*Tenant, error)
	FindAll(ctx context.Context) ([]*Tenant, error)
}

type OrganizationRepository interface {
	Save(ctx context.Context, org *Organization) error
	FindByID(ctx context.Context, id string) (*Organization, error)
	FindByTenantID(ctx context.Context, tenantID string) ([]*Organization, error)
}

type UserRepository interface {
	Save(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, tenantID, email string) (*User, error)
	FindAllByTenant(ctx context.Context, tenantID string) ([]*User, error)
}

type PolicyRepository interface {
	SaveABACPolicy(ctx context.Context, p *ABACPolicy) error
	GetABACPolicies(ctx context.Context, tenantID string) ([]*ABACPolicy, error)
}

type RedisCacheRepository interface {
	SetSession(ctx context.Context, token string, claims *TokenClaims, expiration time.Duration) error
	GetSession(ctx context.Context, token string) (*TokenClaims, error)
	RevokeSession(ctx context.Context, token string) error
}

type KafkaPublisher interface {
	PublishEvent(ctx context.Context, eventType string, payload interface{}) error
}

// -----------------------------------------------------------------
// 6. BUSINESS LOGIC SERVICES PORTS
// -----------------------------------------------------------------

type IdentityUsecase interface {
	// Tenant operations
	CreateTenant(ctx context.Context, name, domain string) (*Tenant, error)
	GetTenant(ctx context.Context, id string) (*Tenant, error)

	// Organization operations
	CreateOrganization(ctx context.Context, tenantID, name string) (*Organization, error)
	GetOrganization(ctx context.Context, id string) (*Organization, error)

	// User operations
	RegisterUser(ctx context.Context, u User, password string) (*User, error)
	GetUser(ctx context.Context, id string) (*User, error)

	// Security operations
	Authenticate(ctx context.Context, tenantID, email, password string) (*TokenResponse, error)
	VerifyToken(ctx context.Context, token string) (*TokenClaims, error)
	EvaluateAuthorization(ctx context.Context, req AuthzCheckRequest) (*AuthzCheckResponse, error)

	// Policy operations
	CreateABACPolicy(ctx context.Context, policy ABACPolicy) error
}
