/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState, useEffect } from "react";
import { motion, AnimatePresence } from "motion/react";
import { ServiceNode, TelemetryPoint, LogLine, Trace, ServiceCode } from "./types";
import TopologyView from "./components/TopologyView";
import MetricsDashboard from "./components/MetricsDashboard";
import LogsAndTraces from "./components/LogsAndTraces";
import ServiceDesigner from "./components/ServiceDesigner";
import Playground from "./components/Playground";
import EventDrivenDashboard from "./components/EventDrivenDashboard";
import AIPlatformDashboard from "./components/AIPlatformDashboard";
import { Server, Activity, Shield, Cpu, Play, RefreshCw, Terminal, Layers, HelpCircle, HardDrive, Cpu as CpuIcon, Network, Database } from "lucide-react";

export default function App() {
  const [activeTab, setActiveTab] = useState<"topology" | "metrics" | "tracing" | "designer" | "playground" | "event-driven" | "ai-platform">("topology");
  const [currentWorkload, setCurrentWorkload] = useState<string>("normal");
  const [deployedServices, setDeployedServices] = useState<ServiceCode[]>([]);

  // 1. Initialize core Service Nodes representing Kubernetes state
  const [nodes, setNodes] = useState<ServiceNode[]>([
    { id: "api-gateway", name: "api-gateway", role: "API Gateway / Envoy Ingress Router", port: 8080, status: "HEALTHY", replicas: { active: 3, total: 3 }, cpu: 15, mem: 42 },
    { id: "auth-service", name: "auth-service", role: "Auth Check & Token Provider", port: 8081, status: "HEALTHY", replicas: { active: 2, total: 2 }, cpu: 8, mem: 28 },
    { id: "compute-engine", name: "compute-engine", role: "Transactional Core / Business Workflows", port: 8082, status: "HEALTHY", replicas: { active: 3, total: 3 }, cpu: 22, mem: 65 },
    { id: "postgres-db", name: "postgres-db", role: "Cloud SQL PostgreSQL Primary Instance", port: 5432, status: "HEALTHY", replicas: { active: 1, total: 1 }, cpu: 32, mem: 120 },
    { id: "telemetry-collector", name: "telemetry-collector", role: "OTel Collector / Prometheus Exporter", port: 8083, status: "HEALTHY", replicas: { active: 1, total: 1 }, cpu: 5, mem: 15 },
  ]);

  // 2. Initialize Telemetry coordinates history for Recharts
  const [telemetryHistory, setTelemetryHistory] = useState<TelemetryPoint[]>(() => {
    // Bootstrap historical coordinates for a gorgeous initial chart rendering
    const points: TelemetryPoint[] = [];
    const now = Date.now();
    for (let i = 29; i >= 0; i--) {
      const timeStr = new Date(now - i * 5000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
      points.push({
        time: timeStr,
        timestamp: now - i * 5000,
        rps: 125 + Math.floor(Math.random() * 20),
        errorRate: 0,
        latencyP50: 45 + Math.floor(Math.random() * 10),
        latencyP95: 75 + Math.floor(Math.random() * 15),
        latencyP99: 110 + Math.floor(Math.random() * 20),
        cpuUsage: 18 + Math.floor(Math.random() * 4),
        memUsage: 38 + Math.floor(Math.random() * 2),
      });
    }
    return points;
  });

  // 3. Initialize structured slog logs list
  const [logs, setLogs] = useState<LogLine[]>(() => [
    { time: new Date(Date.now() - 50000).toISOString(), level: "info", service: "api-gateway", msg: "Gateway Envoy service started and listening on :8080", caller: "cmd/server/main.go:34" },
    { time: new Date(Date.now() - 40000).toISOString(), level: "info", service: "auth-service", msg: "Token validation keys parsed and validated successfully", caller: "internal/delivery/http/handler.go:88" },
    { time: new Date(Date.now() - 30000).toISOString(), level: "info", service: "compute-engine", msg: "Database connection pool established. MinActive=10 MaxActive=100", caller: "internal/repository/postgres.go:42" },
    { time: new Date(Date.now() - 20000).toISOString(), level: "info", service: "telemetry-collector", msg: "OTel Jaeger Collector successfully booted and listening on :4317 (gRPC)", caller: "cmd/collector/main.go:22" },
  ]);

  // 4. Initialize OpenTelemetry Trace List
  const [traces, setTraces] = useState<Trace[]>(() => {
    const traceId = `tr_${Math.random().toString(36).substring(2, 10)}`;
    return [{
      id: traceId,
      name: "POST /transactions",
      method: "POST",
      path: "/transactions",
      statusCode: 201,
      startTime: new Date(Date.now() - 10000).toISOString(),
      totalDuration: 135,
      spans: [
        { id: "span_gw_01", name: "Envoy.Router", service: "api-gateway", startTime: 0, duration: 135, status: "OK", attributes: { "http.method": "POST", "http.route": "/transactions" } },
        { id: "span_auth_01", parentId: "span_gw_01", name: "JWT.Verify", service: "auth-service", startTime: 5, duration: 25, status: "OK", attributes: { "auth.type": "JWT", "auth.client_id": "api_client_core" } },
        { id: "span_comp_01", parentId: "span_gw_01", name: "Usecase.CreateTransaction", service: "compute-engine", startTime: 35, duration: 95, status: "OK", attributes: { "transaction.amount": 450.50 } },
        { id: "span_db_01", parentId: "span_comp_01", name: "Postgres.Exec", service: "postgres-db", startTime: 40, duration: 55, status: "OK", attributes: { "db.statement": "INSERT INTO transactions...", "db.rows_affected": 1 } },
      ]
    }];
  });

  // Action: Add simulated logs/traces triggered directly by user requests
  const handleTriggerEvent = (msg: string, level: "info" | "warn" | "error") => {
    const newLine: LogLine = {
      time: new Date().toISOString(),
      level: level,
      service: "platform-controller",
      msg: msg,
      caller: "internal/usecase/controller.go:120"
    };
    setLogs(prev => [...prev.slice(-200), newLine]);
  };

  const handleUpdateNode = (nodeId: string, updates: Partial<ServiceNode>) => {
    setNodes(prev => prev.map(n => n.id === nodeId ? { ...n, ...updates } : n));
  };

  const handleDeployService = (service: ServiceCode) => {
    setDeployedServices(prev => [...prev, service]);
    
    // Register the custom generated service inside our Kubernetes topology nodes state!
    const size = nodes.length;
    const newK8sNode: ServiceNode = {
      id: service.serviceName.toLowerCase().replace(/[^a-z0-9]/g, "-"),
      name: service.serviceName,
      role: service.description,
      port: 8080 + size,
      status: "HEALTHY",
      replicas: { active: 3, total: 3 },
      cpu: 12,
      mem: 35
    };
    setNodes(prev => [...prev, newK8sNode]);
  };

  // Capture playground mock requests and map them directly to real-time OTel traces and structured logs
  const handleSendMockRequest = (method: string, path: string, body: any) => {
    const now = new Date();
    const traceId = `tr_${Math.random().toString(36).substring(2, 10)}`;
    const spanGwId = `span_gw_${Math.random().toString(36).substring(2, 6)}`;
    const spanAuthId = `span_auth_${Math.random().toString(36).substring(2, 6)}`;
    const spanCompId = `span_comp_${Math.random().toString(36).substring(2, 6)}`;
    const spanDbId = `span_db_${Math.random().toString(36).substring(2, 6)}`;

    const isGet = method === "GET";
    const amount = body ? body.amount : 350.50;

    let isOutage = currentWorkload === "outage" || nodes.find(n => n.id === "postgres-db")?.status === "OFFLINE";
    let isDegraded = currentWorkload === "degraded";

    let statusCode = isGet ? 200 : 201;
    let traceDuration = isOutage ? 1200 : isDegraded ? 2100 : 85 + Math.floor(Math.random() * 30);

    if (isOutage) {
      statusCode = 503;
    } else if (!isGet && amount <= 0) {
      statusCode = 422;
    }

    // 1. Build beautiful contextual trace waterfall spans
    const spans: any[] = [
      { id: spanGwId, name: "Envoy.Router", service: "api-gateway", startTime: 0, duration: traceDuration, status: statusCode >= 500 ? "ERROR" : "OK", attributes: { "http.method": method, "http.route": path } }
    ];

    if (nodes.find(n => n.id === "auth-service")?.status !== "OFFLINE") {
      spans.push({
        id: spanAuthId, parentId: spanGwId, name: "JWT.Verify", service: "auth-service",
        startTime: 5, duration: isDegraded ? 1800 : 25, status: "OK",
        attributes: { "auth.type": "JWT", "auth.client_id": "api_client_core" },
        events: isDegraded ? [{ name: "connection_pool_contention_detected", timestamp: new Date().toISOString() }] : undefined
      });
    }

    if (nodes.find(n => n.id === "compute-engine")?.status !== "OFFLINE" && statusCode !== 401) {
      spans.push({
        id: spanCompId, parentId: spanGwId, name: "Usecase.ProcessTransaction", service: "compute-engine",
        startTime: isDegraded ? 1835 : 35, duration: isOutage ? 1165 : 45, status: statusCode >= 400 ? "ERROR" : "OK",
        attributes: { "transaction.amount": amount }
      });

      if (!isGet && !isOutage && statusCode !== 422) {
        spans.push({
          id: spanDbId, parentId: spanCompId, name: "Postgres.Exec", service: "postgres-db",
          startTime: isDegraded ? 1850 : 50, duration: 30, status: "OK",
          attributes: { "db.statement": "INSERT INTO transactions...", "db.rows_affected": 1 }
        });
      }
    }

    const newTrace: Trace = {
      id: traceId,
      name: `${method} ${path}`,
      method,
      path,
      statusCode,
      startTime: now.toISOString(),
      totalDuration: traceDuration,
      spans: spans
    };

    setTraces(prev => [newTrace, ...prev.slice(0, 19)]);

    // 2. Build related structured logs in the console terminal
    const gatewayLog: LogLine = {
      time: now.toISOString(),
      level: statusCode >= 500 ? "error" : "info",
      service: "api-gateway",
      trace_id: traceId,
      span_id: spanGwId,
      msg: `HTTP Request processed: ${method} ${path} -> ${statusCode}`,
      caller: "internal/delivery/http/handler.go:42",
      attributes: { "latency_ms": traceDuration, "status_code": statusCode }
    };

    const logsToPush = [gatewayLog];

    if (nodes.find(n => n.id === "auth-service")?.status !== "OFFLINE") {
      logsToPush.push({
        time: new Date(now.getTime() + 5).toISOString(),
        level: isDegraded ? "warn" : "info",
        service: "auth-service",
        trace_id: traceId,
        span_id: spanAuthId,
        msg: isDegraded ? "DB Connection pool saturation: waiting for lock" : "Token authenticated: sub=user_nexus_09",
        caller: "internal/usecase/service.go:78"
      });
    }

    if (nodes.find(n => n.id === "compute-engine")?.status !== "OFFLINE" && statusCode !== 401) {
      logsToPush.push({
        time: new Date(now.getTime() + 35).toISOString(),
        level: statusCode >= 400 ? "error" : "info",
        service: "compute-engine",
        trace_id: traceId,
        span_id: spanCompId,
        msg: isOutage 
          ? "Database connectivity handshake timed out. Retrying connection..." 
          : statusCode === 422 
            ? "Validation warning: negative transactional amount supplied" 
            : `Business validation approved for amount $${amount}`,
        caller: "internal/usecase/service.go:114"
      });
    }

    setLogs(prev => [...prev.slice(-200), ...logsToPush]);
  };

  // 5. Orchestrate background telemetry metrics loops
  useEffect(() => {
    const interval = setInterval(() => {
      const now = Date.now();
      const timeStr = new Date(now).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });

      // Shift metrics based on operational workload types
      let rps = 120 + Math.floor(Math.random() * 25);
      let errorRate = 0;
      let latencyP50 = 42 + Math.floor(Math.random() * 8);
      let latencyP95 = 72 + Math.floor(Math.random() * 12);
      let latencyP99 = 105 + Math.floor(Math.random() * 18);
      let cpuUsage = 20 + Math.floor(Math.random() * 4);
      let memUsage = 40 + Math.floor(Math.random() * 2);

      // Check offline nodes and update their dynamic CPU/RAM readouts
      const offlineCount = nodes.filter(n => n.status === "OFFLINE").length;
      if (offlineCount > 0) {
        errorRate = Math.min(offlineCount * 25, 100);
        latencyP99 += offlineCount * 300;
        cpuUsage = Math.max(cpuUsage - offlineCount * 4, 2);
      }

      if (currentWorkload === "high-traffic") {
        rps = 1350 + Math.floor(Math.random() * 120);
        latencyP50 = 110 + Math.floor(Math.random() * 20);
        latencyP95 = 195 + Math.floor(Math.random() * 35);
        latencyP99 = 320 + Math.floor(Math.random() * 45);
        cpuUsage = 82 + Math.floor(Math.random() * 5);
        memUsage = 86 + Math.floor(Math.random() * 3);
      } else if (currentWorkload === "outage") {
        errorRate = 85 + Math.floor(Math.random() * 10);
        latencyP99 = 1150 + Math.floor(Math.random() * 150);
        cpuUsage = 38 + Math.floor(Math.random() * 5);
      } else if (currentWorkload === "degraded") {
        latencyP50 = 1200 + Math.floor(Math.random() * 150);
        latencyP95 = 1850 + Math.floor(Math.random() * 200);
        latencyP99 = 2200 + Math.floor(Math.random() * 250);
        cpuUsage = 58 + Math.floor(Math.random() * 6);
        memUsage = 62 + Math.floor(Math.random() * 4);
      }

      // Append new coordinate point
      setTelemetryHistory(prev => {
        const next = [...prev, {
          time: timeStr,
          timestamp: now,
          rps,
          errorRate,
          latencyP50,
          latencyP95,
          latencyP99,
          cpuUsage,
          memUsage
        }];
        return next.slice(-40); // Maintain a clean history size
      });

      // Update Node CPU & Memory readouts dynamically to reflect real-time loads
      setNodes(prev => prev.map(n => {
        if (n.status === "OFFLINE") {
          return { ...n, cpu: 0, mem: 0 };
        }
        const loadFactor = currentWorkload === "high-traffic" ? 3.5 : currentWorkload === "degraded" ? 1.8 : 1.0;
        return {
          ...n,
          cpu: Math.min(Math.round(n.cpu * (0.9 + Math.random() * 0.2) * loadFactor), 100),
          mem: Math.min(Math.round(n.mem * (0.95 + Math.random() * 0.1) * (currentWorkload === "high-traffic" ? 1.5 : 1.0)), 512)
        };
      }));

      // Append a set of background trace flows to populate trace list dynamically
      if (Math.random() > 0.4) {
        const randomTraceId = `tr_${Math.random().toString(36).substring(2, 10)}`;
        const randomPath = ["/transactions", "/transactions", "/auth/token", "/metrics"][Math.floor(Math.random() * 4)];
        const randomMethod = randomPath === "/transactions" && Math.random() > 0.3 ? "POST" : "GET";
        
        const pathWithId = randomPath === "/transactions" && randomMethod === "GET" 
          ? `${randomPath}/tx_${Math.random().toString(36).substring(2, 8)}`
          : randomPath;

        let statusCode = 200;
        if (randomPath === "/transactions" && randomMethod === "POST") statusCode = 210; // pending or success
        if (errorRate > 30 && Math.random() > 0.2) statusCode = 503;

        const dummyTrace: Trace = {
          id: randomTraceId,
          name: `${randomMethod} ${randomPath}`,
          method: randomMethod,
          path: pathWithId,
          statusCode: statusCode === 210 ? 201 : statusCode,
          startTime: new Date().toISOString(),
          totalDuration: Math.round(latencyP50 * (0.8 + Math.random() * 0.4)),
          spans: [
            { id: `sp_gw_${randomTraceId}`, name: "Envoy.Router", service: "api-gateway", startTime: 0, duration: Math.round(latencyP50), status: statusCode >= 500 ? "ERROR" : "OK", attributes: { "http.method": randomMethod, "http.path": pathWithId } },
            { id: `sp_auth_${randomTraceId}`, parentId: `sp_gw_${randomTraceId}`, name: "JWT.Verify", service: "auth-service", startTime: 2, duration: Math.round(latencyP50 * 0.2), status: "OK", attributes: { "auth.scope": "read_write" } }
          ]
        };

        setTraces(prev => [dummyTrace, ...prev.slice(0, 19)]);
      }

      // Generate Background standard slog JSON logs corresponding to current telemetry
      if (Math.random() > 0.3) {
        const logService = ["api-gateway", "auth-service", "compute-engine", "postgres-db"][Math.floor(Math.random() * 4)];
        const level = errorRate > 50 && Math.random() > 0.3 ? "error" : "info";
        
        let msg = `Completed background check routing in ${latencyP50}ms`;
        if (level === "error") {
          msg = "database/sql connection pool: too many open connections. Dropping socket packet.";
        } else if (currentWorkload === "high-traffic") {
          msg = `High traffic capacity spike resolved. Processed ${rps} concurrent HTTP packets.`;
        }

        const dummyLog: LogLine = {
          time: new Date().toISOString(),
          level,
          service: logService,
          msg,
          caller: "internal/delivery/http/handler.go:210"
        };
        setLogs(prev => [...prev.slice(-180), dummyLog]);
      }

    }, 3000);

    return () => clearInterval(interval);
  }, [currentWorkload, nodes]);

  return (
    <div className="min-h-screen bg-slate-950 text-slate-200 font-sans flex flex-col md:flex-row overflow-hidden select-none" id="nexuscore-root">
      {/* Sidebar: Service Navigation (Desktops) */}
      <aside className="hidden md:flex w-64 border-r border-slate-800 bg-slate-900/50 flex-col shrink-0">
        <div className="p-6 border-b border-slate-800 flex items-center space-x-3">
          <div className="w-8 h-8 bg-indigo-600 rounded flex items-center justify-center font-bold text-white">N</div>
          <span className="text-lg font-bold tracking-tight text-white font-display">NEXUS<span className="text-indigo-400">CORE</span></span>
        </div>
        
        <nav className="flex-1 p-4 space-y-1.5 overflow-y-auto">
          <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3 px-2">System Services</div>
          
          <button
            onClick={() => setActiveTab("topology")}
            className={`w-full text-left px-3 py-2 rounded-md flex items-center justify-between text-xs transition-all ${
              activeTab === "topology"
                ? "bg-slate-800/80 text-white font-semibold border border-slate-700/50"
                : "text-slate-400 hover:bg-slate-800/40 hover:text-slate-200"
            }`}
          >
            <span className="flex items-center">
              <span className={`w-2 h-2 rounded-full mr-3 ${nodes.some(n => n.status === "OFFLINE") ? "bg-amber-500 animate-pulse" : "bg-emerald-500"}`} />
              API Topology
            </span>
            <span className="text-[10px] bg-emerald-500/10 text-emerald-400 px-1.5 py-0.5 rounded border border-emerald-500/20">v2.4.1</span>
          </button>

          <button
            onClick={() => setActiveTab("metrics")}
            className={`w-full text-left px-3 py-2 rounded-md flex items-center justify-between text-xs transition-all ${
              activeTab === "metrics"
                ? "bg-slate-800/80 text-white font-semibold border border-slate-700/50"
                : "text-slate-400 hover:bg-slate-800/40 hover:text-slate-200"
            }`}
          >
            <span className="flex items-center">
              <span className="w-2 h-2 bg-emerald-500 rounded-full mr-3" />
              Data Mesh Metrics
            </span>
            <span className="text-[10px] bg-indigo-500/10 text-indigo-400 px-1.5 py-0.5 rounded border border-indigo-500/20">v3.0.4</span>
          </button>

          <button
            onClick={() => setActiveTab("tracing")}
            className={`w-full text-left px-3 py-2 rounded-md flex items-center justify-between text-xs transition-all ${
              activeTab === "tracing"
                ? "bg-slate-800/80 text-white font-semibold border border-slate-700/50"
                : "text-slate-400 hover:bg-slate-800/40 hover:text-slate-200"
            }`}
          >
            <span className="flex items-center">
              <span className="w-2 h-2 bg-emerald-500 rounded-full mr-3" />
              Logs & Tracing
            </span>
            <span className="text-[10px] bg-slate-800 text-slate-400 px-1.5 py-0.5 rounded">OTel</span>
          </button>

          <button
            onClick={() => setActiveTab("event-driven")}
            className={`w-full text-left px-3 py-2 rounded-md flex items-center justify-between text-xs transition-all ${
              activeTab === "event-driven"
                ? "bg-slate-800/80 text-white font-semibold border border-slate-700/50"
                : "text-slate-400 hover:bg-slate-800/40 hover:text-slate-200"
            }`}
          >
            <span className="flex items-center">
              <Database className="text-emerald-400 mr-3" size={13} />
              Event-Driven EDA
            </span>
            <span className="text-[10px] bg-emerald-500/10 text-emerald-400 px-1.5 py-0.5 rounded border border-emerald-500/20 font-bold">CQRS</span>
          </button>

          <div className="mt-8 text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3 px-2">Cognitive & AI Platform</div>

          <button
            onClick={() => setActiveTab("ai-platform")}
            className={`w-full text-left px-3 py-2 rounded-md flex items-center justify-between text-xs transition-all ${
              activeTab === "ai-platform"
                ? "bg-slate-800/80 text-white font-semibold border border-slate-700/50"
                : "text-slate-400 hover:bg-slate-800/40 hover:text-slate-200"
            }`}
          >
            <span className="flex items-center">
              <Cpu className="text-indigo-400 mr-3" size={13} />
              AI Gateway Engine
            </span>
            <span className="text-[10px] bg-indigo-500/10 text-indigo-400 px-1.5 py-0.5 rounded border border-indigo-500/20 font-bold font-mono">v1.5.0</span>
          </button>

          <div className="mt-8 text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3 px-2">Architecture & Testing</div>

          <button
            onClick={() => setActiveTab("designer")}
            className={`w-full text-left px-3 py-2 rounded-md flex items-center justify-between text-xs transition-all ${
              activeTab === "designer"
                ? "bg-slate-800/80 text-white font-semibold border border-slate-700/50"
                : "text-slate-400 hover:bg-slate-800/40 hover:text-slate-200"
            }`}
          >
            <span className="flex items-center">
              <span className="w-2 h-2 bg-indigo-500 rounded-full mr-3" />
              AI Code Architect
            </span>
            <span className="text-[10px] bg-indigo-500/10 text-indigo-400 px-1.5 py-0.5 rounded border border-indigo-500/20">v1.0.1</span>
          </button>

          <button
            onClick={() => setActiveTab("playground")}
            className={`w-full text-left px-3 py-2 rounded-md flex items-center justify-between text-xs transition-all ${
              activeTab === "playground"
                ? "bg-slate-800/80 text-white font-semibold border border-slate-700/50"
                : "text-slate-400 hover:bg-slate-800/40 hover:text-slate-200"
            }`}
          >
            <span className="flex items-center">
              <span className="w-2 h-2 bg-indigo-500 rounded-full mr-3" />
              API Playground
            </span>
            <span className="text-[10px] bg-slate-800 text-slate-400 px-1.5 py-0.5 rounded">Client</span>
          </button>
        </nav>

        <div className="p-4 border-t border-slate-800">
          <div className="bg-slate-950 p-3 rounded-lg border border-slate-800">
            <div className="text-[10px] text-slate-500 uppercase mb-1">Architect Mode</div>
            <div className="text-xs text-indigo-400 font-mono">PROD_ENVIRONMENT_ON</div>
          </div>
        </div>
      </aside>

      {/* Mobile Header */}
      <div className="flex md:hidden bg-slate-950 border-b border-slate-800 px-4 py-3.5 items-center justify-between shrink-0">
        <div className="flex items-center space-x-2">
          <div className="w-7 h-7 bg-indigo-600 rounded flex items-center justify-center font-bold text-white text-sm">N</div>
          <span className="text-base font-bold tracking-tight text-white font-display">NEXUS<span className="text-indigo-400">CORE</span></span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="h-2.5 w-2.5 rounded-full bg-emerald-500 animate-pulse" />
          <span className="text-[10px] text-slate-400 font-mono font-bold tracking-wider">K8S: ACTIVE</span>
        </div>
      </div>

      {/* Mobile Tab Swipe bar */}
      <div className="flex md:hidden bg-slate-900 border-b border-slate-800 px-3 py-2 gap-1.5 overflow-x-auto scrollbar-none shrink-0">
        {[
          { id: "topology", name: "Topology" },
          { id: "metrics", name: "Metrics" },
          { id: "tracing", name: "Traces & Logs" },
          { id: "event-driven", name: "Event-Driven" },
          { id: "ai-platform", name: "AI Gateway" },
          { id: "designer", name: "AI Architect" },
          { id: "playground", name: "Playground" }
        ].map(tab => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id as any)}
            className={`px-3 py-1.5 rounded text-xs font-semibold whitespace-nowrap transition-all ${
              activeTab === tab.id ? "bg-indigo-600 text-white" : "text-slate-400 hover:bg-slate-800"
            }`}
          >
            {tab.name}
          </button>
        ))}
      </div>

      {/* Main Content Area */}
      <main className="flex-1 flex flex-col min-h-0 bg-slate-950">
        {/* Header */}
        <header className="h-16 border-b border-slate-800 bg-slate-900/30 flex items-center justify-between px-6 md:px-8 shrink-0">
          <div className="flex items-center space-x-6">
            <h1 className="text-xs md:text-sm font-semibold text-white tracking-wide uppercase font-display">NexusCore Global Control Plane</h1>
            <div className="hidden sm:block h-4 w-px bg-slate-700"></div>
            <div className="hidden sm:flex items-center space-x-4">
              <div className="flex items-center space-x-2">
                <span className="text-xs text-slate-500 italic font-serif">Cluster Status:</span>
                <span className={`text-xs font-medium ${nodes.some(n => n.status === "OFFLINE") ? "text-amber-400" : "text-emerald-400"}`}>
                  {nodes.some(n => n.status === "OFFLINE") ? "Degraded" : "Healthy"}
                </span>
              </div>
              <div className="flex items-center space-x-2">
                <span className="text-xs text-slate-500 italic font-serif">Region:</span>
                <span className="text-xs font-medium text-slate-300">us-east-multizone</span>
              </div>
            </div>
          </div>
          
          <div className="flex items-center space-x-4">
            <div className="text-right">
              <div className="text-xs font-bold text-white">Principal Architect</div>
              <div className="text-[10px] text-slate-500">Session ID: 49fa-120x</div>
            </div>
            <div className="w-8 h-8 rounded-full bg-slate-800 border border-slate-700 flex items-center justify-center text-[10px] font-bold text-indigo-400">PA</div>
          </div>
        </header>

        {/* Content Container */}
        <section className="flex-1 p-4 md:p-6 min-h-0 overflow-y-auto">
          <AnimatePresence mode="wait">
            <motion.div
              key={activeTab}
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              transition={{ duration: 0.15 }}
              className="h-full"
            >
              {activeTab === "topology" && (
                <TopologyView
                  nodes={nodes}
                  onUpdateNode={handleUpdateNode}
                  onTriggerEvent={handleTriggerEvent}
                  activeWorkload={currentWorkload}
                />
              )}

              {activeTab === "metrics" && (
                <MetricsDashboard
                  telemetryHistory={telemetryHistory}
                  currentWorkload={currentWorkload}
                  onSetWorkload={setCurrentWorkload}
                />
              )}

              {activeTab === "tracing" && (
                <LogsAndTraces
                  logs={logs}
                  traces={traces}
                  onTriggerEvent={handleTriggerEvent}
                />
              )}

              {activeTab === "designer" && (
                <ServiceDesigner
                  onTriggerEvent={handleTriggerEvent}
                  onDeployService={handleDeployService}
                />
              )}

              {activeTab === "playground" && (
                <Playground
                  onTriggerEvent={handleTriggerEvent}
                  onSendMockRequest={handleSendMockRequest}
                />
              )}

              {activeTab === "event-driven" && (
                <EventDrivenDashboard />
              )}

              {activeTab === "ai-platform" && (
                <AIPlatformDashboard />
              )}
            </motion.div>
          </AnimatePresence>
        </section>

        {/* Footer */}
        <footer className="bg-slate-950 border-t border-slate-800/60 py-3 px-6 md:px-8 flex items-center justify-between text-[11px] text-slate-500 font-mono shrink-0">
          <div className="flex items-center gap-4">
            <span>PLATFORM ENGINE v1.2.5</span>
            <span className="hidden sm:inline">CLUSTER INDEPENDENT NODE APPARENT</span>
          </div>
          <div className="flex items-center gap-2">
            <span>UPTIME: 100%</span>
            <span>● OPERATIONAL</span>
          </div>
        </footer>
      </main>
    </div>
  );
}
