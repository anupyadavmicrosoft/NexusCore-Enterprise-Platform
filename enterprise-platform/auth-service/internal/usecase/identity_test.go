package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nexuscore/auth-service/internal/domain"
	"github.com/nexuscore/auth-service/internal/repository"
)

func TestIdentityUsecase_Workflow(t *testing.T) {
	// Initialize repos and usecase
	postgresRepo := repository.NewPostgresRepository()
	redisRepo := repository.NewRedisCacheRepository()
	kafkaRepo := repository.NewKafkaPublisher([]string{"localhost:9092"})
	jwtSecret := "test_secret_key"

	uc := NewIdentityUsecase(postgresRepo, redisRepo, kafkaRepo, jwtSecret)
	ctx := context.Background()

	// 1. Test Tenant Creation
	t.Run("Create Tenant Success", func(t *testing.T) {
		tenant, err := uc.CreateTenant(ctx, "Acme Corp", "acme.com")
		if err != nil {
			t.Fatalf("unexpected error creating tenant: %v", err)
		}
		if tenant.Name != "Acme Corp" || tenant.Domain != "acme.com" {
			t.Errorf("tenant values mismatch, got Name=%s, Domain=%s", tenant.Name, tenant.Domain)
		}
	})

	t.Run("Create Tenant Duplicate Domain Validation", func(t *testing.T) {
		_, err := uc.CreateTenant(ctx, "Acme Dual", "acme.com")
		if err == nil {
			t.Fatal("expected error for duplicate tenant domain but got nil")
		}
		if !strings.Contains(err.Error(), "already registered") {
			t.Errorf("expected duplicate domain constraint message, got: %v", err)
		}
	})

	// 2. Test Organization Onboarding
	var orgID string
	t.Run("Create Organization Success", func(t *testing.T) {
		org, err := uc.CreateOrganization(ctx, "tenant_nexuscore_global", "Cloud Engineering Team")
		if err != nil {
			t.Fatalf("unexpected error onboarding organization: %v", err)
		}
		if org.Name != "Cloud Engineering Team" {
			t.Errorf("org name mismatch, expected 'Cloud Engineering Team', got '%s'", org.Name)
		}
		orgID = org.ID
	})

	// 3. Test User Registration
	t.Run("Register User Success", func(t *testing.T) {
		u := domain.User{
			TenantID:       "tenant_nexuscore_global",
			OrganizationID: orgID,
			Email:          "developer@nexuscore.io",
			FullName:       "Junior Developer",
			Role:           "StandardUser",
		}
		registered, err := uc.RegisterUser(ctx, u, "password123")
		if err != nil {
			t.Fatalf("unexpected error registering user: %v", err)
		}
		if registered.Email != "developer@nexuscore.io" {
			t.Errorf("user email mismatch, got %s", registered.Email)
		}
		if !strings.HasPrefix(registered.PasswordHash, "$2a$10$") {
			t.Errorf("expected secure password hash format, got %s", registered.PasswordHash)
		}
	})

	// 4. Test User Authenticate Lifecycle
	t.Run("Authenticate Credentials Success", func(t *testing.T) {
		response, err := uc.Authenticate(ctx, "tenant_nexuscore_global", "principal@nexuscore.io", "password_placeholder")
		// note: principal was pre-seeded with fallback password placeholder
		if err != nil {
			t.Fatalf("unexpected auth error: %v", err)
		}
		if response.AccessToken == "" {
			t.Fatal("expected non-empty JWT token string")
		}
		if response.TokenType != "Bearer" {
			t.Errorf("expected token type Bearer, got %s", response.TokenType)
		}

		// Verify Token
		claims, err := uc.VerifyToken(ctx, response.AccessToken)
		if err != nil {
			t.Fatalf("token verification failed: %v", err)
		}
		if claims.Email != "principal@nexuscore.io" {
			t.Errorf("expected email claim principal@nexuscore.io, got %s", claims.Email)
		}
	})

	t.Run("Authenticate Credentials Failure", func(t *testing.T) {
		_, err := uc.Authenticate(ctx, "tenant_nexuscore_global", "principal@nexuscore.io", "wrong_password")
		if err == nil {
			t.Fatal("expected authentication error for bad password but got nil")
		}
	})

	// 5. Test Zero-Trust ABAC Policy Evaluations
	t.Run("Evaluate Authorization - Admin Allowed", func(t *testing.T) {
		req := domain.AuthzCheckRequest{
			UserID:   "usr_principal_arch", // role: Administrator
			Resource: "api/v1/clusters",
			Action:   "WRITE",
			Context: domain.ABACContext{
				ClientIP:    "192.168.1.5",
				RequestTime: time.Now(),
				IsSecureVPN: false,
			},
		}
		res, err := uc.EvaluateAuthorization(ctx, req)
		if err != nil {
			t.Fatalf("authz evaluation errored: %v", err)
		}
		if !res.Allowed {
			t.Errorf("expected Administrator to be allowed on resources, got Deny: %s", res.Reason)
		}
	})

	t.Run("Evaluate Authorization - Compliance Auditor Denied Outside VPN", func(t *testing.T) {
		req := domain.AuthzCheckRequest{
			UserID:   "usr_compliance_eng", // role: ComplianceAuditor
			Resource: "api/v1/ledger/*",
			Action:   "READ",
			Context: domain.ABACContext{
				ClientIP:    "192.168.1.100", // Non-matching IP subnet
				RequestTime: time.Now(),
				UserDept:    "Security",
				IsSecureVPN: false, // ABAC requires secure line
			},
		}
		res, err := uc.EvaluateAuthorization(ctx, req)
		if err != nil {
			t.Fatalf("authz evaluation errored: %v", err)
		}
		if res.Allowed {
			t.Error("expected Compliance Auditor to be DENIED outside of secure VPN line, but got Allow")
		}
	})
}
