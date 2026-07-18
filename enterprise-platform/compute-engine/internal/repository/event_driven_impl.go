package repository

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
// 1. EVENTSTOREDB IMPLEMENTATION WITH OCC & SNAPSHOTTING
// -----------------------------------------------------------------

type MemoryEventStore struct {
	mu        sync.RWMutex
	streams   map[string][]domain.Event
	snapshots map[string]*domain.Snapshot
}

func NewMemoryEventStore() *MemoryEventStore {
	return &MemoryEventStore{
		streams:   make(map[string][]domain.Event),
		snapshots: make(map[string]*domain.Snapshot),
	}
}

func (s *MemoryEventStore) AppendEvents(ctx context.Context, streamID string, expectedVersion int, events []domain.Event) error {
	tr := otel.Tracer("eventstoredb-repository")
	_, span := tr.Start(ctx, "EventStore.AppendEvents")
	defer span.End()

	s.mu.Lock()
	defer s.mu.Unlock()

	slog.Info("EventStoreDB: Appending events to stream", "stream_id", streamID, "expected_version", expectedVersion)

	currentEvents := s.streams[streamID]
	currentVersion := len(currentEvents)

	// Optimistic Concurrency Control (OCC) check
	if expectedVersion != -1 && currentVersion != expectedVersion {
		slog.Error("EventStoreDB Concurrency Conflict: version mismatch",
			"stream_id", streamID,
			"current_version", currentVersion,
			"expected_version", expectedVersion,
		)
		return fmt.Errorf("concurrency conflict: expected stream version %d but got %d", expectedVersion, currentVersion)
	}

	for i := range events {
		events[i].Version = currentVersion + 1 + i
		events[i].CreatedAt = time.Now().UTC()
		s.streams[streamID] = append(s.streams[streamID], events[i])
		slog.Info("EventStoreDB: Appended event success", "stream_id", streamID, "version", events[i].Version, "type", events[i].Type)
	}

	return nil
}

func (s *MemoryEventStore) ReadStream(ctx context.Context, streamID string, fromVersion int) ([]domain.Event, error) {
	tr := otel.Tracer("eventstoredb-repository")
	_, span := tr.Start(ctx, "EventStore.ReadStream")
	defer span.End()

	s.mu.RLock()
	defer s.mu.RUnlock()

	slog.Info("EventStoreDB: Reading stream", "stream_id", streamID, "from_version", fromVersion)

	allEvents, exists := s.streams[streamID]
	if !exists {
		return []domain.Event{}, nil
	}

	var filtered []domain.Event
	for _, e := range allEvents {
		if e.Version >= fromVersion {
			filtered = append(filtered, e)
		}
	}

	return filtered, nil
}

func (s *MemoryEventStore) SaveSnapshot(ctx context.Context, snapshot domain.Snapshot) error {
	tr := otel.Tracer("eventstoredb-repository")
	_, span := tr.Start(ctx, "EventStore.SaveSnapshot")
	defer span.End()

	s.mu.Lock()
	defer s.mu.Unlock()

	slog.Info("EventStoreDB: Saving state snapshot", "stream_id", snapshot.StreamID, "version", snapshot.Version)
	snapshot.CreatedAt = time.Now().UTC()
	s.snapshots[snapshot.StreamID] = &snapshot
	return nil
}

func (s *MemoryEventStore) GetSnapshot(ctx context.Context, streamID string) (*domain.Snapshot, error) {
	tr := otel.Tracer("eventstoredb-repository")
	_, span := tr.Start(ctx, "EventStore.GetSnapshot")
	defer span.End()

	s.mu.RLock()
	defer s.mu.RUnlock()

	slog.Info("EventStoreDB: Querying snapshot", "stream_id", streamID)
	snap, exists := s.snapshots[streamID]
	if !exists {
		return nil, nil
	}
	return snap, nil
}

func (s *MemoryEventStore) ReplayEvents(ctx context.Context, streamID string, fromVersion int) ([]domain.Event, error) {
	tr := otel.Tracer("eventstoredb-repository")
	_, span := tr.Start(ctx, "EventStore.ReplayEvents")
	defer span.End()

	slog.Info("EventStoreDB: Replaying event stream stream", "stream_id", streamID, "from_version", fromVersion)
	return s.ReadStream(ctx, streamID, fromVersion)
}

// -----------------------------------------------------------------
// 2. SCHEMA REGISTRY & SCHEMA VALIDATION ENGINE
// -----------------------------------------------------------------

type MemorySchemaRegistry struct {
	mu      sync.RWMutex
	schemas map[string]string // Key: event_type_version
}

func NewMemorySchemaRegistry() *MemorySchemaRegistry {
	reg := &MemorySchemaRegistry{
		schemas: make(map[string]string),
	}

	// Bootstrap Default Versioned Schemas
	reg.schemas["TransactionCreated_1"] = `{"required":["id","amount","created_by"]}`
	reg.schemas["TransactionCreated_2"] = `{"required":["id","amount","created_by","tenant_id"]}`
	reg.schemas["TransactionProcessed_1"] = `{"required":["id","status","processed_at"]}`

	return reg
}

