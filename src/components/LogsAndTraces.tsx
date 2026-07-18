import React, { useState, useEffect, useRef } from "react";
import { LogLine, Trace, TraceSpan } from "../types";
import { ListFilter, Search, Terminal, Workflow, Clock, Layers, Bug, AlertCircle, CheckCircle } from "lucide-react";

interface LogsTracesProps {
  logs: LogLine[];
  traces: Trace[];
  onTriggerEvent: (msg: string, type: "info" | "warn" | "error") => void;
}

export default function LogsAndTraces({ logs, traces, onTriggerEvent }: LogsTracesProps) {
  const [activeTab, setActiveTab] = useState<"traces" | "logs">("traces");
  const [prettyPrint, setPrettyPrint] = useState<boolean>(true);
  const [logFilterLevel, setLogFilterLevel] = useState<string>("all");
  const [logFilterService, setLogFilterService] = useState<string>("all");
  const [logSearchQuery, setLogSearchQuery] = useState<string>("");
  const [selectedTraceId, setSelectedTraceId] = useState<string | null>(null);
  const [selectedSpanId, setSelectedSpanId] = useState<string | null>(null);

  const logsEndRef = useRef<HTMLDivElement>(null);

  // Auto-scroll logs terminal
  useEffect(() => {
    if (activeTab === "logs" && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [logs, activeTab]);

  // Expand first trace by default if none selected
  useEffect(() => {
    if (traces.length > 0 && !selectedTraceId) {
      setSelectedTraceId(traces[0].id);
    }
  }, [traces, selectedTraceId]);

  const selectedTrace = traces.find(t => t.id === selectedTraceId);

  // Filters for Log lines
  const filteredLogs = logs.filter(line => {
    if (logFilterLevel !== "all" && line.level !== logFilterLevel) return false;
    if (logFilterService !== "all" && line.service !== logFilterService) return false;
    if (logSearchQuery.trim()) {
      const q = logSearchQuery.toLowerCase();
      return (
        line.msg.toLowerCase().includes(q) ||
        line.service.toLowerCase().includes(q) ||
        (line.trace_id && line.trace_id.toLowerCase().includes(q))
      );
    }
    return true;
  });

  const getLogLevelStyle = (level: string) => {
    switch (level) {
      case "info": return "text-indigo-400 bg-indigo-950/20";
      case "warn": return "text-amber-400 bg-amber-950/20";
      case "error": return "text-rose-400 bg-rose-950/20";
      case "debug": return "text-slate-400 bg-slate-800/20";
      default: return "text-slate-300";
    }
  };

  const getStatusColor = (code: number) => {
    if (code >= 200 && code < 300) return "text-emerald-400 bg-emerald-950/30 border-emerald-900/50";
    if (code >= 400 && code < 500) return "text-amber-400 bg-amber-950/30 border-amber-900/50";
    return "text-rose-400 bg-rose-950/30 border-rose-900/50";
  };

  return (
    <div className="grid grid-cols-1 xl:grid-cols-3 gap-6 h-full" id="nexuscore-logs-traces-root">
      {/* Selector sidebar (Left Panel: Traces list or Log Filters) */}
      <div className="bg-slate-950 rounded-xl border border-slate-800 p-5 flex flex-col h-[560px]">
        {/* Navigation Tabs */}
        <div className="flex bg-slate-900 p-1 rounded-lg border border-slate-800 mb-4">
          <button
            onClick={() => setActiveTab("traces")}
            className={`flex-1 py-2 text-xs font-sans font-semibold rounded-md transition-all flex items-center justify-center gap-2 ${activeTab === "traces" ? "bg-slate-950 text-white shadow" : "text-slate-400 hover:text-slate-200"}`}
          >
            <Workflow size={14} />
            Distributed Tracing
          </button>
          <button
            onClick={() => setActiveTab("logs")}
            className={`flex-1 py-2 text-xs font-sans font-semibold rounded-md transition-all flex items-center justify-center gap-2 ${activeTab === "logs" ? "bg-slate-950 text-white shadow" : "text-slate-400 hover:text-slate-200"}`}
          >
            <Terminal size={14} />
            Structured Logs
          </button>
        </div>

        {activeTab === "traces" ? (
          /* Traces List Panel */
          <div className="flex-1 flex flex-col min-h-0">
            <div className="mb-3">
              <span className="text-[10px] text-slate-500 uppercase tracking-wider font-bold">Trace Collector Spans</span>
              <p className="text-xs text-slate-400 mt-0.5">Select a trace to open distributed spans flow</p>
            </div>

            <div className="flex-1 overflow-y-auto space-y-2 pr-1">
              {traces.map(trace => {
                const isSelected = trace.id === selectedTraceId;
                const hasError = trace.spans.some(s => s.status === "ERROR");

                return (
                  <button
                    key={trace.id}
                    onClick={() => {
                      setSelectedTraceId(trace.id);
                      setSelectedSpanId(null);
                    }}
                    className={`w-full text-left p-3 rounded-lg border transition-all duration-200 ${isSelected ? "bg-indigo-950/20 border-indigo-500/50 shadow-md" : "bg-slate-900/40 border-slate-900 hover:border-slate-800"}`}
                  >
                    <div className="flex items-center justify-between mb-1.5">
                      <span className="font-mono text-[10px] text-slate-500">{trace.id.substring(0, 8)}</span>
                      <span className="font-mono text-[10px] text-slate-400 flex items-center gap-1">
                        <Clock size={10} /> {trace.totalDuration}ms
                      </span>
                    </div>

                    <div className="flex items-center justify-between gap-2">
                      <div className="truncate">
                        <span className="font-mono text-xs font-bold text-slate-200 uppercase bg-slate-800 px-1.5 py-0.5 rounded mr-1.5">{trace.method}</span>
                        <span className="font-mono text-xs text-slate-300 truncate">{trace.path}</span>
                      </div>
                      <span className={`font-mono text-xs font-bold px-1.5 py-0.5 rounded border ${getStatusColor(trace.statusCode)}`}>
                        {trace.statusCode}
                      </span>
                    </div>

                    {hasError && (
                      <div className="mt-2 flex items-center gap-1 text-[10px] text-rose-400 font-sans">
                        <Bug size={10} /> Internal span exceptions raised
                      </div>
                    )}
                  </button>
                );
              })}
            </div>
          </div>
        ) : (
          /* Logs Filters Panel */
          <div className="flex-1 flex flex-col min-h-0 space-y-4">
            <div>
              <span className="text-[10px] text-slate-500 uppercase tracking-wider font-bold">Standard Stream Filters</span>
              <p className="text-xs text-slate-400 mt-0.5">Filter structured system logs output</p>
            </div>

            {/* Free Search */}
            <div className="relative">
              <Search className="absolute left-3 top-2.5 h-4 w-4 text-slate-500" />
              <input
                type="text"
                placeholder="Search log fields or trace IDs..."
                value={logSearchQuery}
                onChange={e => setLogSearchQuery(e.target.value)}
                className="w-full bg-slate-900 border border-slate-800 rounded-lg pl-9 pr-4 py-2 text-xs text-slate-300 placeholder-slate-500 focus:outline-none focus:border-indigo-500/50 font-mono"
              />
            </div>

            {/* Severity Filter */}
            <div className="space-y-1.5">
              <label className="text-[10px] text-slate-500 uppercase tracking-wider font-bold block">Severity Level</label>
              <div className="grid grid-cols-2 gap-2">
                {[
                  { id: "all", name: "All Levels" },
                  { id: "info", name: "INFO" },
                  { id: "warn", name: "WARN" },
                  { id: "error", name: "ERROR" },
                ].map(item => (
                  <button
                    key={item.id}
                    onClick={() => setLogFilterLevel(item.id)}
                    className={`py-1.5 px-3 rounded text-[10px] font-mono border text-center transition-colors ${logFilterLevel === item.id ? "bg-indigo-500/10 text-indigo-400 border-indigo-500/30" : "bg-slate-900/60 border-slate-900 text-slate-400 hover:bg-slate-900"}`}
                  >
                    {item.name}
                  </button>
                ))}
              </div>
            </div>

            {/* Service Filter */}
            <div className="space-y-1.5">
              <label className="text-[10px] text-slate-500 uppercase tracking-wider font-bold block">Source Service</label>
              <div className="space-y-1.5 overflow-y-auto max-h-[180px] pr-1">
                {[
                  { id: "all", name: "All Microservices" },
                  { id: "api-gateway", name: "api-gateway" },
                  { id: "auth-service", name: "auth-service" },
                  { id: "compute-engine", name: "compute-engine" },
                  { id: "postgres-db", name: "postgres-db" },
                  { id: "telemetry-collector", name: "telemetry-collector" }
                ].map(item => (
                  <button
                    key={item.id}
                    onClick={() => setLogFilterService(item.id)}
                    className={`w-full text-left py-1.5 px-3 rounded text-[10px] font-mono border transition-colors ${logFilterService === item.id ? "bg-indigo-500/10 text-indigo-400 border-indigo-500/30" : "bg-slate-900/60 border-slate-900 text-slate-400 hover:bg-slate-900"}`}
                  >
                    {item.name}
                  </button>
                ))}
              </div>
            </div>

            {/* Output formatting settings */}
            <div className="pt-4 border-t border-slate-900">
              <label className="text-[10px] text-slate-500 uppercase tracking-wider font-bold block mb-2">Output Format</label>
              <div className="flex bg-slate-900 p-1 rounded-lg border border-slate-800">
                <button
                  onClick={() => setPrettyPrint(true)}
                  className={`flex-1 py-1 text-[10px] font-mono rounded transition-all ${prettyPrint ? "bg-slate-950 text-white" : "text-slate-400"}`}
                >
                  PRETTY PRINT
                </button>
                <button
                  onClick={() => setPrettyPrint(false)}
                  className={`flex-1 py-1 text-[10px] font-mono rounded transition-all ${!prettyPrint ? "bg-slate-950 text-white" : "text-slate-400"}`}
                >
                  RAW JSON
                </button>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Main Terminal/Interactive Waterfall Content (Right Panel) */}
      <div className="xl:col-span-2 bg-slate-950 rounded-xl border border-slate-800 p-5 flex flex-col h-[560px] relative">
        {activeTab === "traces" ? (
          /* Traces Visualization Panel */
          selectedTrace ? (
            <div className="flex-1 flex flex-col min-h-0">
              {/* Trace details header */}
              <div className="border-b border-slate-900 pb-3 mb-4 flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-sm font-bold text-white uppercase bg-slate-800 px-2 py-0.5 rounded">{selectedTrace.method}</span>
                    <span className="font-mono text-sm text-slate-300 font-bold">{selectedTrace.path}</span>
                  </div>
                  <div className="flex items-center gap-4 mt-1.5 text-xs text-slate-400">
                    <span className="font-mono text-[11px]">TraceID: <span className="text-indigo-400 font-bold">{selectedTrace.id}</span></span>
                    <span className="flex items-center gap-1 font-mono text-[11px]"><Layers size={12} /> {selectedTrace.spans.length} Spans</span>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <span className={`font-mono text-xs font-bold px-2 py-1 rounded border ${getStatusColor(selectedTrace.statusCode)}`}>
                    HTTP {selectedTrace.statusCode}
                  </span>
                  <span className="font-mono text-xs text-slate-400 bg-slate-900 px-2.5 py-1 rounded border border-slate-800 flex items-center gap-1">
                    <Clock size={12} /> {selectedTrace.totalDuration}ms
                  </span>
                </div>
              </div>

              {/* Spans Waterfall Section */}
              <div className="flex-1 overflow-y-auto space-y-3 pr-1">
                <span className="text-[10px] text-slate-500 uppercase tracking-wider font-bold block mb-2">OTel Tracer Spans Waterfall</span>

                <div className="space-y-2 border-l border-slate-900 pl-2">
                  {selectedTrace.spans.map((span, idx) => {
                    const traceStartOffset = span.startTime;
                    const spanDuration = span.duration;
                    const traceTotalDuration = selectedTrace.totalDuration;

                    // Calculate horizontal percentages for custom visual waterfall Gantt chart
                    const leftPercent = (traceStartOffset / traceTotalDuration) * 100;
                    const widthPercent = Math.max((spanDuration / traceTotalDuration) * 100, 2); // At least 2% bar width for visual visibility

                    const isSpanSelected = selectedSpanId === span.id;

                    const getSpanColor = (svc: string) => {
                      if (span.status === "ERROR") return "bg-rose-500";
                      switch (svc) {
                        case "api-gateway": return "bg-indigo-500";
                        case "auth-service": return "bg-amber-500";
                        case "compute-engine": return "bg-emerald-500";
                        case "postgres-db": return "bg-purple-500";
                        case "telemetry-collector": return "bg-rose-500";
                        default: return "bg-slate-500";
                      }
                    };

                    return (
                      <div key={span.id} className="space-y-1.5">
                        <div
                          onClick={() => setSelectedSpanId(isSpanSelected ? null : span.id)}
                          className={`grid grid-cols-1 md:grid-cols-3 items-center gap-3 p-2 rounded cursor-pointer transition-all ${isSpanSelected ? "bg-slate-900/60 border border-slate-800" : "hover:bg-slate-900/20"}`}
                        >
                          {/* Span Service & Name */}
                          <div className="truncate flex items-center gap-2">
                            <span className={`h-2 w-2 rounded-full ${getSpanColor(span.service)}`} />
                            <div>
                              <span className="text-[11px] font-mono text-slate-400 font-bold">{span.service}</span>
                              <span className="text-[10px] font-mono text-slate-500 ml-1.5">{span.name}</span>
                            </div>
                          </div>

                          {/* Gantt waterfall line representation */}
                          <div className="relative h-2 bg-slate-900/50 rounded overflow-hidden md:col-span-2">
                            <div
                              className={`absolute h-full rounded transition-all duration-300 ${getSpanColor(span.service)}`}
                              style={{ left: `${leftPercent}%`, width: `${widthPercent}%` }}
                            />
                            <span className="absolute right-2 top-[-3px] text-[8px] font-mono font-bold text-slate-500">{span.duration}ms</span>
                          </div>
                        </div>

                        {/* Expandable span attributes (OTel metadata) */}
                        {isSpanSelected && (
                          <div className="bg-slate-900/80 border border-slate-800 rounded-lg p-3 mx-2 text-[10px] font-mono text-slate-400 space-y-2">
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-2 border-b border-slate-800 pb-2 mb-2">
                              <div><span className="text-slate-500">Span ID:</span> <span className="text-slate-200">{span.id}</span></div>
                              {span.parentId && <div><span className="text-slate-500">Parent ID:</span> <span className="text-slate-200">{span.parentId}</span></div>}
                              <div><span className="text-slate-500">Span Status:</span> <span className={`font-bold ${span.status === "ERROR" ? "text-rose-400" : "text-emerald-400"}`}>{span.status}</span></div>
                              <div><span className="text-slate-500">Library:</span> <span className="text-slate-300">go.opentelemetry.io/otel v1.22.0</span></div>
                            </div>
                            
                            <div>
                              <span className="text-slate-500 block uppercase font-bold text-[8px] tracking-wider mb-1">OTel Attributes</span>
                              <div className="grid grid-cols-1 md:grid-cols-2 gap-x-4 gap-y-1 bg-slate-950 p-2 rounded">
                                {Object.entries(span.attributes).map(([key, val]) => (
                                  <div key={key} className="truncate"><span className="text-indigo-400">{key}:</span> <span className="text-slate-300">{val.toString()}</span></div>
                                ))}
                              </div>
                            </div>

                            {span.events && span.events.length > 0 && (
                              <div className="mt-2">
                                <span className="text-slate-500 block uppercase font-bold text-[8px] tracking-wider mb-1">Events Log Annotations</span>
                                <div className="space-y-1 bg-slate-950 p-2 rounded">
                                  {span.events.map((evt, eIdx) => (
                                    <div key={eIdx} className="flex items-start justify-between border-b border-slate-900/50 pb-1 last:border-0 last:pb-0">
                                      <span className="text-amber-400 font-semibold">{evt.name}</span>
                                      <span className="text-[9px] text-slate-500">{new Date(evt.timestamp).toISOString()}</span>
                                    </div>
                                  ))}
                                </div>
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          ) : (
            <div className="flex-1 flex flex-col items-center justify-center text-slate-500 font-sans">
              <Workflow size={32} className="text-slate-700 mb-2" />
              <p className="text-xs">No active telemetry traces gathered in memory.</p>
            </div>
          )
        ) : (
          /* Logs stdout Terminal View */
          <div className="flex-1 flex flex-col min-h-0">
            <div className="border-b border-slate-900 pb-3 mb-3 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Terminal size={14} className="text-indigo-400 animate-pulse" />
                <span className="text-xs font-mono text-slate-300">stdout // slog JSON logger stream</span>
              </div>
              <span className="text-[10px] font-mono text-slate-500 bg-slate-900 border border-slate-800 px-2 py-0.5 rounded">
                Viewing {filteredLogs.length} matching lines
              </span>
            </div>

            {/* Standard Terminal Window */}
            <div className="flex-1 bg-slate-950/60 border border-slate-900 rounded-lg p-4 font-mono text-[11px] overflow-y-auto space-y-1.5 scrollbar-thin">
              {filteredLogs.map((log, idx) => (
                <div key={idx} className="leading-relaxed hover:bg-slate-900/30 py-0.5 rounded px-1 transition-colors">
                  {prettyPrint ? (
                    /* Human Readable Log Pretty Print format */
                    <div className="flex flex-wrap items-baseline gap-x-2">
                      <span className="text-slate-600 font-normal">[{new Date(log.time).toLocaleTimeString()}]</span>
                      <span className={`font-bold uppercase text-[9px] px-1 rounded ${getLogLevelStyle(log.level)}`}>
                        {log.level}
                      </span>
                      <span className="text-indigo-400 font-semibold">{log.service}:</span>
                      <span className="text-slate-200 font-sans">{log.msg}</span>
                      {log.trace_id && (
                        <span className="text-[9px] text-slate-500">
                          trace_id=<span className="text-slate-400 font-semibold">{log.trace_id.substring(0, 8)}</span>
                        </span>
                      )}
                      {log.attributes && Object.keys(log.attributes).length > 0 && (
                        <span className="text-[9px] text-amber-500">
                          metadata={JSON.stringify(log.attributes)}
                        </span>
                      )}
                    </div>
                  ) : (
                    /* RAW JSON stdout style */
                    <span className="text-slate-300 break-all select-all">
                      {JSON.stringify(log)}
                    </span>
                  )}
                </div>
              ))}
              <div ref={logsEndRef} />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
