package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/nexuscore/identity-platform/tenant-service/db"
)

var (
	ErrInvalidTenantName = errors.New("invalid tenant name: must be at least 3 characters")
	ErrInvalidDomain     = errors.New("invalid domain: must be a valid fully qualified domain name")
	ErrEmptyTenantID     = errors.New("tenant ID cannot be empty")
)

type TenantService struct {
	store db.TenantStore
}

func NewTenantService(store db.TenantStore) *TenantService {
	return &TenantService{
		store: store,
	}
}

func GenerateTenantUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("ten-%s", hex.EncodeToString(b)[:12])
}

func (s *TenantService) ProvisionTenant(ctx context.Context, name, domain string) (*db.Tenant, error) {
	if len(name) < 3 {
		return nil, ErrInvalidTenantName
	}
	if !strings.Contains(domain, ".") || len(domain) < 4 {
		return nil, ErrInvalidDomain
	}

	// Generate a secure 32-byte AES key for tenant-specific encryption
	encKey := make([]byte, 32)
	if _, err := rand.Read(encKey); err != nil {
		return nil, fmt.Errorf("failed to generate secure key: %w", err)
	}

	domainClean := strings.ToLower(strings.TrimSpace(domain))
	tenant := &db.Tenant{
		ID:             GenerateTenantUUID(),
		Name:           name,
		Domain:         domainClean,
		Status:         "ACTIVE",
		DatabaseSchema: "tenant_schema_" + strings.ReplaceAll(domainClean, ".", "_"),
		EncryptionKey:  encKey,
	}

	err := s.store.Create(ctx, tenant)
	if err != nil {
		return nil, err
	}

	return tenant, nil
}

func (s *TenantService) GetTenant(ctx context.Context, id string) (*db.Tenant, error) {
	if id == "" {
		return nil, ErrEmptyTenantID
	}
	return s.store.GetByID(ctx, id)
}

func (s *TenantService) GetTenantByDomain(ctx context.Context, domain string) (*db.Tenant, error) {
	if domain == "" {
		return nil, ErrInvalidDomain
	}
	return s.store.GetByDomain(ctx, domain)
}

func (s *TenantService) ListTenants(ctx context.Context, limit, offset int) ([]*db.Tenant, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return s.store.List(ctx, limit, offset)
}

func (s *TenantService) SuspendTenant(ctx context.Context, id string) (*db.Tenant, error) {
	t, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	t.Status = "SUSPENDED"
	err = s.store.Update(ctx, t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (s *TenantService) ReactivateTenant(ctx context.Context, id string) (*db.Tenant, error) {
	t, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	t.Status = "ACTIVE"
	err = s.store.Update(ctx, t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (s *TenantService) CryptoShredTenant(ctx context.Context, id string) (*db.Tenant, error) {
	t, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Dynamic Crypto-shredding (overwrite encryption key with zeros to render all data unreadable)
	for i := range t.EncryptionKey {
		t.EncryptionKey[i] = 0
	}
	t.Status = "TERMINATED"

	err = s.store.Update(ctx, t)
	if err != nil {
		return nil, err
	}

	// We can choose to delete from the stores or keep as metadata
	err = s.store.Delete(ctx, id)
	if err != nil {
		return nil, err
	}

	return t, nil
}
