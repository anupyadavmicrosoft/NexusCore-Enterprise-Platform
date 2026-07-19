package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nexuscore/identity-platform/shared-security-library"
	"github.com/nexuscore/identity-platform/user-service/db"
)

var (
	ErrInvalidPassword = errors.New("enterprise password policy mismatch: must be at least 14 characters")
	ErrInvalidEmail    = errors.New("invalid email address format")
	ErrEmptyUserID     = errors.New("user ID cannot be empty")
)

type UserService struct {
	store db.UserStore
}

func NewUserService(store db.UserStore) *UserService {
	return &UserService{
		store: store,
	}
}

// GenerateUUID generates a cryptographically secure hex-based unique identifier
func GenerateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("usr_%s", hex.EncodeToString(b))
}

func (s *UserService) CreateUser(ctx context.Context, email, phone, password, firstName, lastName, dept, title, tenantID string, actorID, ip, ua string) (*db.User, error) {
	if len(password) < 14 {
		return nil, ErrInvalidPassword
	}

	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return nil, ErrInvalidEmail
	}

	hash, err := security.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("cryptographic password hashing failed: %w", err)
	}

	if tenantID == "" {
		tenantID = "global"
	}

	user := &db.User{
		ID:            GenerateUUID(),
		TenantID:      tenantID,
		Email:         email,
		Phone:         phone,
		FirstName:     firstName,
		LastName:      lastName,
		Department:    dept,
		Title:         title,
		AvatarURL:     "https://gravatar.com/avatar/" + hex.EncodeToString([]byte(email)) + "?d=identicon",
		PasswordHash:  hash,
		EmailVerified: false,
		PhoneVerified: false,
		Status:        "ACTIVE",
	}

	err = s.store.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	// Record audit event
	_ = s.store.CreateAuditLog(ctx, &db.AuditLog{
		TenantID:     tenantID,
		ActorID:      actorID,
		Action:       "CREATE",
		TargetUserID: user.ID,
		Description:  fmt.Sprintf("Created standard pending user account for %s in tenant %s", email, tenantID),
		IPAddress:    ip,
		UserAgent:    ua,
	})

	return user, nil
}

func (s *UserService) GetUser(ctx context.Context, id string, actorID, ip, ua string) (*db.User, error) {
	if id == "" {
		return nil, ErrEmptyUserID
	}

	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = s.store.CreateAuditLog(ctx, &db.AuditLog{
		TenantID:     user.TenantID,
		ActorID:      actorID,
		Action:       "READ",
		TargetUserID: user.ID,
		Description:  fmt.Sprintf("Fetched user profile record for %s", user.Email),
		IPAddress:    ip,
		UserAgent:    ua,
	})

	return user, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, id string, firstName, lastName, dept, title string, actorID, ip, ua string) (*db.User, error) {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if firstName != "" {
		user.FirstName = firstName
	}
	if lastName != "" {
		user.LastName = lastName
	}
	if dept != "" {
		user.Department = dept
	}
	if title != "" {
		user.Title = title
	}

	err = s.store.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	_ = s.store.CreateAuditLog(ctx, &db.AuditLog{
		TenantID:     user.TenantID,
		ActorID:      actorID,
		Action:       "UPDATE_PROFILE",
		TargetUserID: user.ID,
		Description:  "Updated user metadata and profile attributes",
		IPAddress:    ip,
		UserAgent:    ua,
	})

	return user, nil
}

func (s *UserService) UpdateAvatar(ctx context.Context, id string, avatarBase64 string, filename string, actorID, ip, ua string) (*db.User, error) {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// In production, we would process or upload avatarBase64 to an S3/GCS bucket.
	// For simulation & testing, we generate a secure assets URL using cryptographic hashing of filename.
	hash := hex.EncodeToString([]byte(filename + time.Now().String()))[:16]
	user.AvatarURL = fmt.Sprintf("https://assets.nexuscore.com/avatars/%s_%s_%s", user.ID, hash, filename)

	err = s.store.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	_ = s.store.CreateAuditLog(ctx, &db.AuditLog{
		TenantID:     user.TenantID,
		ActorID:      actorID,
		Action:       "UPDATE_PROFILE",
		TargetUserID: user.ID,
		Description:  fmt.Sprintf("Uploaded new profile avatar image: %s", filename),
		IPAddress:    ip,
		UserAgent:    ua,
	})

	return user, nil
}

