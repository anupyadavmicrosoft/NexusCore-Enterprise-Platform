import React, { useState } from "react";
import { ResponsiveContainer, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, BarChart, Bar, Legend } from "recharts";
import { TelemetryPoint } from "../types";
import { Activity, Radio, AlertOctagon, TrendingUp, Zap, Server } from "lucide-react";

interface MetricsProps {
  telemetryHistory: TelemetryPoint[];
  currentWorkload: string;
  onSetWorkload: (workload: string) => void;
}

export default function MetricsDashboard({ telemetryHistory, currentWorkload, onSetWorkload }: MetricsProps) {
  const latest = telemetryHistory[telemetryHistory.length - 1] || {
    rps: 0,
    errorRate: 0,
    latencyP50: 0,
    latencyP95: 0,
    latencyP99: 0,
    cpuUsage: 0,
    memUsage: 0,
  };

  const getWorkloadDesc = (w: string) => {
    switch (w) {
      case "normal": return "Generates stable, distributed background traffic mimicking healthy production workloads (120-150 RPS).";
      case "high-traffic": return "Simulates a massive seasonal burst or stress test, scaling load up to 1500 RPS and exercising HPA controllers.";
      case "outage": return "Injects core database connectivity timeouts, spiking 503 Service Unavailable errors across 80% of transactions.";
      case "degraded": return "Simulates high connection pool contention, compounding auth validation latency to 2000ms (p99).";
      default: return "";
    }
  };

  // Custom tooltips for Recharts
  const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
      return (
        <div className="bg-slate-950 border border-slate-800 p-3 rounded-lg shadow-2xl font-mono text-xs text-slate-300">
          <p className="text-slate-400 font-sans font-semibold mb-1 border-b border-slate-900 pb-1">{label}</p>
          {payload.map((item: any, idx: number) => (
            <p key={idx} style={{ color: item.color }} className="flex items-center gap-1.5 py-0.5">
              <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: item.color }} />
              {item.name}: <span className="font-bold text-white">{item.value.toFixed(1)}</span>
              {item.unit || ""}
            </p>
          ))}
        </div>
      );
    }
    return null;
  };

  return (
    <div className="space-y-6" id="nexuscore-metrics-root">
      {/* Workload Ingress Controller Header */}
      <div className="bg-slate-950 rounded-xl border border-slate-800 p-5">
        <h3 className="font-display font-semibold text-white tracking-wide mb-1">Synthetic Load Generator</h3>
        <p className="text-xs text-slate-400 font-sans mb-4">Simulate operational environments and evaluate core platform elasticity and tracing behavior.</p>
        
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
          {[
            { id: "normal", name: "Healthy / Standard", color: "border-emerald-500/20 text-emerald-400 bg-emerald-950/10 hover:bg-emerald-950/20", activeColor: "border-emerald-500 text-slate-950 bg-emerald-400" },
            { id: "high-traffic", name: "Seasonal Spike", color: "border-indigo-500/20 text-indigo-400 bg-indigo-950/10 hover:bg-indigo-950/20", activeColor: "border-indigo-500 text-slate-950 bg-indigo-400" },
            { id: "degraded", name: "Latency Degradation", color: "border-amber-500/20 text-amber-400 bg-amber-950/10 hover:bg-amber-950/20", activeColor: "border-amber-500 text-slate-950 bg-amber-400" },
            { id: "outage", name: "Database Outage", color: "border-rose-500/20 text-rose-400 bg-rose-950/10 hover:bg-rose-950/20", activeColor: "border-rose-500 text-slate-950 bg-rose-400" },
          ].map(item => {
            const isCurrent = currentWorkload === item.id;
            return (
              <button
                key={item.id}
                onClick={() => onSetWorkload(item.id)}
                className={`py-3 px-4 rounded-lg border font-sans font-semibold text-xs text-left transition-all duration-300 ${isCurrent ? item.activeColor : item.color}`}
              >
                <span className="flex items-center gap-2">
                  <Radio size={14} className={isCurrent ? "animate-pulse" : ""} />
                  {item.name}
                </span>
              </button>
            );
          })}
        </div>

        <div className="mt-4 p-3 bg-slate-900/40 rounded-lg border border-slate-900 flex items-start gap-2">
          <Activity size={14} className="text-indigo-400 mt-0.5 flex-shrink-0 animate-pulse" />
          <p className="text-xs text-slate-400 font-sans leading-relaxed">
            <span className="font-semibold text-slate-300 uppercase font-mono text-[10px] mr-1.5 bg-slate-800 px-1.5 py-0.5 rounded">Active Workload</span>
            {getWorkloadDesc(currentWorkload)}
          </p>
        </div>
      </div>

      {/* Live Operational Metrics Widgets Grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="bg-slate-950 border border-slate-800 rounded-xl p-4 flex flex-col justify-between">
          <span className="text-[10px] text-slate-500 font-sans uppercase font-bold tracking-wider flex items-center gap-1.5">
            <TrendingUp size={12} className="text-indigo-400" /> Platform Throughput
          </span>
          <div className="mt-2 flex items-baseline gap-1.5">
            <span className="font-mono text-2xl font-bold text-white">{latest.rps.toLocaleString()}</span>
            <span className="text-[10px] text-slate-400 font-sans font-semibold">RPS</span>
          </div>
          <span className="text-[9px] text-slate-500 font-sans mt-1">Live requests per second</span>
        </div>

        <div className="bg-slate-950 border border-slate-800 rounded-xl p-4 flex flex-col justify-between">
          <span className="text-[10px] text-slate-500 font-sans uppercase font-bold tracking-wider flex items-center gap-1.5">
            <AlertOctagon size={12} className="text-rose-400" /> Error Rate
          </span>
          <div className="mt-2 flex items-baseline gap-1.5">
            <span className={`font-mono text-2xl font-bold ${latest.errorRate > 5 ? "text-rose-400" : "text-emerald-400"}`}>
              {latest.errorRate.toFixed(latest.errorRate > 0 ? 2 : 0)}%
            </span>
          </div>
          <span className="text-[9px] text-slate-500 font-sans mt-1">HTTP 5xx & 4xx errors ratio</span>
        </div>

        <div className="bg-slate-950 border border-slate-800 rounded-xl p-4 flex flex-col justify-between">
          <span className="text-[10px] text-slate-500 font-sans uppercase font-bold tracking-wider flex items-center gap-1.5">
            <Zap size={12} className="text-amber-400" /> Latency (p99)
          </span>
          <div className="mt-2 flex items-baseline gap-1.5">
            <span className={`font-mono text-2xl font-bold ${latest.latencyP99 > 800 ? "text-amber-400" : "text-white"}`}>
              {Math.round(latest.latencyP99)}
            </span>
            <span className="text-[10px] text-slate-400 font-sans font-semibold">ms</span>
          </div>
          <span className="text-[9px] text-slate-500 font-sans mt-1">99th-percentile response time</span>
        </div>

        <div className="bg-slate-950 border border-slate-800 rounded-xl p-4 flex flex-col justify-between">
          <span className="text-[10px] text-slate-500 font-sans uppercase font-bold tracking-wider flex items-center gap-1.5">
            <Server size={12} className="text-emerald-400" /> Host Resources
          </span>
          <div className="mt-2 flex items-baseline gap-1.5 justify-between w-full">
            <span className="font-mono text-xs font-semibold text-slate-300">
              CPU: <span className="text-white font-bold">{Math.round(latest.cpuUsage)}%</span>
            </span>
            <span className="font-mono text-xs font-semibold text-slate-300">
              MEM: <span className="text-white font-bold">{Math.round(latest.memUsage)}%</span>
            </span>
          </div>
          <span className="text-[9px] text-slate-500 font-sans mt-1">Average cluster resource requests</span>
        </div>
      </div>

      {/* Live Recharts Diagrams */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* RPS & Errors Area Chart */}
        <div className="bg-slate-950 rounded-xl border border-slate-800 p-5 flex flex-col h-[340px]">
          <h4 className="font-display font-semibold text-slate-200 text-xs uppercase tracking-wider mb-4">Throughput & Error telemetry</h4>
          <div className="flex-1 min-h-0 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={telemetryHistory} margin={{ top: 5, right: 5, left: -20, bottom: 0 }}>
                <defs>
                  <linearGradient id="colorRps" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#6366f1" stopOpacity={0.25} />
                    <stop offset="95%" stopColor="#6366f1" stopOpacity={0.0} />
                  </linearGradient>
                  <linearGradient id="colorError" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#f43f5e" stopOpacity={0.25} />
                    <stop offset="95%" stopColor="#f43f5e" stopOpacity={0.0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#0f172a" vertical={false} />
                <XAxis dataKey="time" stroke="#475569" fontSize={10} fontFamily="monospace" tickLine={false} />
                <YAxis yAxisId="left" stroke="#6366f1" fontSize={10} fontFamily="monospace" tickLine={false} />
                <YAxis yAxisId="right" orientation="right" stroke="#f43f5e" fontSize={10} fontFamily="monospace" tickLine={false} />
                <Tooltip content={<CustomTooltip />} />
                <Area yAxisId="left" type="monotone" dataKey="rps" name="Throughput" stroke="#6366f1" strokeWidth={2} fillOpacity={1} fill="url(#colorRps)" />
                <Area yAxisId="right" type="monotone" dataKey="errorRate" name="Error Rate" stroke="#f43f5e" strokeWidth={2} fillOpacity={1} fill="url(#colorError)" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Latency Distribution Histogram */}
        <div className="bg-slate-950 rounded-xl border border-slate-800 p-5 flex flex-col h-[340px]">
          <h4 className="font-display font-semibold text-slate-200 text-xs uppercase tracking-wider mb-4">Response Latency Percentiles (ms)</h4>
          <div className="flex-1 min-h-0 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={telemetryHistory.slice(-15)} margin={{ top: 5, right: 5, left: -20, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#0f172a" vertical={false} />
                <XAxis dataKey="time" stroke="#475569" fontSize={10} fontFamily="monospace" tickLine={false} />
                <YAxis stroke="#94a3b8" fontSize={10} fontFamily="monospace" tickLine={false} />
                <Tooltip content={<CustomTooltip />} />
                <Legend iconType="circle" wrapperStyle={{ fontSize: 10, fontFamily: "sans-serif", paddingTop: 10 }} />
                <Bar dataKey="latencyP50" name="p50 Latency" fill="#10b981" radius={[4, 4, 0, 0]} />
                <Bar dataKey="latencyP95" name="p95 Latency" fill="#f59e0b" radius={[4, 4, 0, 0]} />
                <Bar dataKey="latencyP99" name="p99 Latency" fill="#e11d48" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>
    </div>
  );
}
