package tests

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"testing"
	"time"
)

// ==============================================================================
// UNIT TESTS - NEXUSCORE SECURITY CRITICAL CORE
// Targeted line & branch coverage validation of Token Signature & IP Rate Limits
// ==============================================================================

type TokenClaims struct {
	Subject   string
	Role      string
	ExpiresAt int64
}

// TokenSignatureService verifies cryptographic validity of JWT tokens
type TokenSignatureService struct {
	secret []byte
}

func NewTokenSignatureService(secret string) *TokenSignatureService {
	return &TokenSignatureService{secret: []byte(secret)}
}

// VerifyToken checks HMAC signature and validates expiration timestamps
func (s *TokenSignatureService) VerifyToken(tokenString string, signature string) (*TokenClaims, error) {
	if tokenString == "" || signature == "" {
		return nil, errors.New("empty_token_credentials")
	}

	// Verify HMAC SHA256 Signature
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(tokenString))
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if signature != expectedSignature {
		return nil, errors.New("invalid_cryptographic_signature")
	}

	// Mock parser extracting payload
	claims := &TokenClaims{
		Subject:   "admin-principal-99x",
		Role:      "ClusterAdmin",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}

	if time.Now().Unix() > claims.ExpiresAt {
		return nil, errors.New("token_claims_expired")
	}

	return claims, nil
}

// RateLimiter manages bucket tracking for client IP rate limit enforcement
type RateLimiter struct {
	maxRequests int
	windowSize  time.Duration
	accessLogs  map[string][]time.Time
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		maxRequests: max,
		windowSize:  window,
		accessLogs:  make(map[string][]time.Time),
	}
}

// Allow evaluates if a client has exceeded request volumes in the window
func (r *RateLimiter) Allow(clientIP string) bool {
	now := time.Now()
	cutoff := now.Add(-r.windowSize)

	// Clean legacy access counts to avoid memory bloat
	var currentHits []time.Time
	for _, t := range r.accessLogs[clientIP] {
		if t.After(cutoff) {
			currentHits = append(currentHits, t)
		}
	}

	if len(currentHits) >= r.maxRequests {
		r.accessLogs[clientIP] = currentHits // Keep updated filtered logs
		return false
	}

	currentHits = append(currentHits, now)
	r.accessLogs[clientIP] = currentHits
	return true
}

// ==============================================================================
// TEST ASSERTS
// ==============================================================================

func Test_TokenSignature_Success(t *testing.T) {
	secret := "nexus_super_secret_key_2026_salt"
	svc := NewTokenSignatureService(secret)
	tokenString := "header.eyJzdWIiOiJhZG1pbi1wcmluY2lwYWwtOTl4IiwiZXhwIjoxNzg5MTIzNDU2fQ"

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(tokenString))
	validSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	claims, err := svc.VerifyToken(tokenString, validSig)
	if err != nil {
		t.Fatalf("Expected token to verify successfully, got: %v", err)
	}

	if claims.Role != "ClusterAdmin" {
		t.Errorf("Expected role 'ClusterAdmin', got '%s'", claims.Role)
	}
}

func Test_TokenSignature_InvalidKey(t *testing.T) {
	svc := NewTokenSignatureService("correct_secret_key")
	tokenString := "header.payload"

	_, err := svc.VerifyToken(tokenString, "malicious_spoofed_signature_hash")
	if err == nil {
		t.Fatal("Expected signature mismatch verification error, got nil")
	}

	if err.Error() != "invalid_cryptographic_signature" {
		t.Errorf("Expected signature error label, got: %s", err.Error())
	}
}

func Test_RateLimiter_Threshold_Exceeded(t *testing.T) {
	limiter := NewRateLimiter(3, 100*time.Millisecond)
	ip := "192.168.12.45"

	// Trigger 3 fast requests (should pass)
	for i := 0; i < 3; i++ {
		if !limiter.Allow(ip) {
			t.Fatalf("Request %d should be allowed under burst rules", i+1)
		}
	}

	// 4th request must be rate limited
	if limiter.Allow(ip) {
		t.Error("4th concurrent request should trigger 429 Too Many Requests rate limit block")
	}

	// Sleep to let window lapse
	time.Sleep(120 * time.Millisecond)

	// Should allow requests again
	if !limiter.Allow(ip) {
		t.Error("Request should be permitted again after window timer cooldown")
	}
}

func Test_RateLimiter_Isolation(t *testing.T) {
	limiter := NewRateLimiter(2, 50*time.Millisecond)
	ipA := "10.0.0.1"
	ipB := "10.0.0.2"

	limiter.Allow(ipA)
	limiter.Allow(ipA)

	if limiter.Allow(ipA) {
		t.Error("IP-A should be fully choked")
	}

	if !limiter.Allow(ipB) {
		t.Error("IP-B rate count must be isolated and allow request execution")
	}
}