func (r *MemorySchemaRegistry) RegisterSchema(ctx context.Context, eventType string, version int, definition string) error {
	tr := otel.Tracer("schema-registry")
	_, span := tr.Start(ctx, "SchemaRegistry.RegisterSchema")
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s_%d", eventType, version)
	slog.Info("SchemaRegistry: Registering schema", "key", key)
	r.schemas[key] = definition
	return nil
}

func (r *MemorySchemaRegistry) ValidateEvent(ctx context.Context, eventType string, version int, data []byte) error {
	tr := otel.Tracer("schema-registry")
	_, span := tr.Start(ctx, "SchemaRegistry.ValidateEvent")
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("%s_%d", eventType, version)
	definition, exists := r.schemas[key]
	if !exists {
		slog.Warn("SchemaRegistry: Validation schema not registered, skipping default strict match validation", "key", key)
		return nil
	}

	// Simulating validation logic based on JSON tags presence
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("schema validation error: invalid JSON payload encoding: %w", err)
	}

	// Read constraints
	var constraints struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal([]byte(definition), &constraints); err == nil {
		for _, reqField := range constraints.Required {
			if _, exists := parsed[reqField]; !exists {
				slog.Error("SchemaRegistry: Validation Failed", "missing_field", reqField, "event_type", eventType)
				return fmt.Errorf("schema mismatch: field '%s' is strictly required under %s v%d specification", reqField, eventType, version)
			}
		}
	}

	slog.Info("SchemaRegistry: Schema validation passed", "key", key)
	return nil
}

// -----------------------------------------------------------------
// 3. KAFKA CLUSTER SIMULATION WITH RETRY & DEAD LETTER QUEUES
// -----------------------------------------------------------------

type MemoryKafkaCluster struct {
	mu             sync.RWMutex
	topics         map[string]domain.TopicMetadata
	consumers      map[string][]domain.ConsumerHandler // key: topic
	deadLetter     []domain.DeadLetterEvent
	retryQueue     []domain.RetryEvent
	maxRetries     int
	backoffBase    time.Duration
	registry       domain.SchemaRegistry
}

func NewMemoryKafkaCluster(reg domain.SchemaRegistry) *MemoryKafkaCluster {
	cluster := &MemoryKafkaCluster{
		topics:      make(map[string]domain.TopicMetadata),
		consumers:   make(map[string][]domain.ConsumerHandler),
		deadLetter:  make([]domain.DeadLetterEvent, 0),
		retryQueue:  make([]domain.RetryEvent, 0),
		maxRetries:  3,
		backoffBase: 100 * time.Millisecond,
		registry:    reg,
	}

	// Provision default operational topics
	_ = cluster.CreateTopic(context.Background(), "transaction-events", 3, 2)
	_ = cluster.CreateTopic(context.Background(), "transaction-events-retry", 3, 2)
	_ = cluster.CreateTopic(context.Background(), "transaction-events-dlq", 1, 3)

	// Run background engine to poll and dispatch retries with Exponential Backoff
	go cluster.startRetryLoop()

	return cluster
}

func (k *MemoryKafkaCluster) CreateTopic(ctx context.Context, name string, partitions int, replication int) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	slog.Info("KafkaCluster: Provisioning new production message topic", "topic", name, "partitions", partitions)
	k.topics[name] = domain.TopicMetadata{
		Name:              name,
		Partitions:        partitions,
		ReplicationFactor: replication,
		Config: map[string]string{
			"cleanup.policy": "compact",
			"retention.ms":   "604800000", // 7 Days retention
		},
	}
	return nil
}

func (k *MemoryKafkaCluster) DeleteTopic(ctx context.Context, name string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	slog.Info("KafkaCluster: Removing dynamic topic", "topic", name)
	delete(k.topics, name)
	return nil
}

func (k *MemoryKafkaCluster) GetTopics(ctx context.Context) ([]domain.TopicMetadata, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	list := make([]domain.TopicMetadata, 0, len(k.topics))
	for _, t := range k.topics {
		list = append(list, t)
	}
	return list, nil
}

