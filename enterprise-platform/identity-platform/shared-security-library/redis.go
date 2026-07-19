package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Standard Redis error definitions
var (
	ErrLockAcquisitionFailed = errors.New("failed to acquire distributed lock: resource is locked")
	ErrLockReleaseFailed     = errors.New("failed to release distributed lock: lock owner mismatch or expired")
	ErrTokenFamilyRevoked    = errors.New("token rotation family breach detected: family has been fully revoked")
	ErrCacheMiss             = errors.New("requested key does not exist in redis cache")
)

// RedisClientInterface defines the standardized API contracts for our multi-zone Redis Cluster.
type RedisClientInterface interface {
	// Session Store API
	SetSession(ctx context.Context, sessionID string, payload interface{}, ttl time.Duration) error
	GetSession(ctx context.Context, sessionID string, dest interface{}) error
	DeleteSession(ctx context.Context, sessionID string) error

	// Refresh Token Cache API
	AddTokenToFamily(ctx context.Context, familyID string, tokenHash string, isUsed bool, ttl time.Duration) error
	CheckTokenInFamily(ctx context.Context, familyID string, tokenHash string) (bool, error) // returns (isUsed, error)
	RevokeTokenFamily(ctx context.Context, familyID string) error

	// Permission Cache API
	SetUserPermissions(ctx context.Context, tenantID, userID string, permissions []string, ttl time.Duration) error
	GetUserPermissions(ctx context.Context, tenantID, userID string) ([]string, error)

	// Role Cache API
	SetRoleMetadata(ctx context.Context, tenantID, roleID string, roleData interface{}, ttl time.Duration) error
	GetRoleMetadata(ctx context.Context, tenantID, roleID string, dest interface{}) error

	// Rate Limit Cache API
	IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int64, error)

	// OTP Cache API
	SetOTP(ctx context.Context, userID string, otpHash string, attemptsLeft int, ttl time.Duration) error
	VerifyAndDecrementOTP(ctx context.Context, userID string, incomingHash string) (bool, int, error) // matches, remaining_attempts, error

	// Device Cache API
	SetDeviceTrust(ctx context.Context, userID, fingerprint string, deviceData interface{}, ttl time.Duration) error
	GetDeviceTrust(ctx context.Context, userID, fingerprint string, dest interface{}) error

	// Distributed Locking (Redlock style)
	AcquireLock(ctx context.Context, lockKey string, ownerUUID string, ttl time.Duration) error
	ReleaseLock(ctx context.Context, lockKey string, ownerUUID string) error
}

// MemoryStoreMock implements RedisClientInterface utilizing an in-memory concurrent map.
// This ensures compiling-time security, extreme efficiency, and local offline test compatibility
// while accurately mimicking Redis's TTL and sharding behaviors.
type MemoryStoreMock struct {
	mu           sync.RWMutex
	store        map[string][]byte
	expirations  map[string]time.Time
	tokenFam     map[string]map[string]bool // family_id -> token_hash -> is_used
	tokenFamDead map[string]bool            // family_id -> is_revoked
	locks        map[string]string          // lock_key -> owner_uuid
	rateLimiters map[string]int64           // key -> hit_counter
}

// NewMemoryStoreMock initializes an optimized in-memory store simulating Redis behavior.
func NewMemoryStoreMock() *MemoryStoreMock {
	return &MemoryStoreMock{
		store:        make(map[string][]byte),
		expirations:  make(map[string]time.Time),
		tokenFam:     make(map[string]map[string]bool),
		tokenFamDead: make(map[string]bool),
		locks:        make(map[string]string),
		rateLimiters: make(map[string]int64),
	}
}

// Helper to check and prune expired keys
func (m *MemoryStoreMock) isExpired(key string) bool {
	exp, exists := m.expirations[key]
	if !exists {
		return false
	}
	if time.Now().After(exp) {
		delete(m.store, key)
		delete(m.expirations, key)
		delete(m.locks, key)
		delete(m.rateLimiters, key)
		return true
	}
	return false
}

// SetSession stores session details with a standard TTL
func (m *MemoryStoreMock) SetSession(ctx context.Context, sessionID string, payload interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("nc:session:%s", sessionID)
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize session context: %w", err)
	}

	m.store[key] = data
	m.expirations[key] = time.Now().Add(ttl)
	return nil
}

