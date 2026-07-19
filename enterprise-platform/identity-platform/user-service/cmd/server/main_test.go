package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nexuscore/identity-platform/user-service/db"
	"github.com/nexuscore/identity-platform/user-service/service"
)

func TestUserService_FullSuite(t *testing.T) {
	ctx := context.Background()

	// 1. Initialize core system layers
	store := db.NewMockUserDB()
	svc := service.NewUserService(store)

	s := &Server{
		userService: svc,
		store:       store,
	}

	// -------------------------------------------------------------
	// Test Case 1: Password Length Constraint & Verification Fails
	// -------------------------------------------------------------
	t.Run("CreateUser_PasswordStrengthValidation", func(t *testing.T) {
		_, err := svc.CreateUser(ctx, "too-short@nexuscore.com", "", "Short123!", "Short", "User", "IT", "Title", "ten-1", "actor-1", "127.0.0.1", "test-agent")
		if err == nil || err != service.ErrInvalidPassword {
			t.Errorf("expected ErrInvalidPassword for short password, got error: %v", err)
		}
	})

	// -------------------------------------------------------------
	// Test Case 2: Email Syntax Constraints
	// -------------------------------------------------------------
	t.Run("CreateUser_EmailFormatValidation", func(t *testing.T) {
		_, err := svc.CreateUser(ctx, "bademailaddress", "", "PasswordMustBeSuperLong123!", "Bad", "Email", "IT", "Title", "ten-1", "actor-1", "127.0.0.1", "test-agent")
		if err == nil || err != service.ErrInvalidEmail {
			t.Errorf("expected ErrInvalidEmail for bad syntax, got error: %v", err)
		}
	})

	// -------------------------------------------------------------
	// Test Case 3: Create User successfully
	// -------------------------------------------------------------
	var testUserID string
	t.Run("CreateUser_Success", func(t *testing.T) {
		user, err := svc.CreateUser(ctx, "test.user@nexuscore.com", "+12025550199", "PasswordMustBeSuperLong123!", "Test", "User", "Sales", "Sales Director", "ten-8888-0001", "actor-1", "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("unexpected failure during user creation: %v", err)
		}

		if user.Status != "ACTIVE" {
			t.Errorf("expected status 'ACTIVE', got '%s'", user.Status)
		}

		testUserID = user.ID
	})

	// -------------------------------------------------------------
	// Test Case 4: Read Profile & Update Profile Attributes
	// -------------------------------------------------------------
	t.Run("Profile_ReadAndUpdate", func(t *testing.T) {
		// Read Profile
		user, err := svc.GetUser(ctx, testUserID, "actor-1", "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("failed to read user profile: %v", err)
		}
		if user.FirstName != "Test" || user.LastName != "User" {
			t.Errorf("name mismatch, got %s %s", user.FirstName, user.LastName)
		}

		// Update Profile
		updated, err := svc.UpdateProfile(ctx, testUserID, "TestModified", "UserModified", "Marketing", "Marketing Manager", "actor-1", "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("failed to update user profile: %v", err)
		}

		if updated.FirstName != "TestModified" || updated.Department != "Marketing" {
			t.Errorf("failed to verify field modifications, got: %+v", updated)
		}
	})

	// -------------------------------------------------------------
	// Test Case 5: Avatar base64 processing
	// -------------------------------------------------------------
	t.Run("Profile_AvatarUpload", func(t *testing.T) {
		avatarBase64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
		user, err := svc.UpdateAvatar(ctx, testUserID, avatarBase64, "profile.png", "actor-1", "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("failed to upload avatar: %v", err)
		}

		if !strings.Contains(user.AvatarURL, "profile.png") {
			t.Errorf("expected avatar url to contain filename, got: %s", user.AvatarURL)
		}
	})

	// -------------------------------------------------------------
	// Test Case 6: Email & Phone modification
	// -------------------------------------------------------------
	t.Run("Contact_EmailAndPhoneUpdate", func(t *testing.T) {
		// Update Email
		user, err := svc.UpdateEmail(ctx, testUserID, "updated.email@nexuscore.com", "actor-1", "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("failed to update email: %v", err)
		}
		if user.Email != "updated.email@nexuscore.com" {
			t.Errorf("expected email to be updated, got: %s", user.Email)
		}
		if user.EmailVerified {
			t.Error("email should be marked unverified after modification")
		}

		// Update Phone
		user, err = svc.UpdatePhone(ctx, testUserID, "+15555559999", "actor-1", "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("failed to update phone: %v", err)
		}
		if user.Phone != "+15555559999" {
			t.Errorf("expected phone to be updated, got: %s", user.Phone)
		}
		if user.PhoneVerified {
			t.Error("phone should be marked unverified after modification")
		}
	})

	// -------------------------------------------------------------
	// Test Case 7: Search, Filtering, Pagination
	// -------------------------------------------------------------
	t.Run("Search_FiltersAndPagination", func(t *testing.T) {
		// Search matched on department
		results, total, err := svc.SearchUsers(ctx, "Marketing", "", "ten-8888-0001", 10, 0, "actor-1", "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("search users failed: %v", err)
		}
		if total != 1 || len(results) != 1 {
			t.Errorf("expected 1 result from query, got %d (len: %d)", total, len(results))
		}

		// Pagination limit check
		resultsPaginated, totalP, _ := svc.SearchUsers(ctx, "", "", "", 1, 0, "actor-1", "127.0.0.1", "test-agent")
		if totalP < 2 {
			t.Errorf("expected at least 2 database users, got %d", totalP)
		}
		if len(resultsPaginated) != 1 {
			t.Errorf("pagination limit was set to 1, but retrieved: %d", len(resultsPaginated))
		}
	})

	// -------------------------------------------------------------
	// Test Case 8: Manual Status deactivation / activation
	// -------------------------------------------------------------
	t.Run("Status_DeactivateAndReactivate", func(t *testing.T) {
		// Deactivate
		user, err := svc.DeactivateUser(ctx, testUserID, "actor-1", "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("failed to deactivate user: %v", err)
		}
		if user.Status != "DEACTIVATED" {
			t.Errorf("expected state to be DEACTIVATED, got: %s", user.Status)
		}

		// Reactivate
		user, err = svc.ReactivateUser(ctx, testUserID, "actor-1", "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("failed to reactivate user: %v", err)
		}
		if user.Status != "ACTIVE" {
			t.Errorf("expected state to be ACTIVE, got: %s", user.Status)
		}
	})

	// -------------------------------------------------------------
	// Test Case 9: GDPR Soft Delete (Crypto Shredding)
	// -------------------------------------------------------------
	t.Run("Delete_SoftDeleteGdprShredding", func(t *testing.T) {
		err := svc.SoftDeleteUser(ctx, testUserID, "actor-1", "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("failed to soft delete: %v", err)
		}

		// Retrieve soft-deleted user (should return details, but with status ARCHIVED and shredded values)
		user, err := svc.GetUser(ctx, testUserID, "actor-1", "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("expected to still read user meta chain, but got error: %v", err)
		}

		if user.Status != "ARCHIVED" {
			t.Errorf("expected state to be ARCHIVED, got: %s", user.Status)
		}

		if user.FirstName != "GDPR-Redacted" || user.LastName != "Anonymized" {
			t.Errorf("expected GDPR shredded fields, got: %s %s", user.FirstName, user.LastName)
		}

		// Search should omit soft-deleted/archived users
		results, _, _ := svc.SearchUsers(ctx, "Marketing", "", "ten-8888-0001", 10, 0, "actor-1", "127.0.0.1", "test-agent")
		if len(results) > 0 {
			t.Error("expected soft deleted users to be omitted from standard searches")
		}
	})

	// -------------------------------------------------------------
	// Test Case 10: Hard Delete
	// -------------------------------------------------------------
	t.Run("Delete_HardDeletePermanence", func(t *testing.T) {
		err := svc.HardDeleteUser(ctx, testUserID, "actor-1", "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("failed to hard delete user: %v", err)
		}

		// Read should now fail with UserNotFound
		_, err = svc.GetUser(ctx, testUserID, "actor-1", "127.0.0.1", "test-agent")
		if err != db.ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got: %v", err)
		}
	})

	// -------------------------------------------------------------
	// Test Case 11: Audit Trail Verification
	// -------------------------------------------------------------
	t.Run("Audit_TrailCompliance", func(t *testing.T) {
		logs, err := svc.GetAuditLogs(ctx, "", 100, 0)
		if err != nil {
			t.Fatalf("failed to query audit logs: %v", err)
		}

		if len(logs) == 0 {
			t.Error("expected audit logs to be recorded for compliance")
		}

		// Find soft delete and hard delete events
		foundSoft := false
		foundHard := false
		for _, log := range logs {
			if log.Action == "SOFT_DELETE" {
				foundSoft = true
			}
			if log.Action == "HARD_DELETE" {
				foundHard = true
			}
		}

		if !foundSoft || !foundHard {
			t.Errorf("expected soft-delete (%t) and hard-delete (%t) events in audit chain", foundSoft, foundHard)
		}
	})

	// -------------------------------------------------------------
	// Test Case 12: HTTP REST endpoint gateways
	// -------------------------------------------------------------
	t.Run("HTTP_REST_Gateways", func(t *testing.T) {
		// Test legacy Create User handler
		payload := map[string]string{
			"email":      "endpoint.test@nexuscore.com",
			"password":   "MustBeAtLeast14CharactersLong!",
			"first_name": "Endpoint",
			"last_name":  "Tester",
			"tenant_id":  "ten-8888-0001",
		}
		payloadBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/users/create", bytes.NewReader(payloadBytes))
		rec := httptest.NewRecorder()

		s.handleLegacyCreateUser(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected 201 Created from legacy route, got: %d", rec.Code)
		}

		var resp map[string]interface{}
		_ = json.NewDecoder(rec.Body).Decode(&resp)
		createdID := resp["user_id"].(string)

		// Test Search Route
		searchPayload := map[string]string{
			"query": "Tester",
		}
		searchPayloadBytes, _ := json.Marshal(searchPayload)
		reqSearch := httptest.NewRequest("POST", "/users/search", bytes.NewReader(searchPayloadBytes))
		recSearch := httptest.NewRecorder()

		s.handleSearchUsers(recSearch, reqSearch)
		if recSearch.Code != http.StatusOK {
			t.Errorf("expected search route to return 200 OK, got: %d", recSearch.Code)
		}

		// Test Deactivate Route
		deactPayload := map[string]string{
			"user_id": createdID,
		}
		deactBytes, _ := json.Marshal(deactPayload)
		reqDeact := httptest.NewRequest("POST", "/users/deactivate", bytes.NewReader(deactBytes))
		recDeact := httptest.NewRecorder()

		s.handleDeactivate(recDeact, reqDeact)
		if recDeact.Code != http.StatusOK {
			t.Errorf("expected deactivate route to return 200 OK, got: %d", recDeact.Code)
		}

		// Verify Deactivated in DB
		user, _ := store.GetByID(ctx, createdID)
		if user.Status != "DEACTIVATED" {
			t.Errorf("expected user status to be DEACTIVATED, got: %s", user.Status)
		}
	})
}
