package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/nexuscore/compute-engine/internal/domain"
	"go.opentelemetry.io/otel"
)

type computeService struct {
	eventDriven *EventDrivenService
}

func NewComputeService(
	eventStore domain.EventStore,
	kafka domain.KafkaCluster,
	queryStore *MemoryQueryStore,
) *computeService {
	return &computeService{
		eventDriven: NewEventDrivenService(eventStore, kafka, queryStore),
	}
}

func (s *computeService) ProcessTransaction(ctx context.Context, tx domain.Transaction) (*domain.Transaction, error) {
	tr := otel.Tracer("compute-usecase")
	ctx, span := tr.Start(ctx, "Usecase.ProcessTransaction")
	defer span.End()

	slog.Info("Usecase: Intercepted process request. Dispatching CQRS CreateTransactionCommand", "id", tx.ID, "amount", tx.Amount)

	cmd := domain.CreateTransactionCommand{
		ID:        tx.ID,
		Amount:    tx.Amount,
		CreatedBy: tx.CreatedBy,
	}

	err := s.eventDriven.ExecuteCreateTransaction(ctx, cmd)
	if err != nil {
		return nil, err
	}

	// Wait briefly for projection or read directly from our separation query projection model
	time.Sleep(5 * time.Millisecond) // Simulated brief projection settlement

	qm, err := s.eventDriven.QueryGetTransaction(ctx, domain.GetTransactionQuery{ID: tx.ID})
	if err != nil {
		// Fallback to direct mapping if projection is slow
		return &domain.Transaction{
			ID:        tx.ID,
			Amount:    tx.Amount,
			Status:    "PENDING",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			CreatedBy: tx.CreatedBy,
		}, nil
	}

	return &domain.Transaction{
		ID:        qm.ID,
		Amount:    qm.Amount,
		Status:    qm.Status,
		CreatedAt: qm.UpdatedAt.Format(time.RFC3339),
		CreatedBy: tx.CreatedBy,
	}, nil
}

func (s *computeService) GetTransaction(ctx context.Context, id string) (*domain.Transaction, error) {
	tr := otel.Tracer("compute-usecase")
	ctx, span := tr.Start(ctx, "Usecase.GetTransaction")
	defer span.End()

	slog.Info("Usecase: Querying transactional read projection model database", "id", id)

	qm, err := s.eventDriven.QueryGetTransaction(ctx, domain.GetTransactionQuery{ID: id})
	if err != nil {
		return nil, err
	}

	return &domain.Transaction{
		ID:        qm.ID,
		Amount:    qm.Amount,
		Status:    qm.Status,
		CreatedAt: qm.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *computeService) ListTransactions(ctx context.Context) ([]*domain.Transaction, error) {
	tr := otel.Tracer("compute-usecase")
	ctx, span := tr.Start(ctx, "Usecase.ListTransactions")
	defer span.End()

	slog.Info("Usecase: Listing transactional records from read projection model database")

	list, err := s.eventDriven.QueryListTransactions(ctx, GetTransactionsQueryWithParams())
	if err != nil {
		return nil, err
	}

	var txs []*domain.Transaction
	for _, qm := range list {
		txs = append(txs, &domain.Transaction{
			ID:        qm.ID,
			Amount:    qm.Amount,
			Status:    qm.Status,
			CreatedAt: qm.UpdatedAt.Format(time.RFC3339),
		})
	}

	return txs, nil
}

// Helper getter to expose EventDrivenService to the HTTP delivery routing layers
func (s *computeService) GetEventDrivenService() *EventDrivenService {
	return s.eventDriven
}

// Inject helpers into domain queries
type extListQuery struct {
	Limit  int
	Offset int
}

func (e extListQuery) GetLimit() int {
	return e.Limit
}

// Helper default bounds
func GetTransactionsQueryWithParams() domain.ListTransactionsQuery {
	return domain.ListTransactionsQuery{
		Limit:  100,
		Offset: 0,
	}
}
