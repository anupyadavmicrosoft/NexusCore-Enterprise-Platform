package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nexuscore/compute-engine/internal/domain"
	"go.opentelemetry.io/otel"
)

// -----------------------------------------------------------------
// 1. CQRS SEPARATE READ MODEL STORAGE DEFINITIONS
// -----------------------------------------------------------------

type MemoryQueryStore struct {
	mu     sync.RWMutex
	models map[string]*domain.QueryModel
}

func NewMemoryQueryStore() *MemoryQueryStore {
	store := &MemoryQueryStore{
		models: make(map[string]*domain.QueryModel),
	}

	// Seed some historical projections for instant rendering
	store.models["tx_9921"] = &domain.QueryModel{
		ID:        "tx_9921",
		Amount:    15200.50,
		Status:    "APPROVED",
		Version:   2,
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	store.models["tx_9922"] = &domain.QueryModel{
		ID:        "tx_9922",
		Amount:    89.90,
		Status:    "APPROVED",
		Version:   2,
		UpdatedAt: time.Now().Add(-15 * time.Minute),
	}

	return store
}

func (s *MemoryQueryStore) Save(model *domain.QueryModel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.models[model.ID] = model
}

func (s *MemoryQueryStore) FindByID(id string) (*domain.QueryModel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, exists := s.models[id]
	if !exists {
		return nil, errors.New("read-model: transaction query entity missing")
	}
	return m, nil
}

func (s *MemoryQueryStore) FindAll(limit, offset int) []*domain.QueryModel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*domain.QueryModel
	for _, m := range s.models {
		list = append(list, m)
	}

	// Simple pagination bounds
	if offset >= len(list) {
		return []*domain.QueryModel{}
	}
	end := offset + limit
	if end > len(list) {
		end = len(list)
	}

	return list[offset:end]
}

func (s *MemoryQueryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.models = make(map[string]*domain.QueryModel)
}

// -----------------------------------------------------------------
// 2. MAIN CQRS COMMAND HANDLER SERVICE
// -----------------------------------------------------------------

type EventDrivenService struct {
	eventStore domain.EventStore
	kafka      domain.KafkaCluster
	queryStore *MemoryQueryStore
}

func NewEventDrivenService(
	es domain.EventStore,
	kafka domain.KafkaCluster,
	qs *MemoryQueryStore,
) *EventDrivenService {
	svc := &EventDrivenService{
		eventStore: es,
		kafka:      kafka,
		queryStore: qs,
	}

	// Register projection updater as Kafka Subscriber to handle async updates
	_ = kafka.Subscribe(context.Background(), "transaction-events", "compute-projection-group", svc)

	return svc
}

// Handle implements domain.ConsumerHandler interface to process incoming Kafka stream packets
func (s *EventDrivenService) Handle(ctx context.Context, event domain.Event) error {
	tr := otel.Tracer("compute-consumer")
	_, span := tr.Start(ctx, "KafkaConsumer.ProcessEvent: "+event.Type)
	defer span.End()

	slog.Info("KafkaConsumer: Received message for read model updating", "event_id", event.ID, "type", event.Type, "version", event.Version)

	switch event.Type {
	case "TransactionCreated":
		var payload domain.CreateTransactionCommand
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return err
		}

		// Update or Insert read projection model
		model := &domain.QueryModel{
			ID:        payload.ID,
			Amount:    payload.Amount,
			Status:    "PENDING",
			Version:   event.Version,
			UpdatedAt: event.CreatedAt,
		}
		s.queryStore.Save(model)
		slog.Info("Projection Database updated: Transaction status marked PENDING", "id", payload.ID)

	case "TransactionProcessed":
		var payload struct {
			ID        string  `json:"id"`
			Amount    float64 `json:"amount"`
			Status    string  `json:"status"`
			CreatedBy string  `json:"created_by"`
		}
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return err
		}

		// Ensure we don't apply an out of order state update (version check)
		existing, err := s.queryStore.FindByID(payload.ID)
		if err == nil && existing.Version > event.Version {
			slog.Warn("Projection Engine: Skipping obsolete status event version", "current", existing.Version, "event", event.Version)
			return nil
		}

		model := &domain.QueryModel{
			ID:        payload.ID,
			Amount:    payload.Amount,
			Status:    payload.Status,
			Version:   event.Version,
			UpdatedAt: event.CreatedAt,
		}
		s.queryStore.Save(model)
		slog.Info("Projection Database updated: Transaction state resolved", "id", payload.ID, "status", payload.Status)
	}

	return nil
}

// -----------------------------------------------------------------
// COMMAND EXECUTION LOGIC
// -----------------------------------------------------------------

