package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nexuscore/identity-platform/tenant-service/db"
	"github.com/nexuscore/identity-platform/tenant-service/service"
)

func TestTenantService_Suite(t *testing.T) {
	ctx := context.Background()

	store := db.NewMockTenantDB()
	svc := service.NewTenantService(store)
	srv := &Server{
		tenantService: svc,
		store:         store,
	}

	var testTenantID string

	t.Run("ProvisionTenant_Success", func(t *testing.T) {
		tenant, err := svc.ProvisionTenant(ctx, "Test Company", "testcompany.com")
		if err != nil {
			t.Fatalf("unexpected error provisioning: %v", err)
		}

		if tenant.Name != "Test Company" {
			t.Errorf("expected 'Test Company', got '%s'", tenant.Name)
		}

		if tenant.Status != "ACTIVE" {
			t.Errorf("expected ACTIVE status, got '%s'", tenant.Status)
		}

		testTenantID = tenant.ID
	})

	t.Run("ProvisionTenant_ValidationFailures", func(t *testing.T) {
		_, err := svc.ProvisionTenant(ctx, "A", "test.com")
		if err != service.ErrInvalidTenantName {
			t.Errorf("expected ErrInvalidTenantName, got: %v", err)
		}

		_, err = svc.ProvisionTenant(ctx, "Valid Name", "bad")
		if err != service.ErrInvalidDomain {
			t.Errorf("expected ErrInvalidDomain, got: %v", err)
		}
	})

	t.Run("GetTenant", func(t *testing.T) {
		tenant, err := svc.GetTenant(ctx, testTenantID)
		if err != nil {
			t.Fatalf("unexpected error getting tenant: %v", err)
		}

		if tenant.ID != testTenantID {
			t.Errorf("expected %s, got %s", testTenantID, tenant.ID)
		}
	})

	t.Run("SuspendAndReactivate", func(t *testing.T) {
		tenant, err := svc.SuspendTenant(ctx, testTenantID)
		if err != nil {
			t.Fatalf("unexpected error suspending: %v", err)
		}

		if tenant.Status != "SUSPENDED" {
			t.Errorf("expected SUSPENDED, got %s", tenant.Status)
		}

		tenant, err = svc.ReactivateTenant(ctx, testTenantID)
		if err != nil {
			t.Fatalf("unexpected error reactivating: %v", err)
		}

		if tenant.Status != "ACTIVE" {
			t.Errorf("expected ACTIVE, got %s", tenant.Status)
		}
	})

	t.Run("CryptoShredTenant", func(t *testing.T) {
		tenant, err := svc.CryptoShredTenant(ctx, testTenantID)
		if err != nil {
			t.Fatalf("failed to shred: %v", err)
		}

		if tenant.Status != "TERMINATED" {
			t.Errorf("expected TERMINATED, got %s", tenant.Status)
		}

		// Ensure encryption key is zerofilled
		for _, b := range tenant.EncryptionKey {
			if b != 0 {
				t.Error("expected encryption key to be fully shredded (overwritten with zeroes)")
				break
			}
		}

		// Verify deleted from store
		_, err = svc.GetTenant(ctx, testTenantID)
		if err != db.ErrTenantNotFound {
			t.Errorf("expected ErrTenantNotFound, got %v", err)
		}
	})

	t.Run("HTTP_Handlers_Provision_And_List", func(t *testing.T) {
		// Provision HTTP Request
		payload := ProvisionTenantRequest{
			Name:   "HTTP Corp",
			Domain: "httpcorp.com",
		}
		payloadBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/tenants/provision", bytes.NewReader(payloadBytes))
		rec := httptest.NewRecorder()

		srv.handleProvisionTenant(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d", rec.Code)
		}

		var resp map[string]interface{}
		_ = json.NewDecoder(rec.Body).Decode(&resp)
		createdID := resp["tenant_id"].(string)

		// List HTTP Request
		reqList := httptest.NewRequest("GET", "/tenants/list", nil)
		recList := httptest.NewRecorder()

		srv.handleListTenants(recList, reqList)

		if recList.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", recList.Code)
		}

		// Shred HTTP Request
		shredPayload := ShredTenantRequest{
			TenantID: createdID,
		}
		shredBytes, _ := json.Marshal(shredPayload)
		reqShred := httptest.NewRequest("POST", "/tenants/shred", bytes.NewReader(shredBytes))
		recShred := httptest.NewRecorder()

		srv.handleShredTenant(recShred, reqShred)

		if recShred.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", recShred.Code)
		}
	})
}
