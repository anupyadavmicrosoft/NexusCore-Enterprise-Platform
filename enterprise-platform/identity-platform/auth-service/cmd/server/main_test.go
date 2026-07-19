package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nexuscore/identity-platform/auth-service/db"
	"github.com/nexuscore/identity-platform/auth-service/service"
	"github.com/nexuscore/identity-platform/shared-event-library"
	"github.com/nexuscore/identity-platform/shared-jwt-library"
	"github.com/nexuscore/identity-platform/shared-security-library"
)

func TestAuthService_FullLifecycleFlow(t *testing.T) {
	ctx := context.Background()

	// 1. Initialize DB, Cache, and Event Broker
	postgresDB := db.NewMockPostgresDB()
	redisCache := security.NewMemoryStoreMock()
	kafkaBroker := event.NewMemoryEventBroker(3, 1*time.Millisecond)

	auth, err := service.NewAuthService(postgresDB, redisCache, kafkaBroker)
	if err != nil {
		t.Fatalf("failed to boot auth service: %v", err)
	}

	// 2. Validate Bad Credentials Login (Returns invalid credentials error)
	_, err = auth.Login(ctx, "admin@nexuscore.com", "wrong-password", "127.0.0.1", "Go-Test")
	if err == nil || !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}

	// 3. Validate Account Lockout Security Policy (5 failed attempts -> LOCK)
	for i := 0; i < 4; i++ {
		_, _ = auth.Login(ctx, "admin@nexuscore.com", "wrong-password", "127.0.0.1", "Go-Test")
	}
	_, err = auth.Login(ctx, "admin@nexuscore.com", "wrong-password", "127.0.0.1", "Go-Test")
	if err == nil || !strings.Contains(err.Error(), "locked") {
		t.Errorf("expected account to be locked out on the 5th attempt, got: %v", err)
	}

	// Unlock manually for subsequent test scenarios
	user, _ := postgresDB.GetUserByEmail(ctx, "admin@nexuscore.com")
	user.Status = "ACTIVE"
	user.FailedLoginAttempts = 0
	_ = postgresDB.UpdateUser(ctx, user)

	// 4. Validate Login for MFA-Enabled Account (Returns MFA_REQUIRED & ticket_id)
	res, err := auth.Login(ctx, "admin@nexuscore.com", "SecureP@ss123!", "127.0.0.1", "Go-Test")
	if err != nil {
		t.Fatalf("unexpected login failure: %v", err)
	}
	if res["status"] != "MFA_REQUIRED" {
		t.Errorf("expected status 'MFA_REQUIRED', got: %s", res["status"])
	}
	ticketID := res["ticket_id"].(string)

	// 5. Validate MFA Verification & Token Generation
	// Query the OTP cached code from Redis simulation (mock user ID + ":mfa")
	var otpCode string
	// Because of private simulation or internal keys, we can just extract from Redis or set a code
	otpCode, _ = service.GenerateOTPCode()
	_ = redisCache.SetOTP(ctx, user.ID+":mfa", otpCode, 5, 5*time.Minute)

	authRes, err := auth.VerifyMFA(ctx, ticketID, otpCode, "127.0.0.1", "Go-Test")
	if err != nil {
		t.Fatalf("MFA Verification failed: %v", err)
	}

	if authRes["status"] != "AUTHENTICATED" {
		t.Errorf("expected MFA verify status 'AUTHENTICATED', got: %v", authRes["status"])
	}

	accessToken := authRes["access_token"].(string)
	refreshToken := authRes["refresh_token"].(string)
	familyID := authRes["family_id"].(string)
	sessionID := authRes["session_id"].(string)

	// Verify Access Token Signature
	claims, err := jwt.VerifyTokenRS256(accessToken, auth.GetPublicKey())
	if err != nil {
		t.Fatalf("Access token verification failed: %v", err)
	}
	if claims.Subject != user.ID {
		t.Errorf("token claim mismatch: expected %s, got %s", user.ID, claims.Subject)
	}

	// 6. Validate Sliding Refresh Token Rotation (Succeeds under normal flow)
	refreshRes, err := auth.RefreshToken(ctx, refreshToken, familyID, claims)
	if err != nil {
		t.Fatalf("Refresh Token rotation failed: %v", err)
	}

	newRefreshToken := refreshRes["refresh_token"].(string)

	// 7. Validate Sliding Refresh Replay-Attack Mitigation (Reuse of old token triggers immediate revocation of whole family!)
	_, err = auth.RefreshToken(ctx, refreshToken, familyID, claims)
	if err == nil || !errors.Is(err, security.ErrTokenFamilyRevoked) {
		t.Errorf("expected ErrTokenFamilyRevoked on refresh token replay, got: %v", err)
	}

	// Verify the family is dead (new token should fail as well now!)
	_, err = auth.RefreshToken(ctx, newRefreshToken, familyID, claims)
	if err == nil || !errors.Is(err, security.ErrTokenFamilyRevoked) {
		t.Errorf("expected entire family to be invalidated, but new token succeeded or got error: %v", err)
	}

	// 8. Validate Email Verification Pipeline
	standardUser, _ := postgresDB.GetUserByEmail(ctx, "user@nexuscore.com")
	if standardUser.EmailVerified {
		t.Fatal("expected newly seeded standard user's email to be unverified")
	}

	err = auth.RequestEmailVerification(ctx, standardUser.ID)
	if err != nil {
		t.Fatalf("failed to request email verification: %v", err)
	}

	// Verify OTP code matches & completes validation
	emailOTPCode, _ := service.GenerateOTPCode()
	_ = redisCache.SetOTP(ctx, standardUser.ID+":email_verify", emailOTPCode, 3, 10*time.Minute)

	err = auth.ConfirmEmailVerification(ctx, standardUser.ID, emailOTPCode)
	if err != nil {
		t.Fatalf("failed to confirm email verification: %v", err)
	}

	updatedUser, _ := postgresDB.GetUserByID(ctx, standardUser.ID)
	if !updatedUser.EmailVerified {
		t.Error("expected user's email to be marked verified")
	}

	// 9. Validate Forgot/Reset Password Pipeline
	err = auth.ForgotPassword(ctx, "user@nexuscore.com")
	if err != nil {
		t.Fatalf("forgot password error: %v", err)
	}

	// Extract the generated token
	userForReset, _ := postgresDB.GetUserByEmail(ctx, "user@nexuscore.com")
	resetToken := userForReset.ResetToken
	if resetToken == "" {
		t.Fatal("expected reset token to be generated and stored")
	}

	// Test history requirement constraint (re-using current password fails)
	err = auth.ResetPassword(ctx, resetToken, "StandardUser!123")
	if err == nil || !errors.Is(err, service.ErrPasswordHistory) {
		t.Errorf("expected ErrPasswordHistory, got: %v", err)
	}

	// Reset with a new high-strength password
	err = auth.ResetPassword(ctx, resetToken, "NewSecurePassword123!!")
	if err != nil {
		t.Fatalf("password reset failed: %v", err)
	}

	// Verify token is invalidated
	postResetUser, _ := postgresDB.GetUserByEmail(ctx, "user@nexuscore.com")
	if postResetUser.ResetToken != "" {
		t.Error("reset token was not cleared after reset completion")
	}

	// 10. Validate Logout (Cleans up state)
	err = auth.Logout(ctx, sessionID, familyID, user.ID, user.TenantID)
	if err != nil {
		t.Errorf("logout failed: %v", err)
	}

	// Ensure session is removed from cache
	var cachedSess map[string]string
	err = redisCache.GetSession(ctx, sessionID, &cachedSess)
	if err == nil {
		t.Error("expected session to be fully purged from Redis store after logout")
	}
}
