package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nexuscore/identity-platform/authorization-service/service"
	"github.com/nexuscore/identity-platform/shared-jwt-library"
)

func TestAuthzService_FullSuite(t *testing.T) {
	ctx := context.Background()

	// 1. Initialize core system dependencies
	privKey, pubKey, err := jwt.GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("failed to generate RSA key pair: %v", err)
	}

	rbac := service.NewRBACService()
	apiKey := service.NewAPIKeyService()
	pe := service.NewPolicyEngine()
	validator := service.NewJWTValidator(pubKey)

	s := &Server{
		rbacService:   rbac,
		apiKeyService: apiKey,
		policyEngine:  pe,
		jwtValidator:  validator,
		privateKey:    privKey,
	}

	// -------------------------------------------------------------
	// Test Case 1: Hierarchical RBAC Permission Inheritance
	// -------------------------------------------------------------
	t.Run("RBAC_HierarchicalInheritance", func(t *testing.T) {
		// TENANT_ADMIN should possess "billing:read" directly and "users:read" (inherited from MEMBER)
		allowed, err := rbac.EvaluateAccess(ctx, "ten-1", "TENANT_ADMIN", "billing:read")
		if err != nil || !allowed {
			t.Errorf("expected TENANT_ADMIN to have direct permission billing:read, got error: %v", err)
		}

		allowedInherited, err := rbac.EvaluateAccess(ctx, "ten-1", "TENANT_ADMIN", "users:read")
		if err != nil || !allowedInherited {
			t.Errorf("expected TENANT_ADMIN to inherit users:read from MEMBER, got error: %v", err)
		}

		// GUEST should not possess "billing:write"
		allowedBad, err := rbac.EvaluateAccess(ctx, "ten-1", "GUEST", "billing:write")
		if err == nil && allowedBad {
			t.Error("expected GUEST to be rejected for billing:write")
		}
	})

	// -------------------------------------------------------------
	// Test Case 2: Custom Role Definition
	// -------------------------------------------------------------
	t.Run("RBAC_CustomRoleCreation", func(t *testing.T) {
		customRole := &service.Role{
			ID:          "role-auditor",
			TenantID:    "ten-8888-0001",
			Name:        "AUDITOR",
			ParentRole:  "GUEST",
			Permissions: []string{"audit:read"},
		}

		err := rbac.CreateRole(ctx, customRole)
		if err != nil {
			t.Fatalf("failed to register custom role: %v", err)
		}

		// Check that custom role now has audit:read
		allowed, err := rbac.EvaluateAccess(ctx, "ten-8888-0001", "AUDITOR", "audit:read")
		if err != nil || !allowed {
			t.Errorf("expected AUDITOR to have audit:read, got: %v, allowed: %t", err, allowed)
		}

		// Check that custom role inherits from GUEST (tenant:read)
		allowedInherited, err := rbac.EvaluateAccess(ctx, "ten-8888-0001", "AUDITOR", "tenant:read")
		if err != nil || !allowedInherited {
			t.Errorf("expected AUDITOR to inherit tenant:read from GUEST, got error: %v", err)
		}
	})

	// -------------------------------------------------------------
	// Test Case 3: API Key Management Lifecycle
	// -------------------------------------------------------------
	t.Run("APIKey_Lifecycle", func(t *testing.T) {
		rawKey, err := apiKey.GenerateAPIKey(ctx, "ten-2", "org-2", "Partner Key", []string{"users:read"}, 1*time.Hour)
		if err != nil {
			t.Fatalf("failed to generate api key: %v", err)
		}

		// Validate valid key & scope
		meta, err := apiKey.ValidateAPIKey(ctx, rawKey, "users:read")
		if err != nil {
			t.Fatalf("failed to validate api key: %v", err)
		}
		if meta.TenantID != "ten-2" {
			t.Errorf("meta tenantID mismatch: expected ten-2, got %s", meta.TenantID)
		}

		// Validate scope mismatch
		_, err = apiKey.ValidateAPIKey(ctx, rawKey, "users:delete")
		if err == nil {
			t.Error("expected validation failure due to missing scope users:delete")
		}

		// Revoke key
		err = apiKey.RevokeAPIKey(ctx, meta.ID)
		if err != nil {
			t.Fatalf("failed to deactivate api key: %v", err)
		}

		// Validate revoked key fails
		_, err = apiKey.ValidateAPIKey(ctx, rawKey, "users:read")
		if err == nil || !strings.Contains(err.Error(), "revoked") {
			t.Errorf("expected error indicating key was revoked, got: %v", err)
		}
	})

	// -------------------------------------------------------------
	// Test Case 4: OPA declarative policy engine ABAC simulation
	// -------------------------------------------------------------
	t.Run("OPA_ABAC_PolicyRules", func(t *testing.T) {
		// Rule 1: High Confidentiality requires High Clearance (3), Finance Dept, and Low Risk (<30)
		abacOK := service.ABACContext{
			SubjectDept:      "finance",
			SubjectRiskScore: 15,
			ClearanceLevel:   3,
			Classification:   "highly-confidential",
		}

		allowed, _, err := pe.EvaluateABACAndOPA(ctx, abacOK, "financial_ledger", "read")
		if err != nil || !allowed {
			t.Errorf("expected access ALLOWED for correct finance/high-clearance context, got: %v", err)
		}

		abacBadDept := service.ABACContext{
			SubjectDept:      "engineering",
			SubjectRiskScore: 10,
			ClearanceLevel:   3,
			Classification:   "highly-confidential",
		}

		allowedBad, _, _ := pe.EvaluateABACAndOPA(ctx, abacBadDept, "financial_ledger", "read")
		if allowedBad {
			t.Error("expected access DENIED for non-finance department")
		}

		abacHighRisk := service.ABACContext{
			SubjectDept:      "finance",
			SubjectRiskScore: 85,
			ClearanceLevel:   3,
			Classification:   "highly-confidential",
		}

		allowedRisk, _, _ := pe.EvaluateABACAndOPA(ctx, abacHighRisk, "financial_ledger", "read")
		if allowedRisk {
			t.Error("expected access DENIED for high risk score")
		}

		// Rule 2: Writes/Deletes are DENIED from outside the internal zone
		abacExternalWrite := service.ABACContext{
			NetworkZone: "external",
		}
		allowedExternal, _, _ := pe.EvaluateABACAndOPA(ctx, abacExternalWrite, "billing_profile", "write")
		if allowedExternal {
			t.Error("expected administrative mutations from external network zones to be rejected")
		}
	})

	// -------------------------------------------------------------
	// Test Case 5: HTTP Endpoints Evaluation Gateway Integration
	// -------------------------------------------------------------
	t.Run("HTTP_REST_API_Evaluate", func(t *testing.T) {
		// Prepare a self-signed JWT using the test server's private key
		testClaims := jwt.Claims{
			Subject:     "usr-http-test",
			TenantID:    "ten-8888-0001",
			Role:        "TENANT_ADMIN",
			Permissions: []string{"users:read", "billing:read"},
			Expiry:      time.Now().Add(1 * time.Hour).Unix(),
		}
		token, err := jwt.SignTokenRS256(testClaims, privKey, "test-kid")
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		// Construct endpoint request
		reqPayload := AuthzEvaluationRequest{
			JWT:                token,
			RequiredPermission: "users:read",
			Resource:           "users_directory",
			Action:             "read",
		}

		payloadBytes, _ := json.Marshal(reqPayload)
		req := httptest.NewRequest("POST", "/authz/evaluate", bytes.NewReader(payloadBytes))
		rec := httptest.NewRecorder()

		s.handleEvaluate(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200 OK, got: %d", rec.Code)
		}

		var resp AuthzEvaluationResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to parse evaluation response: %v", err)
		}

		if !resp.Allowed {
			t.Errorf("expected access to be allowed, got: %v, reason: %s", resp.Allowed, resp.Reason)
		}
	})
}