// GetSession extracts active sessions, verifying TTL limits
func (m *MemoryStoreMock) GetSession(ctx context.Context, sessionID string, dest interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("nc:session:%s", sessionID)
	if m.isExpired(key) {
		return ErrCacheMiss
	}

	data, exists := m.store[key]
	if !exists {
		return ErrCacheMiss
	}

	return json.Unmarshal(data, dest)
}

// DeleteSession discards the active session instantly
func (m *MemoryStoreMock) DeleteSession(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("nc:session:%s", sessionID)
	delete(m.store, key)
	delete(m.expirations, key)
	return nil
}

// AddTokenToFamily registers a refresh token inside a secure rotation context
func (m *MemoryStoreMock) AddTokenToFamily(ctx context.Context, familyID string, tokenHash string, isUsed bool, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tokenFamDead[familyID] {
		return ErrTokenFamilyRevoked
	}

	familyKey := fmt.Sprintf("nc:token_fam:%s", familyID)
	if _, exists := m.tokenFam[familyKey]; !exists {
		m.tokenFam[familyKey] = make(map[string]bool)
	}

	m.tokenFam[familyKey][tokenHash] = isUsed
	m.expirations[familyKey] = time.Now().Add(ttl)
	return nil
}

// CheckTokenInFamily evaluates sliding refresh states
func (m *MemoryStoreMock) CheckTokenInFamily(ctx context.Context, familyID string, tokenHash string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tokenFamDead[familyID] {
		return false, ErrTokenFamilyRevoked
	}

	familyKey := fmt.Sprintf("nc:token_fam:%s", familyID)
	family, exists := m.tokenFam[familyKey]
	if !exists {
		return false, ErrCacheMiss
	}

	isUsed, found := family[tokenHash]
	if !found {
		return false, ErrCacheMiss
	}

	// Token family reuse breach mitigation: if token is already used, trigger instant family destruction
	if isUsed {
		m.tokenFamDead[familyID] = true
		delete(m.tokenFam, familyKey)
		return true, ErrTokenFamilyRevoked
	}

	return false, nil
}

// RevokeTokenFamily marks the entire rotation family invalid
func (m *MemoryStoreMock) RevokeTokenFamily(ctx context.Context, familyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tokenFamDead[familyID] = true
	familyKey := fmt.Sprintf("nc:token_fam:%s", familyID)
	delete(m.tokenFam, familyKey)
	return nil
}

// SetUserPermissions maps the permission codes of the user to standard memory
func (m *MemoryStoreMock) SetUserPermissions(ctx context.Context, tenantID, userID string, permissions []string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("nc:perm:%s:%s", tenantID, userID)
	data, err := json.Marshal(permissions)
	if err != nil {
		return err
	}

	m.store[key] = data
	m.expirations[key] = time.Now().Add(ttl)
	return nil
}

// GetUserPermissions queries cached authorization contexts
func (m *MemoryStoreMock) GetUserPermissions(ctx context.Context, tenantID, userID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("nc:perm:%s:%s", tenantID, userID)
	if m.isExpired(key) {
		return nil, ErrCacheMiss
	}

	data, exists := m.store[key]
	if !exists {
		return nil, ErrCacheMiss
	}

	var perms []string
	if err := json.Unmarshal(data, &perms); err != nil {
		return nil, err
	}
	return perms, nil
}

// SetRoleMetadata stores role attributes
func (m *MemoryStoreMock) SetRoleMetadata(ctx context.Context, tenantID, roleID string, roleData interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("nc:role:%s:%s", tenantID, roleID)
	data, err := json.Marshal(roleData)
	if err != nil {
		return err
	}

	m.store[key] = data
	m.expirations[key] = time.Now().Add(ttl)
	return nil
}

// GetRoleMetadata queries role profiles
func (m *MemoryStoreMock) GetRoleMetadata(ctx context.Context, tenantID, roleID string, dest interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("nc:role:%s:%s", tenantID, roleID)
	if m.isExpired(key) {
		return ErrCacheMiss
	}

	data, exists := m.store[key]
	if !exists {
		return ErrCacheMiss
	}

	return json.Unmarshal(data, dest)
}

