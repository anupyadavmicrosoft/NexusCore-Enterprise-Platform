import React, { useState } from "react";
import { Send, Clock, Server, Check, ArrowRight, ShieldAlert, Key } from "lucide-react";

interface PlaygroundProps {
  onTriggerEvent: (msg: string, type: "info" | "warn" | "error") => void;
  onSendMockRequest: (method: string, path: string, body: any) => void;
}

export default function Playground({ onTriggerEvent, onSendMockRequest }: PlaygroundProps) {
  const [selectedEndpoint, setSelectedEndpoint] = useState<string>("POST /transactions");
  const [authHeader, setAuthHeader] = useState<string>("Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJuZXh1c2NvcmUiLCJpYXQiOjE3MTYwMDM2MDB9");
  const [payloadId, setPayloadId] = useState<string>(`tx_${Math.random().toString(36).substring(2, 9)}`);
  const [payloadAmount, setPayloadAmount] = useState<number>(350.50);
  const [payloadUser, setPayloadUser] = useState<string>("user_nexus_09");
  
  // Response states
  const [loading, setLoading] = useState<boolean>(false);
  const [responseStatus, setResponseStatus] = useState<number | null>(null);
  const [responseHeaders, setResponseHeaders] = useState<Record<string, string> | null>(null);
  const [responseBody, setResponseBody] = useState<any | null>(null);
  const [responseMs, setResponseMs] = useState<number | null>(null);

  const endpoints = [
    { id: "POST /transactions", method: "POST", path: "/transactions", service: "api-gateway", desc: "Register a financial transaction payload and persist in Postgres." },
    { id: "GET /transactions/:id", method: "GET", path: "/transactions/", service: "api-gateway", desc: "Retrieve transaction record from PostgreSQL cache/DB state." },
    { id: "POST /auth/token", method: "POST", path: "/auth/token", service: "auth-service", desc: "Acquire secure JWT token for core service API auth headers." },
    { id: "GET /metrics", method: "GET", path: "/metrics", service: "telemetry-collector", desc: "Scrape system level Prometheus metric counters directly." }
  ];

  const currentEndpoint = endpoints.find(ep => ep.id === selectedEndpoint) || endpoints[0];

  const handleSendRequest = () => {
    setLoading(true);
    setResponseStatus(null);
    setResponseBody(null);
    setResponseHeaders(null);
    setResponseMs(null);

    const isGet = currentEndpoint.method === "GET";
    const actualPath = isGet && currentEndpoint.path.includes("transactions") 
      ? `${currentEndpoint.path}${payloadId}` 
      : currentEndpoint.path;

    onTriggerEvent(`Playground Client: Dispatching HTTP request ${currentEndpoint.method} ${actualPath}`, "info");

    // Construct mock payload
    const body = isGet ? undefined : {
      id: payloadId,
      amount: payloadAmount,
      created_by: payloadUser,
    };

    setTimeout(() => {
      let status = 200;
      let bodyData: any = {};
      let headers: Record<string, string> = {
        "Content-Type": "application/json",
        "Server": "NexusCore Gateway / v1.2.0",
        "X-Trace-ID": `tx_${Math.random().toString(36).substring(2, 10)}`,
        "X-Span-ID": `sp_${Math.random().toString(36).substring(2, 10)}`,
        "Date": new Date().toUTCString()
      };
      const duration = Math.floor(Math.random() * 80) + 40; // 40-120ms response time

      // API Logic simulation
      if (selectedEndpoint === "POST /transactions") {
        if (!authHeader) {
          status = 401;
          bodyData = { error: "unauthorized", message: "JWT Authorization header missing or empty" };
        } else if (payloadAmount <= 0) {
          status = 422;
          bodyData = { error: "unprocessable_entity", message: "Validation failure: amount must be positive" };
        } else {
          status = 201;
          bodyData = {
            id: payloadId,
            amount: payloadAmount,
            status: "PENDING",
            created_at: new Date().toISOString(),
            created_by: payloadUser
          };
        }
      } else if (selectedEndpoint === "GET /transactions/:id") {
        status = 200;
        bodyData = {
          id: payloadId,
          amount: payloadAmount,
          status: "SUCCESSFUL",
          created_at: new Date(Date.now() - 3600000).toISOString(),
          created_by: payloadUser
        };
      } else if (selectedEndpoint === "POST /auth/token") {
        status = 200;
        bodyData = {
          access_token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJuZXh1c2NvcmUiLCJpYXQiOjE3MTYwMDM2MDB9",
          token_type: "Bearer",
          expires_in: 3600
        };
      } else if (selectedEndpoint === "GET /metrics") {
        status = 200;
        headers["Content-Type"] = "text/plain; version=0.0.4; charset=utf-8";
        bodyData = `# HELP http_requests_total Total number of HTTP requests processed.\n# TYPE http_requests_total counter\nhttp_requests_total{code="201",method="POST",service="payment-service"} 14502\nhttp_requests_total{code="200",method="GET",service="auth-service"} 859332\n\n# HELP http_request_duration_seconds HTTP request latencies buckets.\n# TYPE http_request_duration_seconds histogram\nhttp_request_duration_seconds_bucket{le="0.1",service="payment-service"} 12200\nhttp_request_duration_seconds_bucket{le="0.5",service="payment-service"} 14100\nhttp_request_duration_seconds_sum 124.52\nhttp_request_duration_seconds_count 14502`;
      }

      setResponseStatus(status);
      setResponseBody(bodyData);
      setResponseHeaders(headers);
      setResponseMs(duration);
      setLoading(false);

      // Propagate mock request triggers to generate server traces/logs automatically
      onSendMockRequest(currentEndpoint.method, actualPath, body);
      onTriggerEvent(`Playground Client: Received HTTP response ${status} in ${duration}ms`, status >= 400 ? "warn" : "info");

    }, 800);
  };

  const handleRegenId = () => {
    setPayloadId(`tx_${Math.random().toString(36).substring(2, 9)}`);
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 h-full" id="nexuscore-playground-root">
      {/* Interactive API Request Form */}
      <div className="bg-slate-950 rounded-xl border border-slate-800 p-5 flex flex-col h-[520px] overflow-y-auto">
        <div className="border-b border-slate-900 pb-3 mb-4">
          <h3 className="font-display font-semibold text-white tracking-wide">Interactive API Playground</h3>
          <p className="text-xs text-slate-400 font-sans mt-0.5">Issue live HTTP calls against the distributed mesh and watch OTel tracing live</p>
        </div>

        <div className="space-y-4 flex-1">
          {/* Endpoint Selector */}
          <div className="space-y-1">
            <label className="text-[10px] text-slate-500 uppercase tracking-wider font-bold block">Target API Endpoint</label>
            <div className="flex gap-2">
              <select
                value={selectedEndpoint}
                onChange={e => setSelectedEndpoint(e.target.value)}
                className="w-full bg-slate-900 border border-slate-800 rounded-lg px-3 py-2.5 text-xs text-slate-200 font-mono focus:outline-none focus:border-indigo-500/50"
              >
                {endpoints.map(ep => (
                  <option key={ep.id} value={ep.id}>
                    {ep.method} {ep.path}
                  </option>
                ))}
              </select>
            </div>
            <span className="text-[10px] text-slate-500 font-sans italic leading-relaxed mt-1 block">
              {currentEndpoint.desc}
            </span>
          </div>

          {/* Secure Headers Section */}
          <div className="bg-slate-900/40 p-3 rounded-lg border border-slate-900 space-y-3">
            <div className="flex items-center gap-1 text-[10px] text-slate-400 font-sans uppercase font-bold tracking-wider">
              <Key size={12} className="text-amber-400" /> HTTP Request Headers
            </div>
            <div className="space-y-2">
              <div className="grid grid-cols-3 gap-2 items-center">
                <span className="text-[10px] font-mono text-slate-500">Authorization</span>
                <input
                  type="text"
                  value={authHeader}
                  onChange={e => setAuthHeader(e.target.value)}
                  className="bg-slate-950 border border-slate-800 text-[10px] font-mono text-slate-300 rounded px-2 py-1.5 col-span-2 focus:outline-none"
                  placeholder="Bearer <JWT TOKEN>"
                />
              </div>
              <div className="grid grid-cols-3 gap-2 items-center">
                <span className="text-[10px] font-mono text-slate-500">Accept</span>
                <span className="text-[10px] font-mono text-slate-400 bg-slate-950 border border-slate-900 rounded px-2 py-1.5 col-span-2">application/json</span>
              </div>
            </div>
          </div>

          {/* Body Parameters (Contextual) */}
          {currentEndpoint.method === "POST" && selectedEndpoint === "POST /transactions" && (
            <div className="space-y-3">
              <label className="text-[10px] text-slate-500 uppercase tracking-wider font-bold block">HTTP JSON Body Parameters</label>
              
              <div className="grid grid-cols-3 gap-3">
                <div className="space-y-1">
                  <span className="text-[9px] font-mono text-slate-400">ID</span>
                  <div className="flex bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
                    <input
                      type="text"
                      value={payloadId}
                      onChange={e => setPayloadId(e.target.value)}
                      className="bg-transparent text-[10px] font-mono text-slate-200 px-2 py-1.5 flex-1 min-w-0 focus:outline-none"
                    />
                    <button onClick={handleRegenId} className="px-1.5 bg-slate-800 text-slate-400 text-[9px] font-bold font-sans border-l border-slate-700 hover:bg-slate-700">
                      GEN
                    </button>
                  </div>
                </div>

                <div className="space-y-1">
                  <span className="text-[9px] font-mono text-slate-400">Amount ($)</span>
                  <input
                    type="number"
                    value={payloadAmount}
                    onChange={e => setPayloadAmount(parseFloat(e.target.value) || 0)}
                    className="w-full bg-slate-900 border border-slate-800 rounded-lg px-2 py-1.5 text-[10px] font-mono text-slate-200 focus:outline-none focus:border-indigo-500/50"
                  />
                </div>

                <div className="space-y-1">
                  <span className="text-[9px] font-mono text-slate-400">Created By</span>
                  <input
                    type="text"
                    value={payloadUser}
                    onChange={e => setPayloadUser(e.target.value)}
                    className="w-full bg-slate-900 border border-slate-800 rounded-lg px-2 py-1.5 text-[10px] font-mono text-slate-200 focus:outline-none focus:border-indigo-500/50"
                  />
                </div>
              </div>
            </div>
          )}

          {currentEndpoint.method === "GET" && selectedEndpoint === "GET /transactions/:id" && (
            <div className="bg-slate-900/20 p-3 border border-slate-900 rounded-lg space-y-1.5">
              <span className="text-[10px] text-slate-500 uppercase tracking-wider font-bold block">Route path variable bindings</span>
              <div className="flex items-center gap-2 font-mono text-xs">
                <span className="text-slate-400">GET /transactions/</span>
                <span className="text-indigo-400 font-bold bg-indigo-950/30 px-1.5 py-0.5 rounded">{payloadId}</span>
              </div>
            </div>
          )}
        </div>

        <button
          onClick={handleSendRequest}
          disabled={loading}
          className="w-full mt-4 py-2.5 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white rounded-lg font-sans font-bold text-xs transition-all flex items-center justify-center gap-1.5 shadow-md hover:shadow-indigo-600/10"
        >
          {loading ? <Clock size={14} className="animate-spin" /> : <Send size={14} />}
          {loading ? "DISPATCHING PACKETS..." : "SEND HTTP REQUEST"}
        </button>
      </div>

      {/* Response Display Box */}
      <div className="bg-slate-950 rounded-xl border border-slate-800 p-5 flex flex-col h-[520px] relative">
        <div className="border-b border-slate-900 pb-3 mb-4 flex items-center justify-between">
          <span className="text-[10px] text-slate-400 uppercase tracking-wider font-bold font-mono">Response console</span>
          {responseMs && (
            <span className="text-[10px] font-mono text-slate-500 flex items-center gap-1">
              <Clock size={11} /> Roundtrip latency: <span className="text-emerald-400 font-bold">{responseMs}ms</span>
            </span>
          )}
        </div>

        {loading ? (
          /* Sending Request loading visualization */
          <div className="flex-1 flex flex-col items-center justify-center text-slate-500 text-center gap-2 font-sans animate-pulse">
            <Server size={32} className="text-indigo-400" />
            <p className="text-xs">Connecting to ingress gateway gateway-service:8080...</p>
            <div className="flex items-center gap-1 text-[10px] text-slate-600 font-mono mt-2">
              <span>TCP handshake</span>
              <ArrowRight size={10} />
              <span>Proxy routing</span>
              <ArrowRight size={10} />
              <span>TLS handshake</span>
            </div>
          </div>
        ) : responseStatus ? (
          /* Response Display */
          <div className="flex-1 flex flex-col min-h-0 space-y-4">
            <div className="flex items-center gap-3">
              <span className={`font-mono text-xs font-bold px-2.5 py-1 rounded border ${responseStatus >= 200 && responseStatus < 300 ? "bg-emerald-950/20 text-emerald-400 border-emerald-900/50" : "bg-rose-950/20 text-rose-400 border-rose-900/50"}`}>
                STATUS {responseStatus}
              </span>
              <span className="text-[10px] font-mono text-slate-500">application/json</span>
            </div>

            {/* Split view: Headers (Top) and Body (Bottom) */}
            <div className="flex-1 flex flex-col min-h-0 space-y-3">
              {/* Response Headers */}
              {responseHeaders && (
                <div className="space-y-1">
                  <span className="text-[9px] text-slate-500 uppercase font-bold tracking-wider font-sans">HTTP Response Headers</span>
                  <div className="bg-slate-900/50 border border-slate-900 rounded p-2 text-[9px] font-mono text-slate-400 max-h-[85px] overflow-y-auto">
                    {Object.entries(responseHeaders).map(([key, val]) => (
                      <div key={key} className="truncate"><span className="text-slate-500">{key}:</span> <span className="text-slate-300">{val}</span></div>
                    ))}
                  </div>
                </div>
              )}

              {/* Response Body */}
              <div className="flex-1 flex flex-col min-h-0">
                <span className="text-[9px] text-slate-500 uppercase font-bold tracking-wider font-sans mb-1">JSON Response Payload Body</span>
                <div className="flex-1 bg-slate-900/20 border border-slate-900 rounded p-3 font-mono text-[10px] text-slate-300 overflow-y-auto select-all selection:bg-slate-800">
                  {typeof responseBody === "string" ? (
                    <pre className="whitespace-pre-wrap">{responseBody}</pre>
                  ) : (
                    <pre>{JSON.stringify(responseBody, null, 2)}</pre>
                  )}
                </div>
              </div>
            </div>
          </div>
        ) : (
          /* Empty/Landing State */
          <div className="flex-1 flex flex-col items-center justify-center text-slate-500 font-sans text-center max-w-[280px] mx-auto gap-1">
            <Server size={32} className="text-slate-700 mb-1" />
            <h4 className="font-semibold text-slate-300 text-xs">Awaiting client request dispatch</h4>
            <p className="text-[10px] text-slate-400 leading-relaxed">
              Define the query parameters in the left-hand panel and click "Send HTTP Request" to send packets to the local Kubernetes service clusters.
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
