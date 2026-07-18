package domain

import (
	"context"
	"time"
)

// -----------------------------------------------------------------
// 1. EVENT SOURCE & EVENTSTOREDB ENTITIES
// -----------------------------------------------------------------

type Event struct {
	ID        string    `json:"id"`
	StreamID  string    `json:"stream_id"`
	Type      string    `json:"type"`
	Version   int       `json:"version"` // Event Versioning
	Data      []byte    `json:"data"`    // Serialized event payload
	CreatedAt time.Time `json:"created_at"`
}

type Snapshot struct {
	StreamID  string    `json:"stream_id"`
	Version   int       `json:"version"` // Aggregate version at snapshot
	State     []byte    `json:"state"`   // Serialized snapshot state
	CreatedAt time.Time `json:"created_at"`
}

type EventStore interface {
	AppendEvents(ctx context.Context, streamID string, expectedVersion int, events []Event) error
	ReadStream(ctx context.Context, streamID string, fromVersion int) ([]Event, error)
	SaveSnapshot(ctx context.Context, snapshot Snapshot) error
	GetSnapshot(ctx context.Context, streamID string) (*Snapshot, error)
	ReplayEvents(ctx context.Context, streamID string, fromVersion int) ([]Event, error)
}

// -----------------------------------------------------------------
// 2. SCHEMA REGISTRY & EVENT VERSIONING
// -----------------------------------------------------------------

type Schema struct {
	EventType string `json:"event_type"`
	Version   int    `json:"version"`
	Definition string `json:"definition"` // JSON Schema or validation rules
}

type SchemaRegistry interface {
	RegisterSchema(ctx context.Context, eventType string, version int, definition string) error
	ValidateEvent(ctx context.Context, eventType string, version int, data []byte) error
}

// -----------------------------------------------------------------
// 3. KAFKA CLUSTER & TOPIC MANAGEMENT
// -----------------------------------------------------------------

type TopicMetadata struct {
	Name              string `json:"name"`
	Partitions        int    `json:"partitions"`
	ReplicationFactor int    `json:"replication_factor"`
	Config            map[string]string `json:"config"`
}

type KafkaCluster interface {
	CreateTopic(ctx context.Context, name string, partitions int, replication int) error
	DeleteTopic(ctx context.Context, name string) error
	GetTopics(ctx context.Context) ([]TopicMetadata, error)
	PublishEvent(ctx context.Context, topic string, partitionKey string, event Event) error
	Subscribe(ctx context.Context, topic string, consumerGroupID string, handler ConsumerHandler) error
}

// -----------------------------------------------------------------
// 4. RETRY QUEUE & DEAD LETTER QUEUE (DLQ)
// -----------------------------------------------------------------

type DeadLetterEvent struct {
	ID          string    `json:"id"`
	Topic       string    `json:"topic"`
	Partition   int       `json:"partition"`
	Event       Event     `json:"event"`
	Reason      string    `json:"reason"`
	FailedAt    time.Time `json:"failed_at"`
}

type RetryEvent struct {
	Event       Event     `json:"event"`
	Topic       string    `json:"topic"`
	RetryCount  int       `json:"retry_count"`
	NextRetryAt time.Time `json:"next_retry_at"`
	LastError   string    `json:"last_error"`
}

type ConsumerHandler interface {
	Handle(ctx context.Context, event Event) error
}

// -----------------------------------------------------------------
// 5. CQRS (COMMANDS & QUERIES) DEFINITIONS
// -----------------------------------------------------------------

// Commands
type CreateTransactionCommand struct {
	ID        string  `json:"id"`
	Amount    float64 `json:"amount"`
	CreatedBy string  `json:"created_by"`
}

type ReplayEventsCommand struct {
	StreamID    string `json:"stream_id"`
	FromVersion int    `json:"from_version"`
}

type TakeSnapshotCommand struct {
	StreamID string `json:"stream_id"`
}

// Queries
type GetTransactionQuery struct {
	ID string `json:"id"`
}

type ListTransactionsQuery struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type QueryModel struct {
	ID        string    `json:"id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}
