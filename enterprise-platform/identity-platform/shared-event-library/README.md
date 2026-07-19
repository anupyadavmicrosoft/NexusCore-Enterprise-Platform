# NexusCore Shared Event Library

An enterprise-grade, high-performance event-driven library modeling strongly typed CloudEvents, retry mechanisms, dead letter queue (DLQ) topology, and resilient pub/sub contracts for Apache Kafka.

## Features

- **CloudEvents v1.0 Envelope Spec Compliance**: Ensures universal wrapping, metadata extraction, and multi-tenant tracing vector persistence.
- **Strongly-Typed Domain Schemas**: Comprehensive schema structures for all core IAM and tenant-boundary activities.
- **Built-in Fault Tolerance**: Automatic exponential-backoff retries and redirection to DLQ sinks on unrecoverable failures.
- **Simulation Suite**: Includes `MemoryEventBroker` to enable lightweight, ultra-fast local development and end-to-end integration tests without requiring an active external Kafka cluster.

## Quick Start

```go
import (
	"context"
	"encoding/json"
	"time"

	"github.com/nexuscore/identity-platform/shared-event-library"
)

func main() {
	// Initialize a resilient event broker
	broker := event.NewMemoryEventBroker(3, 10*time.Millisecond)

	// Define Subscriber
	_ = broker.Subscribe(context.Background(), "nc.iam.user.user-created.v1", func(ctx context.Context, ev *event.CloudEvent) error {
		var payload event.UserCreatedEventPayload
		_ = json.Unmarshal(ev.Data, &payload)
		println("Successfully processed user initialization:", payload.Email)
		return nil
	})

	// Generate and Publish Event
	payload := event.UserCreatedEventPayload{
		UserID:    "user-123",
		TenantID:  "tenant-456",
		Email:     "user@nexuscore.com",
		CreatedAt: time.Now(),
	}
	data, _ := json.Marshal(payload)

	cloudevent := &event.CloudEvent{
		SpecVersion:     "1.0",
		ID:              "evt-789",
		Source:          "auth-service",
		Type:            "user-created",
		Time:            time.Now(),
		DataContentType: "application/json",
		TenantID:        "tenant-456",
		Data:            data,
	}

	_ = broker.Publish(context.Background(), "nc.iam.user.user-created.v1", "tenant-456", cloudevent)
}
```