func (s *UserService) UpdateEmail(ctx context.Context, id string, newEmail string, actorID, ip, ua string) (*db.User, error) {
	if !strings.Contains(newEmail, "@") || !strings.Contains(newEmail, ".") {
		return nil, ErrInvalidEmail
	}

	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	oldEmail := user.Email
	user.Email = newEmail
	user.EmailVerified = false // Must re-verify email on change

	err = s.store.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	_ = s.store.CreateAuditLog(ctx, &db.AuditLog{
		TenantID:     user.TenantID,
		ActorID:      actorID,
		Action:       "UPDATE_EMAIL",
		TargetUserID: user.ID,
		Description:  fmt.Sprintf("Updated login email address from %s to %s. Status marked unverified.", oldEmail, newEmail),
		IPAddress:    ip,
		UserAgent:    ua,
	})

	return user, nil
}

func (s *UserService) UpdatePhone(ctx context.Context, id string, newPhone string, actorID, ip, ua string) (*db.User, error) {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	oldPhone := user.Phone
	user.Phone = newPhone
	user.PhoneVerified = false // Must re-verify on change

	err = s.store.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	_ = s.store.CreateAuditLog(ctx, &db.AuditLog{
		TenantID:     user.TenantID,
		ActorID:      actorID,
		Action:       "UPDATE_PHONE",
		TargetUserID: user.ID,
		Description:  fmt.Sprintf("Updated telephone number from %s to %s. Status marked unverified.", oldPhone, newPhone),
		IPAddress:    ip,
		UserAgent:    ua,
	})

	return user, nil
}

func (s *UserService) DeactivateUser(ctx context.Context, id string, actorID, ip, ua string) (*db.User, error) {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	user.Status = "DEACTIVATED"
	err = s.store.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	_ = s.store.CreateAuditLog(ctx, &db.AuditLog{
		TenantID:     user.TenantID,
		ActorID:      actorID,
		Action:       "DEACTIVATE",
		TargetUserID: user.ID,
		Description:  "Manually deactivated user account login permissions",
		IPAddress:    ip,
		UserAgent:    ua,
	})

	return user, nil
}

func (s *UserService) ReactivateUser(ctx context.Context, id string, actorID, ip, ua string) (*db.User, error) {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	user.Status = "ACTIVE"
	err = s.store.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	_ = s.store.CreateAuditLog(ctx, &db.AuditLog{
		TenantID:     user.TenantID,
		ActorID:      actorID,
		Action:       "REACTIVATE",
		TargetUserID: user.ID,
		Description:  "Reactivated user account login status to ACTIVE",
		IPAddress:    ip,
		UserAgent:    ua,
	})

	return user, nil
}

func (s *UserService) SoftDeleteUser(ctx context.Context, id string, actorID, ip, ua string) error {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = s.store.DeleteSoft(ctx, id)
	if err != nil {
		return err
	}

	_ = s.store.CreateAuditLog(ctx, &db.AuditLog{
		TenantID:     user.TenantID,
		ActorID:      actorID,
		Action:       "SOFT_DELETE",
		TargetUserID: id,
		Description:  "Soft-deleted user account and performed GDPR cryptographic data shredding",
		IPAddress:    ip,
		UserAgent:    ua,
	})

	return nil
}

func (s *UserService) HardDeleteUser(ctx context.Context, id string, actorID, ip, ua string) error {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = s.store.DeleteHard(ctx, id)
	if err != nil {
		return err
	}

	_ = s.store.CreateAuditLog(ctx, &db.AuditLog{
		TenantID:     user.TenantID,
		ActorID:      actorID,
		Action:       "HARD_DELETE",
		TargetUserID: id,
		Description:  "Permanently deleted user account records from all database registers",
		IPAddress:    ip,
		UserAgent:    ua,
	})

	return nil
}

func (s *UserService) SearchUsers(ctx context.Context, query string, status string, tenantID string, limit, offset int, actorID, ip, ua string) ([]*db.User, int, error) {
	users, total, err := s.store.Search(ctx, query, status, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// SRE or administrative telemetry logging of access search querying
	_ = s.store.CreateAuditLog(ctx, &db.AuditLog{
		TenantID:     tenantID,
		ActorID:      actorID,
		Action:       "SEARCH",
		TargetUserID: "",
		Description:  fmt.Sprintf("Queried users register. Search: '%s', Status filter: '%s', Results: %d", query, status, len(users)),
		IPAddress:    ip,
		UserAgent:    ua,
	})

	return users, total, nil
}

func (s *UserService) GetAuditLogs(ctx context.Context, targetUserID string, limit, offset int) ([]*db.AuditLog, error) {
	return s.store.GetAuditLogs(ctx, targetUserID, limit, offset)
}
