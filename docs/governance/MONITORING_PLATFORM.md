# NexusCore Integrated Monitoring, Observability & Alerting Specification

## 1. Executive Mandate & Observability Architecture

This document establishes the official **NexusCore Enterprise Observability, Telemetry, and Alerting Standards**. 

Observability is a core pillar of our operational posture. We implement a unified, multi-dimensional telemetry system based on the **LGTM Stack (Loki, Grafana, Tempo, Mimir/Prometheus)** integrated with the **OpenTelemetry (OTel)** framework and **Jaeger/Alertmanager** backends. This delivers complete correlation across the four pillars of modern observability: **Metrics, Logs, Traces, and Alerts**.

All microservices must emit telemetry according to the specifications defined herein.

---

## 2. Integrated Telemetry Pipeline Architecture

The telemetry pipeline decouples data collection from storage and visualization, utilizing the industry-standard OpenTelemetry Collector as a high-throughput, low-latency forwarding engine:

```
┌──────────────────────────────────────────────────────────────────────────────────────────┐
│                                APPLICATION LAYER (Pods)                                  │
│  - Metrics: Prometheus Exporters (scraping port /metrics)                                 │
│  - Logs: stdout / stderr in JSON format                                                 │
│  - Traces: OpenTelemetry SDK (gRPC/HTTP forwarding to agent)                              │
└──────────────────────────────────────────┬───────────────────────────────────────────────┘
                                           ▼
┌──────────────────────────────────────────────────────────────────────────────────────────┐
│                            OPENTELEMETRY COLLECTOR DAEMONSET                             │
│  - Receives Traces (OTLP/gRPC), Logs (host paths), and Metrics (Prometheus)              │
│  - Batches, filters, scrubs sensitive data, and distributes payloads                    │
└──────┬───────────────────────────────────┼────────────────────────────────────────┬──────┘
       ▼ (Metrics OTLP)                    ▼ (Logs OTLP)                            ▼ (Traces OTLP)
┌──────────────┐                   ┌──────────────┐                         ┌──────────────┐
│  PROMETHEUS  │                   │  LOKI STACK  │                         │ JAEGER/TEMPO │
│ (Metrics DB) │                   │  (Log Index) │                         │  (Traces DB) │
└──────┬───────┘                   └──────┬───────┘                         └──────┬───────┘
       │                                  │                                        │
       └──────────────────────────────────┼────────────────────────────────────────┘
                                          ▼ (Correlated Dashboards)
                           ┌──────────────────────────────┐
                           │      GRAFANA ENTERPRISE      │
                           │  - Single Pane of Glass      │
                           │  - Alert routing gateway     │
                           └──────────────┬───────────────┘
                                          ▼
                           ┌──────────────────────────────┐
                           │         ALERTMANAGER         │
                           │  - Slack, PagerDuty, Email   │
                           └──────────────────────────────┘
```

---

## 3. Metrics Specification (Prometheus & OpenTelemetry)

### 3.1 Scraping & Discovery
*   Metrics collection is driven by the **Prometheus Operator** using `ServiceMonitor` and `PodMonitor` CRDs.
*   All microservices must expose Prometheus metrics on `/metrics` via port `8080` (or their native service ports).
*   Standard JVM, Go-runtime, or Node-runtime metrics must be enabled to track garbage collection cycles, heap usage, active goroutines/threads, and memory allocations.

### 3.2 Standard Golden Signals
Every microservice dashboard must display the **Four Golden Signals**:
1.  **Latency**: Time taken to service a request (split by route and response code, tracking p50, p95, and p99).
2.  **Traffic**: Demand placed on the system (e.g., HTTP requests per second).
3.  **Errors**: Rate of requests that fail (e.g., HTTP 5xx responses or unhandled exceptions).
4.  **Saturation**: How "full" the service is (typically memory utilization, thread pool usage, or queue depth).

---

## 4. Log Aggregation & Parsing Specification (Loki)

### 4.1 Structured Logging Standards
*   **All logs must be emitted exclusively in JSON format to standard output (`stdout`).**
*   Text-based logging is strictly prohibited in production to ensure deterministic indexing.
*   Every log message must contain the following structural schema:
    ```json
    {
      "timestamp": "2026-07-18T16:40:52.123Z",
      "level": "ERROR",
      "service": "auth-service",
      "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
      "span_id": "00f067aa0ba902b7",
      "message": "failed to connect to user cache",
      "error": "redis: connection refused",
      "context": {
        "user_id": "usr-883912",
        "attempt": 3
      }
    }
    ```

### 4.2 Log Correlation (Loki to Tempo/Jaeger)
*   By injecting `trace_id` and `span_id` directly into the JSON log payloads, Grafana automatically provisions interactive links.
*   An operator investigating a spike in logs can click a log line and instantly visualize the corresponding distributed trace in Tempo or Jaeger, reducing Mean Time to Resolution (MTTR).

---

## 5. Distributed Tracing Specification (Jaeger & Tempo)

### 5.1 Context Propagation
*   All microservices must implement W3C Trace Context propagation headers (`traceparent`, `tracestate`) on outbound HTTP/gRPC requests.
*   Downstream systems must extract the parent context to compile a unified, multi-service trace timeline.

### 5.2 Sampling Strategies
*   To prevent network bandwidth saturation and reduce storage costs:
    *   **Production Sampling**: Configured with a **10% Probabilistic Sampler** or an **Adaptive/Tail-Based Sampler** which selectively captures 100% of traces featuring HTTP 5xx errors or latencies exceeding 2 seconds, while discarding normal requests.
    *   **Staging/Development Sampling**: Configured with a **100% Const Sampler** to allow SRE teams to thoroughly validate distributed routing logic.

---

## 6. Enterprise Dashboard and Alertmanager Rules

*   **Alert Routing**: Alerts generated by Prometheus or Loki are funneled through **Alertmanager** for deduplication, silencing, grouping, and active routing.
*   **Severity Tiers**:
    *   `critical` (PagerDuty/On-Call SMS): Triggers immediately if an core system goes offline, database transactions fail, or user-facing error rate exceeds 5% for 2 minutes.
    *   `warning` (Slack/Email): Triggers if disk capacity reaches 80%, average latency exceeds SLA limits, or replica sets fail to achieve targeted counts.
