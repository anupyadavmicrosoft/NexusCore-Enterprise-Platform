package domain

import "context"

// Transaction represents a financial state-backed transaction inside Compute Engine
type Transaction struct {
	ID        string  `json:"id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"` // PENDING, APPROVED, REJECTED
	CreatedAt string  `json:"created_at"`
	CreatedBy string  `json:"created_by"`
}

// TransactionRepository represents infrastructure persistence ports
type TransactionRepository interface {
	Save(ctx context.Context, tx *Transaction) error
	FindByID(ctx context.Context, id string) (*Transaction, error)
	FindAll(ctx context.Context) ([]*Transaction, error)
}

// ComputeUsecase describes CQRS operations on transactional entities
type ComputeUsecase interface {
	ProcessTransaction(ctx context.Context, tx Transaction) (*Transaction, error)
	GetTransaction(ctx context.Context, id string) (*Transaction, error)
	ListTransactions(ctx context.Context) ([]*Transaction, error)
}
