package db

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/nexuscore/identity-platform/shared-security-library"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user with this email already exists")
	ErrUserArchived      = errors.New("cannot perform operations on an archived user")
)

type User struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	Email         string    `json:"email"`
	Phone         string    `json:"phone"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Department    string    `json:"department"`
	Title         string    `json:"title"`
	AvatarURL     string    `json:"avatar_url"`
	PasswordHash  string    `json:"-"`
	EmailVerified bool      `json:"email_verified"`
	PhoneVerified bool      `json:"phone_verified"`
	Status        string    `json:"status"` // ACTIVE, DEACTIVATED, SUSPENDED, LOCKED, ARCHIVED (Soft Deleted)
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

type AuditLog struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	ActorID      string    `json:"actor_id"`
	Action       string    `json:"action"` // "CREATE", "READ", "UPDATE_PROFILE", "UPDATE_EMAIL", "UPDATE_PHONE", "DEACTIVATE", "REACTIVATE", "SOFT_DELETE", "HARD_DELETE", "SEARCH"
	TargetUserID string    `json:"target_user_id"`
	Description  string    `json:"description"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	Timestamp    time.Time `json:"timestamp"`
}

type UserStore interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, u *User) error
	Search(ctx context.Context, query string, status string, tenantID string, limit, offset int) ([]*User, int, error)
	DeleteSoft(ctx context.Context, id string) error
	DeleteHard(ctx context.Context, id string) error
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	GetAuditLogs(ctx context.Context, targetUserID string, limit, offset int) ([]*AuditLog, error)
}

type MockUserDB struct {
	mu        sync.RWMutex
	users     map[string]*User
	auditLogs []*AuditLog
}

func NewMockUserDB() *MockUserDB {
	db := &MockUserDB{
		users:     make(map[string]*User),
		auditLogs: make([]*AuditLog, 0),
	}
	db.seedDemoUsers()
	return db
}

func (db *MockUserDB) seedDemoUsers() {
	hash1, _ := security.HashPassword("SecureP@ss123!")
	db.users["usr-9999-0001"] = &User{
		ID:            "usr-9999-0001",
		TenantID:      "ten-8888-0001",
		Email:         "admin@nexuscore.com",
		Phone:         "+15555550001",
		FirstName:     "Enterprise",
		LastName:      "Admin",
		Department:    "Information Technology",
		Title:         "System Administrator",
		AvatarURL:     "https://gravatar.com/avatar/64704533?d=identicon",
		PasswordHash:  hash1,
		EmailVerified: true,
		PhoneVerified: true,
		Status:        "ACTIVE",
		CreatedAt:     time.Now().Add(-100 * 24 * time.Hour),
		UpdatedAt:     time.Now().Add(-100 * 24 * time.Hour),
	}

	hash2, _ := security.HashPassword("StandardUser!123")
	db.users["usr-9999-0002"] = &User{
		ID:            "usr-9999-0002",
		TenantID:      "ten-8888-0001",
		Email:         "user@nexuscore.com",
		Phone:         "+15555550002",
		FirstName:     "John",
		LastName:      "Doe",
		Department:    "Engineering",
		Title:         "Software Engineer",
		AvatarURL:     "https://gravatar.com/avatar/e48f5a1?d=identicon",
		PasswordHash:  hash2,
		EmailVerified: false,
		PhoneVerified: false,
		Status:        "ACTIVE",
		CreatedAt:     time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt:     time.Now().Add(-30 * 24 * time.Hour),
	}
}

func (db *MockUserDB) Create(ctx context.Context, u *User) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, existing := range db.users {
		if strings.EqualFold(existing.Email, u.Email) && existing.Status != "ARCHIVED" {
			return ErrUserAlreadyExists
		}
	}

	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	db.users[u.ID] = u
	return nil
}

func (db *MockUserDB) GetByID(ctx context.Context, id string) (*User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	u, exists := db.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}

	copied := *u
	return &copied, nil
}

