package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrAPIKeyExpired = errors.New("the api key has expired")
	ErrAPIKeyRevoked = errors.New("the api key has been suspended or revoked")
	ErrAPIKeyInvalid = errors.New("the provided api key signature is invalid")
)

type APIKeyMetadata struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	Hash      string    `json:"hash"` // SHA-256 hash of the key
	Prefix    string    `json:"prefix"`
	Scopes    []string  `json:"scopes"`
	Status    string    `json:"status"` // ACTIVE, REVOKED
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type APIKeyService struct {
	mu   sync.RWMutex
	keys map[string]*APIKeyMetadata // hash -> APIKeyMetadata
}

func NewAPIKeyService() *APIKeyService {
	s := &APIKeyService{
		keys: make(map[string]*APIKeyMetadata),
	}
	s.seedDemoAPIKey()
	return s
}

// ComputeHash computes SHA-256 sum of the raw API Key
func (s *APIKeyService) ComputeHash(rawKey string) string {
	h := sha256.New()
	h.Write([]byte(rawKey))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *APIKeyService) seedDemoAPIKey() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Seed a production-ready demo API key
	// Key: nc_live_7a3d90e2f581c8b74f32a51f8a7e
	rawKey := "nc_live_7a3d90e2f581c8b74f32a51f8a7e"
	hash := s.ComputeHash(rawKey)

	s.keys[hash] = &APIKeyMetadata{
		ID:        "key-0001",
		TenantID:  "ten-8888-0001",
		OrgID:     "org-7777-0001",
		Name:      "Production Integration Key",
		Hash:      hash,
		Prefix:    "nc_live",
		Scopes:    []string{"tenant:read", "users:read", "users:create"},
		Status:    "ACTIVE",
		ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
}

// GenerateAPIKey generates a new secure API Key and stores its SHA-256 hash
func (s *APIKeyService) GenerateAPIKey(ctx context.Context, tenantID, orgID, name string, scopes []string, duration time.Duration) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure random seed: %w", err)
	}

	rawKey := fmt.Sprintf("nc_live_%s", hex.EncodeToString(bytes))
	hash := s.ComputeHash(rawKey)

	metadata := &APIKeyMetadata{
		ID:        fmt.Sprintf("key-%s", hex.EncodeToString(bytes[:4])),
		TenantID:  tenantID,
		OrgID:     orgID,
		Name:      name,
		Hash:      hash,
		Prefix:    "nc_live",
		Scopes:    scopes,
		Status:    "ACTIVE",
		ExpiresAt: time.Now().Add(duration),
		CreatedAt: time.Now(),
	}

	s.keys[hash] = metadata
	return rawKey, nil
}

// ValidateAPIKey validates the incoming raw key against stored hashes and scopes
func (s *APIKeyService) ValidateAPIKey(ctx context.Context, rawKey string, requiredScope string) (*APIKeyMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hash := s.ComputeHash(rawKey)
	metadata, exists := s.keys[hash]
	if !exists {
		return nil, ErrAPIKeyInvalid
	}

	if metadata.Status != "ACTIVE" {
		return nil, ErrAPIKeyRevoked
	}

	if !metadata.ExpiresAt.IsZero() && time.Now().After(metadata.ExpiresAt) {
		return nil, ErrAPIKeyExpired
	}

	// If a scope condition is required, check scope match
	if requiredScope != "" {
		scopeMatched := false
		for _, scope := range metadata.Scopes {
			if scope == "*" || scope == requiredScope {
				scopeMatched = true
				break
			}
			// Wildcard suffix mapping (e.g. "users:*" matches "users:read")
			if strings.HasSuffix(scope, ":*") {
				prefix := strings.TrimSuffix(scope, ":*")
				if strings.HasPrefix(requiredScope, prefix) {
					scopeMatched = true
					break
				}
			}
		}
		if !scopeMatched {
			return nil, fmt.Errorf("key does not possess the required authorization scope: %s", requiredScope)
		}
	}

	return metadata, nil
}

// RevokeAPIKey invalidates an API Key instantly
func (s *APIKeyService) RevokeAPIKey(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, k := range s.keys {
		if k.ID == id {
			k.Status = "REVOKED"
			return nil
		}
	}
	return errors.New("api key metadata not found")
}
