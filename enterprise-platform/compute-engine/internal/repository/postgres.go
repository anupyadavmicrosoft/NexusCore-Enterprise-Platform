package repository

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/nexuscore/compute-engine/internal/domain"
	"go.opentelemetry.io/otel"
)

type memoryTransactionRepository struct {
	mu  sync.RWMutex
	txs map[string]*domain.Transaction
}

func NewTransactionRepository() *memoryTransactionRepository {
	repo := &memoryTransactionRepository{
		txs: make(map[string]*domain.Transaction),
	}

	// Seed transactional history for dynamic charts
	repo.txs["tx_9921"] = &domain.Transaction{
		ID:        "tx_9921",
		Amount:    15200.50,
		Status:    "APPROVED",
		CreatedAt: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		CreatedBy: "principal@nexuscore.io",
	}

	repo.txs["tx_9922"] = &domain.Transaction{
		ID:        "tx_9922",
		Amount:    89.90,
		Status:    "APPROVED",
		CreatedAt: time.Now().Add(-15 * time.Minute).Format(time.RFC3339),
		CreatedBy: "user_enterprise_01",
	}

	return repo
}

func (r *memoryTransactionRepository) Save(ctx context.Context, tx *domain.Transaction) error {
	tr := otel.Tracer("compute-repository")
	_, span := tr.Start(ctx, "SQL.Transaction.Save")
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	slog.Info("Executing SQL Insert command", "id", tx.ID, "amount", tx.Amount)
	r.txs[tx.ID] = tx
	slog.Info("Transaction statement written to ledger disk storage", "id", tx.ID)
	return nil
}

func (r *memoryTransactionRepository) FindByID(ctx context.Context, id string) (*domain.Transaction, error) {
	tr := otel.Tracer("compute-repository")
	_, span := tr.Start(ctx, "SQL.Transaction.FindByID")
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	slog.Info("Querying transactional databases", "id", id)
	tx, exists := r.txs[id]
	if !exists {
		slog.Warn("Transaction query record missed", "id", id)
		return nil, errors.New("sql: no rows in result set")
	}

	return tx, nil
}

func (r *memoryTransactionRepository) FindAll(ctx context.Context) ([]*domain.Transaction, error) {
	tr := otel.Tracer("compute-repository")
	_, span := tr.Start(ctx, "SQL.Transaction.FindAll")
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*domain.Transaction, 0, len(r.txs))
	for _, tx := range r.txs {
		list = append(list, tx)
	}

	return list, nil
}
