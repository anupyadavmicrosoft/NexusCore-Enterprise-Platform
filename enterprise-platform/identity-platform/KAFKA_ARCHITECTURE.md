# NexusCore Identity Platform - Kafka Event Architecture Specification
## Sprint 3 High-Throughput Event-Driven IAM Ecosystem

This document describes the design patterns, topic topologies, schema governance, transactional outbox routing, and retry/DLQ (Dead Letter Queue) strategies for the **NexusCore Event Backbone**. Powered by Apache Kafka, this architecture enables real-time propagation of identity state changes across tenant bounds while guaranteeing strict sequence ordering and fault tolerance.

---

## 1. Topographical Topic Configurations & Partitioning Matrix

To ensure horizontal scalability, topics are configured with multiple partitions. Events relating to the same tenant or user are routed to the same partition to maintain strict causal ordering (e.g., ensuring `user-created` is always processed before `user-updated`).

### 1.1 Topic Catalog & SLA Tuning

All topics follow the canonical naming convention: `nc.iam.{domain}.{event-type}.v{version}`

| Topic Name | Partitions | Replication Factor | Cleanup Policy | Retention Period | Partitioning Key |
| --- | :---: | :---: | :---: | :---: | --- |
| `nc.iam.tenant.tenant-created.v1` | 6 | 3 | `compact` | Infinite (Log Compacted) | `tenant_id` |
| `nc.iam.org.organization-created.v1` | 12 | 3 | `compact` | Infinite (Log Compacted) | `tenant_id` |
| `nc.iam.user.user-created.v1` | 24 | 3 | `compact` | Infinite (Log Compacted) | `user_id` |
| `nc.iam.user.user-updated.v1` | 24 | 3 | `delete` | 14 Days | `user_id` |
| `nc.iam.user.user-deleted.v1` | 24 | 3 | `delete` | 14 Days | `user_id` |
| `nc.iam.auth.login-success.v1` | 48 | 3 | `delete` | 7 Days | `user_id` |
| `nc.iam.auth.login-failed.v1` | 48 | 3 | `delete` | 3 Days | `client_ip` / `user_id` |
| `nc.iam.auth.logout.v1` | 24 | 3 | `delete` | 7 Days | `user_id` |
| `nc.iam.role.role-created.v1` | 6 | 3 | `compact` | Infinite (Log Compacted) | `tenant_id` |
| `nc.iam.role.permission-created.v1` | 6 | 3 | `compact` | Infinite (Log Compacted) | Global String |
| `nc.iam.audit.audit-events.v1` | 48 | 3 | `delete` | 365 Days | `tenant_id` |
| `nc.iam.notification.notification-events.v1` | 48 | 3 | `delete` | 24 Hours | `user_id` |

---

## 2. Event Envelope Structure & Schema Registry

Every event produced onto the NexusCore Kafka backbone conforms to the **CloudEvents v1.0 spec**. This ensures a universal wrapping mechanism containing operational metadata, tenant tracing vectors, and a strongly typed domain payload.

### 2.1 Unified Event Envelope (JSON Schema Representation)
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["specversion", "id", "source", "type", "time", "datacontenttype", "tenant_id", "data"],
  "properties": {
    "specversion": { "type": "string", "const": "1.0" },
    "id": { "type": "string", "format": "uuid" },
    "source": { "type": "string" },
    "type": { "type": "string" },
    "time": { "type": "string", "format": "date-time" },
    "datacontenttype": { "type": "string", "const": "application/json" },
    "tenant_id": { "type": "string", "format": "uuid" },
    "trace_id": { "type": "string" },
    "data": { "type": "object" }
  }
}
```

### 2.2 Schema Registry & Compatibility Policy
- **Registry Provider:** Confluent Schema Registry (JSON Schema / Avro compatibility mode).
- **Compatibility Level:** `BACKWARD_TRANSITIVE` (Guarantees that new consumers can process legacy messages, facilitating seamless Canary and Rolling updates).
- **Versioning Strategy:** Incrementing version suffix in topic name (`.v1`, `.v2`) is reserved purely for breaking changes that violate schema compatibility criteria.

---

## 3. High-Reliability Resiliency & DLQ Topology

Distributed systems suffer from transient database locks, network partitions, and down-time of third-party APIs. To prevent cascading system degradation, we implement a **Three-Tiered Retry/DLQ Pipeline** per consumer group.

```
                           +------------------------+
                           |  Primary Target Topic  |
                           +-----------+------------+
                                       |
                                       ▼
                             +--------------------+
                             |  Consumer Worker   |
                             +---------+----------+
                                       |
                 Processing Fails      | (First failure)
                 Transient Error       ▼
                           +------------------------+
                           | nc.iam.*.retry-1       | <--- 10s backoff delay
                           +-----------+------------+
                                       |
                                       ▼
                             +--------------------+
                             | Retry 1 Consumer   |
                             +---------+----------+
                                       |
                 Processing Fails      | (Second failure)
                 Transient Error       ▼
                           +------------------------+
                           | nc.iam.*.retry-2       | <--- 60s backoff delay
                           +-----------+------------+
                                       |
                                       ▼
                             +--------------------+
                             | Retry 2 Consumer   |
                             +---------+----------+
                                       |
                 Processing Fails      | (Final failure OR unrecoverable error)
                                       ▼
                           +------------------------+
                           | nc.iam.*.dlq           | <--- SRE Dashboard & Manual Replay
                           +------------------------+
```

### 3.1 Failure Classification Matrix
1. **Transient Failures (Retryable):** DB lockouts, HTTP 503 Service Unavailable, network connection timeouts.
   * *Action:* Publish to corresponding `.retry` queue with exponential backoff incrementing headers.
2. **Fatal Failures (Non-Retryable):** NullPointerException, Invalid Signature, Schema Validation Mismatch, JSON Parsing Failures.
   * *Action:* Route directly to the corresponding `.dlq` immediately.

### 3.2 Kafka Headers for Traceability
When routing message payloads across the pipeline, the producer injects the following structural tracing headers:
- `x-death-count`: Integer counting execution attempts across consumers.
- `x-original-topic`: String mapping the original target queue.
- `x-exception-message`: String conveying error context.
- `x-exception-stack`: Stack trace capture for SRE analysis.

---

### 🏆 Sprint 3 Kafka Architecture Review Sign-Off
- **Ordering Guarantee:** Enforced per partitioned key.
- **Delivery Semantics:** `At-Least-Once` utilizing manual consumer offset commits only after successful processing and DB write confirmations.
- **Throughput Profile:** Optimized batch sizes (`batch.size=131072` (128kb)), aggressive compression (`compression.type=zstd`), and low-latency acknowledgment settings (`acks=all` for security).