func (db *MockUserDB) GetByEmail(ctx context.Context, email string) (*User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	for _, u := range db.users {
		if strings.EqualFold(u.Email, email) && u.Status != "ARCHIVED" {
			copied := *u
			return &copied, nil
		}
	}
	return nil, ErrUserNotFound
}

func (db *MockUserDB) Update(ctx context.Context, u *User) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	existing, exists := db.users[u.ID]
	if !exists {
		return ErrUserNotFound
	}

	if existing.Status == "ARCHIVED" {
		return ErrUserArchived
	}

	u.UpdatedAt = time.Now()
	db.users[u.ID] = u
	return nil
}

func (db *MockUserDB) Search(ctx context.Context, query string, status string, tenantID string, limit, offset int) ([]*User, int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var matched []*User
	query = strings.ToLower(query)

	for _, u := range db.users {
		// Filter by soft deleted/archived
		if u.Status == "ARCHIVED" {
			continue
		}

		// Tenant ID filter
		if tenantID != "" && u.TenantID != tenantID {
			continue
		}

		// Status filter
		if status != "" && !strings.EqualFold(u.Status, status) {
			continue
		}

		// Text Search Query Match
		if query != "" {
			nameMatch := strings.Contains(strings.ToLower(u.FirstName), query) ||
				strings.Contains(strings.ToLower(u.LastName), query) ||
				strings.Contains(strings.ToLower(u.FirstName+" "+u.LastName), query)
			emailMatch := strings.Contains(strings.ToLower(u.Email), query)
			phoneMatch := strings.Contains(u.Phone, query)
			deptMatch := strings.Contains(strings.ToLower(u.Department), query)

			if !nameMatch && !emailMatch && !phoneMatch && !deptMatch {
				continue
			}
		}

		copied := *u
		matched = append(matched, &copied)
	}

	total := len(matched)
	if offset >= total {
		return []*User{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return matched[offset:end], total, nil
}

func (db *MockUserDB) DeleteSoft(ctx context.Context, id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	u, exists := db.users[id]
	if !exists {
		return ErrUserNotFound
	}

	if u.Status == "ARCHIVED" {
		return nil // Already soft-deleted
	}

	now := time.Now()
	u.DeletedAt = &now
	u.Status = "ARCHIVED"

	// GDPR Compliance Cryptographic Shredding
	// Scramble personal attributes to render records permanently anonymous
	u.FirstName = "GDPR-Redacted"
	u.LastName = "Anonymized"
	u.Email = "shredded-" + u.ID + "@nexuscore.internal"
	u.Phone = "shredded-" + u.ID
	u.AvatarURL = ""
	u.Department = "N/A"
	u.Title = "N/A"
	u.PasswordHash = "SHREDDED_MD5_ARGON2ID_HASH_IMPOSSIBLE_TO_SOLVE"

	u.UpdatedAt = now
	return nil
}

func (db *MockUserDB) DeleteHard(ctx context.Context, id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.users[id]; !exists {
		return ErrUserNotFound
	}

	delete(db.users, id)
	return nil
}

func (db *MockUserDB) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	log.ID = "aud-" + time.Now().Format("20060102150405") + "-" + string(rune(len(db.auditLogs)))
	log.Timestamp = time.Now()
	db.auditLogs = append(db.auditLogs, log)
	return nil
}

func (db *MockUserDB) GetAuditLogs(ctx context.Context, targetUserID string, limit, offset int) ([]*AuditLog, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var matched []*AuditLog
	for i := len(db.auditLogs) - 1; i >= 0; i-- { // Reverse chronological order
		log := db.auditLogs[i]
		if targetUserID == "" || log.TargetUserID == targetUserID {
			matched = append(matched, log)
		}
	}

	total := len(matched)
	if offset >= total {
		return []*AuditLog{}, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return matched[offset:end], nil
}
