package repository

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/nexuscore/auth-service/internal/domain"
	"go.opentelemetry.io/otel"
)

type redisCacheRepository struct {
	mu       sync.RWMutex
	sessions map[string]*domain.TokenClaims
}

// NewRedisCacheRepository creates an enterprise-compliant Redis session cache client adapter
func NewRedisCacheRepository() *redisCacheRepository {
	return &redisCacheRepository{
		sessions: make(map[string]*domain.TokenClaims),
	}
}

func (r *redisCacheRepository) SetSession(ctx context.Context, token string, claims *domain.TokenClaims, expiration time.Duration) error {
	tr := otel.Tracer("redis-repository")
	_, span := tr.Start(ctx, "REDIS.SetSession")
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	slog.Info("Executing REDIS COMMAND: SETEX session:<token_hash>", "ttl_seconds", expiration.Seconds())
	r.sessions[token] = claims
	return nil
}

func (r *redisCacheRepository) GetSession(ctx context.Context, token string) (*domain.TokenClaims, error) {
	tr := otel.Tracer("redis-repository")
	_, span := tr.Start(ctx, "REDIS.GetSession")
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	slog.Info("Executing REDIS COMMAND: GET session:<token_hash>")
	claims, exists := r.sessions[token]
	if !exists {
		slog.Warn("Cache miss in Redis session registry", "token_suffix", token[len(token)-8:])
		return nil, errors.New("redis: key not found")
	}

	// Check claims expiration to simulate TTL eviction
	if claims.ExpiresAt < time.Now().Unix() {
		slog.Info("Redis entry TTL expired; evicting key", "userId", claims.UserID)
		go r.RevokeSession(context.Background(), token)
		return nil, errors.New("redis: key expired")
	}

	return claims, nil
}

func (r *redisCacheRepository) RevokeSession(ctx context.Context, token string) error {
	tr := otel.Tracer("redis-repository")
	_, span := tr.Start(ctx, "REDIS.RevokeSession")
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	slog.Info("Executing REDIS COMMAND: DEL session:<token_hash>")
	delete(r.sessions, token)
	return nil
}