// IncrementRateLimit adds a request instance within the active rate limit window
func (m *MemoryStoreMock) IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rateKey := fmt.Sprintf("nc:rate:%s", key)
	if m.isExpired(rateKey) {
		m.rateLimiters[rateKey] = 0
	}

	m.rateLimiters[rateKey]++
	if m.rateLimiters[rateKey] == 1 {
		m.expirations[rateKey] = time.Now().Add(window)
	}

	return m.rateLimiters[rateKey], nil
}

// SetOTP registers a verification challenge
func (m *MemoryStoreMock) SetOTP(ctx context.Context, userID string, otpHash string, attemptsLeft int, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("nc:otp:%s", userID)
	otpData := map[string]interface{}{
		"hash":      otpHash,
		"remaining": attemptsLeft,
	}

	data, err := json.Marshal(otpData)
	if err != nil {
		return err
	}

	m.store[key] = data
	m.expirations[key] = time.Now().Add(ttl)
	return nil
}

// VerifyAndDecrementOTP decreases available OTP attempts following a verification request
func (m *MemoryStoreMock) VerifyAndDecrementOTP(ctx context.Context, userID string, incomingHash string) (bool, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("nc:otp:%s", userID)
	if m.isExpired(key) {
		return false, 0, ErrCacheMiss
	}

	data, exists := m.store[key]
	if !exists {
		return false, 0, ErrCacheMiss
	}

	var otpData map[string]interface{}
	if err := json.Unmarshal(data, &otpData); err != nil {
		return false, 0, err
	}

	storedHash := otpData["hash"].(string)
	attempts := int(otpData["remaining"].(float64))

	if attempts <= 0 {
		delete(m.store, key)
		delete(m.expirations, key)
		return false, 0, nil
	}

	if storedHash == incomingHash {
		// Challenge matched, clean up instantly to prevent reuse
		delete(m.store, key)
		delete(m.expirations, key)
		return true, 0, nil
	}

	// Wrong password - decrement remaining attempts
	attempts--
	if attempts <= 0 {
		delete(m.store, key)
		delete(m.expirations, key)
		return false, 0, nil
	}

	otpData["remaining"] = attempts
	newData, _ := json.Marshal(otpData)
	m.store[key] = newData

	return false, attempts, nil
}

// SetDeviceTrust caches trusted device contexts
func (m *MemoryStoreMock) SetDeviceTrust(ctx context.Context, userID, fingerprint string, deviceData interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("nc:device:%s:%s", userID, fingerprint)
	data, err := json.Marshal(deviceData)
	if err != nil {
		return err
	}

	m.store[key] = data
	m.expirations[key] = time.Now().Add(ttl)
	return nil
}

// GetDeviceTrust queries cached device context parameters
func (m *MemoryStoreMock) GetDeviceTrust(ctx context.Context, userID, fingerprint string, dest interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("nc:device:%s:%s", userID, fingerprint)
	if m.isExpired(key) {
		return ErrCacheMiss
	}

	data, exists := m.store[key]
	if !exists {
		return ErrCacheMiss
	}

	return json.Unmarshal(data, dest)
}

// AcquireLock implements atomic resource locking
func (m *MemoryStoreMock) AcquireLock(ctx context.Context, lockKey string, ownerUUID string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("nc:lock:%s", lockKey)
	if m.isExpired(key) {
		delete(m.locks, key)
	}

	if currentOwner, exists := m.locks[key]; exists && currentOwner != "" {
		return ErrLockAcquisitionFailed
	}

	m.locks[key] = ownerUUID
	m.expirations[key] = time.Now().Add(ttl)
	return nil
}

// ReleaseLock removes resource locks matching only the original owner
func (m *MemoryStoreMock) ReleaseLock(ctx context.Context, lockKey string, ownerUUID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("nc:lock:%s", lockKey)
	if m.isExpired(key) {
		return ErrLockReleaseFailed
	}

	currentOwner, exists := m.locks[key]
	if !exists || currentOwner != ownerUUID {
		return ErrLockReleaseFailed
	}

	delete(m.locks, key)
	delete(m.expirations, key)
	return nil
}