func (k *MemoryKafkaCluster) PublishEvent(ctx context.Context, topic string, partitionKey string, event domain.Event) error {
	tr := otel.Tracer("kafka-cluster")
	ctx, span := tr.Start(ctx, "Kafka.PublishEvent")
	defer span.End()

	// 1. Strict Schema Registry check
	err := k.registry.ValidateEvent(ctx, event.Type, event.Version, event.Data)
	if err != nil {
		slog.Error("KafkaCluster Publish Blocked: Schema registry validation failed", "type", event.Type, "error", err)
		return fmt.Errorf("schema validation exception: %w", err)
	}

	k.mu.RLock()
	_, topicExists := k.topics[topic]
	handlers := k.consumers[topic]
	k.mu.RUnlock()

	if !topicExists {
		return fmt.Errorf("kafka topic exception: topic '%s' does not exist in cluster metadata", topic)
	}

	slog.Info("KafkaCluster: Dispatched payload packet", "topic", topic, "key", partitionKey, "type", event.Type)

	// Dispatch to active subscriber handlers (Simulated consumer processing)
	for _, handler := range handlers {
		go func(h domain.ConsumerHandler, e domain.Event) {
			hCtx := context.Background()
			err := h.Handle(hCtx, e)
			if err != nil {
				slog.Warn("KafkaConsumer: Handler failed to consume message. Routing to Retry engine pipeline...",
					"topic", topic,
					"event_id", e.ID,
					"error", err,
				)
				k.routeToRetry(e, topic, err.Error())
			}
		}(handler, event)
	}

	return nil
}

func (k *MemoryKafkaCluster) Subscribe(ctx context.Context, topic string, consumerGroupID string, handler domain.ConsumerHandler) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	slog.Info("KafkaCluster: New worker subscribed to topic stream", "topic", topic, "consumer_group", consumerGroupID)
	k.consumers[topic] = append(k.consumers[topic], handler)
	return nil
}

// -----------------------------------------------------------------
// 4. DETAILED RETRY & DLQ RESILIENCE ENGINE
// -----------------------------------------------------------------

func (k *MemoryKafkaCluster) routeToRetry(event domain.Event, topic string, lastError string) {
	k.mu.Lock()
	defer k.mu.Unlock()

	retryCount := 1
	nextRetryDelay := k.backoffBase // Exponential Base

	// Check if this event is already in our retry processing pipelines
	for i, r := range k.retryQueue {
		if r.Event.ID == event.ID {
			retryCount = r.RetryCount + 1
			// Exponential formula: delay = Base * 2^(retryCount - 1)
			for factor := 1; factor < retryCount; factor++ {
				nextRetryDelay *= 2
			}
			// Delete existing reference
			k.retryQueue = append(k.retryQueue[:i], k.retryQueue[i+1:]...)
			break
		}
	}

	if retryCount > k.maxRetries {
		slog.Error("KafkaCluster Max Retries Exhausted! Offloading event package to DLQ",
			"event_id", event.ID,
			"topic", topic,
			"retry_count", retryCount,
		)
		dlqEvent := domain.DeadLetterEvent{
			ID:        fmt.Sprintf("dlq_%d", time.Now().UnixNano()),
			Topic:     topic,
			Partition: 0,
			Event:     event,
			Reason:    fmt.Sprintf("max retries exhausted: %s", lastError),
			FailedAt:  time.Now().UTC(),
		}
		k.deadLetter = append(k.deadLetter, dlqEvent)
		return
	}

	nextRetryAt := time.Now().Add(nextRetryDelay)
	slog.Info("KafkaCluster Retry Scheduled",
		"event_id", event.ID,
		"retry_attempt", retryCount,
		"delay", nextRetryDelay.String(),
		"scheduled_at", nextRetryAt.Format(time.RFC3339),
	)

	k.retryQueue = append(k.retryQueue, domain.RetryEvent{
		Event:       event,
		Topic:       topic,
		RetryCount:  retryCount,
		NextRetryAt: nextRetryAt,
		LastError:   lastError,
	})
}

func (k *MemoryKafkaCluster) startRetryLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	for range ticker.C {
		k.mu.Lock()
		now := time.Now()
		var readyEvents []domain.RetryEvent
		var remainingEvents []domain.RetryEvent

		for _, r := range k.retryQueue {
			if now.After(r.NextRetryAt) {
				readyEvents = append(readyEvents, r)
			} else {
				remainingEvents = append(remainingEvents, r)
			}
		}
		k.retryQueue = remainingEvents
		k.mu.Unlock()

		// Execute ready retries
		for _, re := range readyEvents {
			slog.Info("KafkaCluster: Triggering scheduled retry event execute", "event_id", re.Event.ID, "attempt", re.RetryCount)
			
			k.mu.RLock()
			handlers := k.consumers[re.Topic]
			k.mu.RUnlock()

			success := false
			for _, h := range handlers {
				err := h.Handle(context.Background(), re.Event)
				if err == nil {
					success = true
					slog.Info("KafkaCluster: Retry execution SUCCEEDED", "event_id", re.Event.ID)
					break
				} else {
					slog.Warn("KafkaCluster: Retry execution failed again", "event_id", re.Event.ID, "error", err.Error())
				}
			}

			if !success {
				// Re-schedule retry
				k.routeToRetry(re.Event, re.Topic, "consecutive retry failure")
			}
		}
	}
}

func (k *MemoryKafkaCluster) GetDLQEvents() []domain.DeadLetterEvent {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.deadLetter
}

func (k *MemoryKafkaCluster) GetRetryQueueEvents() []domain.RetryEvent {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.retryQueue
}
