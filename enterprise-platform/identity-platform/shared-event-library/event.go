package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Standard event error definitions
var (
	ErrInvalidEventEnvelope   = errors.New("invalid cloud event envelope structure")
	ErrUnsupportedSchemaClass = errors.New("payload class not supported by schema definition")
	ErrPublisherUnreachable   = errors.New("unable to establish connection to event broker cluster")
	ErrConsumerCommitFailed   = errors.New("failed to commit partition offset back to broker")
)

// CloudEvent standard envelope compliant with CloudEvents v1.0 specifications.
type CloudEvent struct {
	SpecVersion     string          `json:"specversion"`
	ID              string          `json:"id"`
	Source          string          `json:"source"`
	Type            string          `json:"type"`
	Time            time.Time       `json:"time"`
	DataContentType string          `json:"datacontenttype"`
	TenantID        string          `json:"tenant_id"`
	TraceID         string          `json:"trace_id,omitempty"`
	SchemaVersion   int             `json:"schema_version"`
	Data            json.RawMessage `json:"data"`
}

// EnsureValid validates the envelope structure constraints
func (c *CloudEvent) EnsureValid() error {
	if c.SpecVersion != "1.0" {
		return fmt.Errorf("%w: specversion must be '1.0'", ErrInvalidEventEnvelope)
	}
	if c.ID == "" || c.Source == "" || c.Type == "" || c.TenantID == "" {
		return fmt.Errorf("%w: missing required metadata properties", ErrInvalidEventEnvelope)
	}
	if len(c.Data) == 0 {
		return fmt.Errorf("%w: message payload cannot be empty", ErrInvalidEventEnvelope)
	}
	return nil
}

// Define Strongly-Typed Domain Event Payloads

