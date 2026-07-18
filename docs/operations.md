# Operations & Observability Specification (NexusCore)

This document details the telemetry stack, alerting metrics thresholds, log aggregation models, and distributed tracing setups.

## 1. Observability Infrastructure Architecture

Observability is implemented at three distinct tiers: Metrics (Prometheus), Logs (Grafana Loki), and Distributed Tracing (Jaeger / OpenTelemetry Collector).

```
 +------------------+      +--------------------+      +------------------+
 | Metrics Scraper  |      |   Log Aggregator   |      |  Trace Collector |
 |  (Prometheus)    |      |   (Grafana Loki)   |      |  (OpenTelemetry) |
 +------------------+      +--------------------+      +------------------+
         ^                            ^                           ^
         |                            |                           |
         +----------------------------+---------------------------+
                                      |
                     [ Microservice Mesh / Pod Nodes ]
```

---

## 2. Prometheus Scraping & Custom Instrumentation

Every microservice exposes a `/metrics` Prometheus endpoint at its respective port, instrumented using the `prometheus/client_golang` library.

### 2.1 Prometheus Service Monitor Configuration
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: nexuscore-metrics
  namespace: nexus-core
spec:
  selector:
    matchLabels:
      tier: edge
  endpoints:
  - port: http
    path: /metrics
    interval: 15s
    scrapeTimeout: 10s
```

### 2.2 SRE Golden Metrics Threshold Alarms
Prometheus AlertManager rules are configured in the cluster to alert SRE personnel immediately upon breach of vital thresholds:

*   **API Latency Alarm (`p95` response duration > 200ms)**:
    ```yaml
    alert: APIHighLatency
    expr: histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le)) > 0.200
    for: 2m
    labels:
      severity: critical
      tier: edge
    annotations:
      summary: "High API Ingress Latency on {{ $labels.instance }}"
      description: "p95 response latency exceeded 200ms baseline limits (current: {{ $value }}s)."
    ```

*   **API High Error Rate (HTTP 5xx rate > 1%)**:
    ```yaml
    alert: APIHighErrorRate
    expr: sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) * 100 > 1.0
    for: 1m
    labels:
      severity: page
    annotations:
      summary: "Elevated HTTP 5xx errors detected"
      description: "System error rates hit {{ $value }}% on API paths."
    ```

---

## 3. Distributed Tracing with OpenTelemetry & Jaeger

Every inbound HTTP request generates a unique `X-Correlation-ID` and OpenTelemetry span context at the `api-gateway`. This span context is propagated across downstream gRPC/HTTP requests.

### 3.1 Trace Context Header Propagation (W3C Trace Context)
Downstream request clients inside the Go microservices must inject trace headers:
```http
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
```

### 3.2 OTel Exporter config
```yaml
receivers:
  otlp:
    protocols:
      grpc:
      http:
exporters:
  jaeger:
    endpoint: "jaeger-collector.nexus-core:14250"
    tls:
      insecure: true
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: []
      exporters: [jaeger]
```

---

## 4. Central Log Aggregation

All standard stdout/stderr outputs are written in structured JSON formatting. **FluentBit** daemonsets collect node logs and forward them to **Grafana Loki** or **Elasticsearch**.

### 4.1 Production Structured JSON Log Blueprint
```json
{
  "timestamp": "2026-07-18T14:10:02.991Z",
  "level": "ERROR",
  "service": "compute-engine",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "correlation_id": "tx_902831093123",
  "msg": "Transaction failed: balance would fall below zero threshold limit",
  "account_id": "acc-9921-prod-core",
  "stacktrace": "main.go:120: ... exception trace"
}
```