func (s *EventDrivenService) ExecuteCreateTransaction(ctx context.Context, cmd domain.CreateTransactionCommand) error {
	tr := otel.Tracer("cqrs-commands")
	ctx, span := tr.Start(ctx, "Command.CreateTransaction")
	defer span.End()

	if cmd.ID == "" {
		return errors.New("command validation error: transaction ID cannot be empty")
	}
	if cmd.Amount <= 0 {
		return errors.New("command validation error: transaction amount must be positive")
	}

	streamID := fmt.Sprintf("transaction-stream-%s", cmd.ID)

	// Verify if stream already exists
	existingEvents, _ := s.eventStore.ReadStream(ctx, streamID, 1)
	if len(existingEvents) > 0 {
		return fmt.Errorf("aggregate stream constraint: stream for transaction ID %s already instantiated", cmd.ID)
	}

	// 1. Map to Domain Events (TransactionCreated)
	dataCreated, _ := json.Marshal(cmd)
	eventCreated := domain.Event{
		ID:        fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		StreamID:  streamID,
		Type:      "TransactionCreated",
		Data:      dataCreated,
		CreatedAt: time.Now().UTC(),
	}

	// 2. Process Business rule calculations (e.g. anti-fraud status evaluation)
	status := "APPROVED"
	if cmd.Amount > 1000000 {
		status = "REJECTED"
		slog.Warn("Risk limit threshold violated! Creating default REJECTED state event", "id", cmd.ID)
	}

	processedPayload := struct {
		ID        string  `json:"id"`
		Amount    float64 `json:"amount"`
		Status    string  `json:"status"`
		CreatedBy string  `json:"created_by"`
	}{
		ID:        cmd.ID,
		Amount:    cmd.Amount,
		Status:    status,
		CreatedBy: cmd.CreatedBy,
	}
	dataProcessed, _ := json.Marshal(processedPayload)
	eventProcessed := domain.Event{
		ID:        fmt.Sprintf("evt_%d", time.Now().UnixNano()+10),
		StreamID:  streamID,
		Type:      "TransactionProcessed",
		Data:      dataProcessed,
		CreatedAt: time.Now().UTC(),
	}

	// 3. Append to immutable EventStoreDB (OCC verify: expected stream version = 0)
	eventsToAppend := []domain.Event{eventCreated, eventProcessed}
	err := s.eventStore.AppendEvents(ctx, streamID, 0, eventsToAppend)
	if err != nil {
		return err
	}

	// 4. Publish to external messaging broker (Kafka Topic: "transaction-events")
	for _, evt := range eventsToAppend {
		err = s.kafka.PublishEvent(ctx, "transaction-events", cmd.ID, evt)
		if err != nil {
			slog.Error("Kafka dispatch failed: streaming fallback active", "error", err)
		}
	}

	return nil
}

// -----------------------------------------------------------------
// EVENT REPLAY & SNAPSHOTTING BUSINESS LOGIC
// -----------------------------------------------------------------

func (s *EventDrivenService) ExecuteReplay(ctx context.Context, cmd domain.ReplayEventsCommand) error {
	tr := otel.Tracer("cqrs-commands")
	ctx, span := tr.Start(ctx, "Command.ReplayEvents")
	defer span.End()

	slog.Info("Executing Command: ReplayEvents", "stream_id", cmd.StreamID, "from_version", cmd.FromVersion)

	// Fetch historical events starting from chosen version
	events, err := s.eventStore.ReplayEvents(ctx, cmd.StreamID, cmd.FromVersion)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return errors.New("event replay exception: no events matching search parameters")
	}

	// Temporarily clear or update matching projection cache
	slog.Info("Projection Rebuilding active...", "events_reconstructed", len(events))
	for _, event := range events {
		_ = s.Handle(ctx, event)
	}

	return nil
}

func (s *EventDrivenService) ExecuteSnapshot(ctx context.Context, cmd domain.TakeSnapshotCommand) error {
	tr := otel.Tracer("cqrs-commands")
	ctx, span := tr.Start(ctx, "Command.Snapshot")
	defer span.End()

	slog.Info("Executing Command: TakeSnapshot", "stream_id", cmd.StreamID)

	// Fetch all events to construct current aggregated state
	events, err := s.eventStore.ReadStream(ctx, cmd.StreamID, 1)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return errors.New("snapshot creation exception: no events in target stream")
	}

	latestVersion := events[len(events)-1].Version

	// Derive consolidated model state
	var currentStatus string
	var amount float64
	for _, e := range events {
		if e.Type == "TransactionProcessed" {
			var payload struct {
				Status string  `json:"status"`
				Amount float64 `json:"amount"`
			}
			_ = json.Unmarshal(e.Data, &payload)
			currentStatus = payload.Status
			amount = payload.Amount
		}
	}

	statePayload := struct {
		StreamID  string    `json:"stream_id"`
		Version   int       `json:"version"`
		Amount    float64   `json:"amount"`
		Status    string    `json:"status"`
		UpdatedAt time.Time `json:"updated_at"`
	}{
		StreamID:  cmd.StreamID,
		Version:   latestVersion,
		Amount:    amount,
		Status:    currentStatus,
		UpdatedAt: time.Now().UTC(),
	}

	bytes, _ := json.Marshal(statePayload)

	snapshot := domain.Snapshot{
		StreamID:  cmd.StreamID,
		Version:   latestVersion,
		State:     bytes,
		CreatedAt: time.Now().UTC(),
	}

	return s.eventStore.SaveSnapshot(ctx, snapshot)
}

// -----------------------------------------------------------------
// QUERY IMPLEMENTATION
// -----------------------------------------------------------------

func (s *EventDrivenService) QueryGetTransaction(ctx context.Context, query domain.GetTransactionQuery) (*domain.QueryModel, error) {
	tr := otel.Tracer("cqrs-queries")
	_, span := tr.Start(ctx, "Query.GetTransaction")
	defer span.End()

	return s.queryStore.FindByID(query.ID)
}

func (s *EventDrivenService) QueryListTransactions(ctx context.Context, query domain.ListTransactionsQuery) ([]*domain.QueryModel, error) {
	tr := otel.Tracer("cqrs-queries")
	_, span := tr.Start(ctx, "Query.ListTransactions")
	defer span.End()

	return s.queryStore.FindAll(query.Limit, query.Offset), nil
}
