package db

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nexuscore/identity-platform/shared-security-library"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserLocked        = errors.New("user account is locked due to security policy")
	ErrInvalidVerificationCode = errors.New("invalid or expired verification code")
)

type UserDB struct {
	ID                 string
	TenantID           string
	Email              string
	Phone              string
	PasswordHash       string
	EmailVerified      bool
	PhoneVerified      bool
	MFAEnabled         bool
	MFASecret          string
	Status             string // ACTIVE, SUSPENDED, LOCKED
	FailedLoginAttempts int
	LockedUntil        time.Time
	ResetToken         string
	ResetTokenExpires  time.Time
}

type MockPostgresDB struct {
	mu    sync.RWMutex
	users map[string]*UserDB // email -> user
}

func NewMockPostgresDB() *MockPostgresDB {
	db := &MockPostgresDB{
		users: make(map[string]*UserDB),
	}
	db.SeedDemoData()
	return db
}

func (db *MockPostgresDB) SeedDemoData() {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Seed some production-ready accounts
	hash1, _ := security.HashPassword("SecureP@ss123!")
	db.users["admin@nexuscore.com"] = &UserDB{
		ID:                 "usr-9999-0001",
		TenantID:           "ten-8888-0001",
		Email:              "admin@nexuscore.com",
		Phone:              "+15555550001",
		PasswordHash:       hash1,
		EmailVerified:      true,
		PhoneVerified:      true,
		MFAEnabled:         true,
		MFASecret:          "NEXUSCOREBASE32SECRETKEY",
		Status:             "ACTIVE",
		FailedLoginAttempts: 0,
	}

	hash2, _ := security.HashPassword("StandardUser!123")
	db.users["user@nexuscore.com"] = &UserDB{
		ID:                 "usr-9999-0002",
		TenantID:           "ten-8888-0001",
		Email:              "user@nexuscore.com",
		Phone:              "+15555550002",
		PasswordHash:       hash2,
		EmailVerified:      false,
		PhoneVerified:      false,
		MFAEnabled:         false,
		Status:             "ACTIVE",
		FailedLoginAttempts: 0,
	}
}

func (db *MockPostgresDB) GetUserByEmail(ctx context.Context, email string) (*UserDB, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	user, exists := db.users[email]
	if !exists {
		return nil, ErrUserNotFound
	}

	// Deep copy to prevent side effects
	copied := *user
	return &copied, nil
}

func (db *MockPostgresDB) GetUserByID(ctx context.Context, userID string) (*UserDB, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	for _, u := range db.users {
		if u.ID == userID {
			copied := *u
			return &copied, nil
		}
	}
	return nil, ErrUserNotFound
}

func (db *MockPostgresDB) UpdateUser(ctx context.Context, user *UserDB) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Locate and update
	for email, u := range db.users {
		if u.ID == user.ID {
			db.users[email] = user
			return nil
		}
	}

	// If update by email key mapping directly
	db.users[user.Email] = user
	return nil
}
