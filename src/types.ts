export interface ServiceCode {
  serviceName: string;
  description: string;
  files: {
    "cmd/server/main.go": string;
    "internal/domain/entity.go": string;
    "internal/repository/postgres.go": string;
    "internal/usecase/service.go": string;
    "internal/delivery/http/handler.go": string;
    "Dockerfile": string;
    "k8s/deployment.yaml": string;
    "charts/helm/values.yaml": string;
    "telemetry/prometheus.yml": string;
    ".github/workflows/ci-cd.yaml": string;
    "api/openapi.yaml": string;
    "internal/usecase/service_test.go": string;
    [key: string]: string;
  };
}

export interface TraceSpan {
  id: string;
  parentId?: string;
  name: string;
  service: string;
  startTime: number; // ms offset relative to trace start
  duration: number; // ms
  status: "OK" | "ERROR";
  attributes: Record<string, string | number | boolean>;
  events?: Array<{
    name: string;
    timestamp: string;
    attributes?: Record<string, any>;
  }>;
}

export interface Trace {
  id: string;
  name: string;
  method: string;
  path: string;
  statusCode: number;
  startTime: string;
  totalDuration: number; // ms
  spans: TraceSpan[];
}

export interface LogLine {
  time: string;
  level: "info" | "warn" | "error" | "debug";
  service: string;
  trace_id?: string;
  span_id?: string;
  msg: string;
  caller: string;
  attributes?: Record<string, any>;
}

export interface TelemetryPoint {
  time: string;
  timestamp: number;
  rps: number;
  errorRate: number; // percentage
  latencyP50: number; // ms
  latencyP95: number; // ms
  latencyP99: number; // ms
  cpuUsage: number; // percentage
  memUsage: number; // percentage
}

export interface ServiceNode {
  id: string;
  name: string;
  role: string;
  port: number;
  status: "HEALTHY" | "DEGRADED" | "CRITICAL" | "OFFLINE";
  replicas: {
    active: number;
    total: number;
  };
  cpu: number;
  mem: number;
}
