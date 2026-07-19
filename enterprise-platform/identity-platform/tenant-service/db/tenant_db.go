package db

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrTenantNotFound      = errors.New("tenant not found")
	ErrTenantAlreadyExists = errors.New("tenant with this domain already exists")
)

type Tenant struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Domain         string    `json:"domain"`
	Status         string    `json:"status"` // ACTIVE, SUSPENDED, TERMINATED
	DatabaseSchema string    `json:"database_schema"`
	EncryptionKey  []byte    `json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type TenantStore interface {
	Create(ctx context.Context, t *Tenant) error
	GetByID(ctx context.Context, id string) (*Tenant, error)
	GetByDomain(ctx context.Context, domain string) (*Tenant, error)
	Update(ctx context.Context, t *Tenant) error
	List(ctx context.Context, limit, offset int) ([]*Tenant, int, error)
	Delete(ctx context.Context, id string) error
}

type MockTenantDB struct {
	mu      sync.RWMutex
	tenants map[string]*Tenant
}

func NewMockTenantDB() *MockTenantDB {
	db := &MockTenantDB{
		tenants: make(map[string]*Tenant),
	}
	db.seedDemoTenants()
	return db
}

func (db *MockTenantDB) seedDemoTenants() {
	key1 := make([]byte, 32)
	copy(key1, []byte("nexuscore-super-secret-key-32-b"))
	db.tenants["ten-8888-0001"] = &Tenant{
		ID:             "ten-8888-0001",
		Name:           "NexusCore Default Tenant",
		Domain:         "nexuscore.com",
		Status:         "ACTIVE",
		DatabaseSchema: "tenant_schema_nexuscore",
		EncryptionKey:  key1,
		CreatedAt:      time.Now().Add(-100 * 24 * time.Hour),
		UpdatedAt:      time.Now().Add(-100 * 24 * time.Hour),
	}
}

func (db *MockTenantDB) Create(ctx context.Context, t *Tenant) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, existing := range db.tenants {
		if strings.EqualFold(existing.Domain, t.Domain) {
			return ErrTenantAlreadyExists
		}
	}

	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	db.tenants[t.ID] = t
	return nil
}

func (db *MockTenantDB) GetByID(ctx context.Context, id string) (*Tenant, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	t, exists := db.tenants[id]
	if !exists {
		return nil, ErrTenantNotFound
	}

	copied := *t
	return &copied, nil
}

func (db *MockTenantDB) GetByDomain(ctx context.Context, domain string) (*Tenant, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	for _, t := range db.tenants {
		if strings.EqualFold(t.Domain, domain) {
			copied := *t
			return &copied, nil
		}
	}
	return nil, ErrTenantNotFound
}

func (db *MockTenantDB) Update(ctx context.Context, t *Tenant) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.tenants[t.ID]; !exists {
		return ErrTenantNotFound
	}

	t.UpdatedAt = time.Now()
	db.tenants[t.ID] = t
	return nil
}

func (db *MockTenantDB) List(ctx context.Context, limit, offset int) ([]*Tenant, int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var all []*Tenant
	for _, t := range db.tenants {
		copied := *t
		all = append(all, &copied)
	}

	total := len(all)
	if offset >= total {
		return []*Tenant{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return all[offset:end], total, nil
}

func (db *MockTenantDB) Delete(ctx context.Context, id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.tenants[id]; !exists {
		return ErrTenantNotFound
	}

	delete(db.tenants, id)
	return nil
}