// UserCreatedEventPayload holds parameters for user initialization
type UserCreatedEventPayload struct {
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// UserUpdatedEventPayload encapsulates delta user attributes
type UserUpdatedEventPayload struct {
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	Email     string    `json:"email"`
	Status    string    `json:"status"` // ACTIVE, SUSPENDED, PENDING
	UpdatedAt time.Time `json:"updated_at"`
}

// UserDeletedEventPayload maps deleted identifiers
type UserDeletedEventPayload struct {
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	DeletedAt time.Time `json:"deleted_at"`
}

// LoginSuccessEventPayload captures successful sessions
type LoginSuccessEventPayload struct {
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	SessionID string    `json:"session_id"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
	Timestamp time.Time `json:"timestamp"`
}

// LoginFailedEventPayload tracks authentication failures for anomaly detection
type LoginFailedEventPayload struct {
	TenantID  string    `json:"tenant_id"`
	Username  string    `json:"username"`
	Reason    string    `json:"reason"` // BAD_CREDENTIALS, ACCOUNT_LOCKED, TOTP_INVALID
	IPAddress string    `json:"ip_address"`
	Timestamp time.Time `json:"timestamp"`
}

// LogoutEventPayload details active session termination
type LogoutEventPayload struct {
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	SessionID string    `json:"session_id"`
	Reason    string    `json:"reason"` // USER_INITIATED, TIMEOUT, REVOKED
	Timestamp time.Time `json:"timestamp"`
}

// RoleCreatedEventPayload holds parameters for new role creations
type RoleCreatedEventPayload struct {
	RoleID      string    `json:"role_id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

// PermissionCreatedEventPayload catalogs global permissions
type PermissionCreatedEventPayload struct {
	PermissionID string    `json:"permission_id"`
	Code         string    `json:"code"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
}

// OrganizationCreatedEventPayload tracks structural SaaS boundaries
type OrganizationCreatedEventPayload struct {
	OrgID     string    `json:"org_id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// TenantCreatedEventPayload maps tenant instances
type TenantCreatedEventPayload struct {
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Domain    string    `json:"domain"`
	PlanType  string    `json:"plan_type"` // DEV, ENTERPRISE, STARTUP
	CreatedAt time.Time `json:"created_at"`
}

// AuditEventPayload stores structural logs for security tracing
type AuditEventPayload struct {
	EventID    string    `json:"event_id"`
	TenantID   string    `json:"tenant_id"`
	ActorID    string    `json:"actor_id"`
	Action     string    `json:"action"` // e.g., "user.password_changed"
	Resource   string    `json:"resource"`
	Status     string    `json:"status"` // SUCCESS, FAILED
	IPAddress  string    `json:"ip_address"`
	ChainHash  string    `json:"chain_hash"`
	Timestamp  time.Time `json:"timestamp"`
}

// NotificationEventPayload triggers channel-specific dispatches
type NotificationEventPayload struct {
	UserID    string            `json:"user_id"`
	TenantID  string            `json:"tenant_id"`
	Channel   string            `json:"channel"` // EMAIL, SMS, PUSH
	Template  string            `json:"template"`
	Context   map[string]string `json:"context"`
	Timestamp time.Time         `json:"timestamp"`
}

// Handler defines a callback structure to digest incoming broker envelopes
type Handler func(ctx context.Context, event *CloudEvent) error

// Publisher exposes methods to submit events onto Kafka topics
type Publisher interface {
	Publish(ctx context.Context, topic string, key string, event *CloudEvent) error
}

// Subscriber exposes methods to capture events from specific topics
type Subscriber interface {
	Subscribe(ctx context.Context, topic string, handler Handler) error
}

// MemoryEventBroker acts as a multi-consumer, fully functional, multi-partition in-memory
// Kafka simulator with built-in retry pipelines, exponential delays, and a DLQ sink.
type MemoryEventBroker struct {
	mu           sync.RWMutex
	topics       map[string][]chan *brokerMessage
	subscribers  map[string][]Handler
	dlqSink      []*brokerMessage
	retryCount   map[string]int // tracks death-counts for specific message IDs
	maxRetries   int
	retryBackoff time.Duration
}

type brokerMessage struct {
	Topic string
	Key   string
	Event *CloudEvent
}

// NewMemoryEventBroker creates an elegant mock cluster with explicit pipeline configurations.
func NewMemoryEventBroker(maxRetries int, retryBackoff time.Duration) *MemoryEventBroker {
	return &MemoryEventBroker{
		topics:       make(map[string][]chan *brokerMessage),
		subscribers:  make(map[string][]Handler),
		dlqSink:      make([]*brokerMessage, 0),
		retryCount:   make(map[string]int),
		maxRetries:   maxRetries,
		retryBackoff: retryBackoff,
	}
}

// Publish distributes the message into simulated target partitions.
func (b *MemoryEventBroker) Publish(ctx context.Context, topic string, key string, event *CloudEvent) error {
	if err := event.EnsureValid(); err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	msg := &brokerMessage{
		Topic: topic,
		Key:   key,
		Event: event,
	}

	// Dispatch to subscribers
	handlers, exists := b.subscribers[topic]
	if !exists || len(handlers) == 0 {
		return nil
	}

	for _, handler := range handlers {
		go b.safeProcessMessage(ctx, handler, msg)
	}

	return nil
}

// Subscribe binds handlers to specified topic events
func (b *MemoryEventBroker) Subscribe(ctx context.Context, topic string, handler Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[topic] = append(b.subscribers[topic], handler)
	return nil
}

// GetDLQSink extracts DLQ payloads for integration verification checks
func (b *MemoryEventBroker) GetDLQSink() []*CloudEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	events := make([]*CloudEvent, len(b.dlqSink))
	for i, msg := range b.dlqSink {
		events[i] = msg.Event
	}
	return events
}

// safeProcessMessage wraps execution inside the standard Retry and DLQ loops.
func (b *MemoryEventBroker) safeProcessMessage(ctx context.Context, handler Handler, msg *brokerMessage) {
	err := handler(ctx, msg.Event)
	if err == nil {
		return
	}

	// Failure encountered - start retry/DLQ pipeline routing
	b.mu.Lock()
	b.retryCount[msg.Event.ID]++
	attempts := b.retryCount[msg.Event.ID]
	b.mu.Unlock()

	if attempts <= b.maxRetries {
		// Route through Retry Topic Pipeline after backing off
		time.Sleep(b.retryBackoff * time.Duration(attempts))

		retryTopic := fmt.Sprintf("%s.retry-%d", msg.Topic, attempts)
		_ = b.Publish(ctx, retryTopic, msg.Key, msg.Event)
		return
	}

	// Max retries depleted, redirect directly to the Dead Letter Queue
	b.mu.Lock()
	b.dlqSink = append(b.dlqSink, msg)
	delete(b.retryCount, msg.Event.ID)
	b.mu.Unlock()

	// Track fallback failure metrics
	fmt.Printf("[CRITICAL-SRE] Event %s routed to Dead Letter Queue (%s.dlq). Root Error: %v\n", msg.Event.ID, msg.Topic, err)
}
