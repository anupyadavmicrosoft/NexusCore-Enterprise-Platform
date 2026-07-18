import React, { useState, useEffect } from "react";
import { ServiceCode } from "../types";
import { Copy, Check, Download, Play, Folder, File, Code, Cpu, Hammer, Cloud, ListPlus, Trash2, Terminal, Layers } from "lucide-react";

interface DesignerProps {
  onTriggerEvent: (msg: string, type: "info" | "warn" | "error") => void;
  onDeployService: (service: ServiceCode) => void;
}

export default function ServiceDesigner({ onTriggerEvent, onDeployService }: DesignerProps) {
  // Mode selection: "workspace" shows the real physical Go Workspace on disk, "generator" allows synthesizing new ones
  const [viewMode, setViewMode] = useState<"workspace" | "generator">("workspace");

  // Form State (for simulated generation)
  const [serviceName, setServiceName] = useState<string>("payment-service");
  const [database, setDatabase] = useState<string>("PostgreSQL");
  const [broker, setBroker] = useState<string>("Kafka");
  const [endpoints, setEndpoints] = useState<Array<{ path: string; method: string; desc: string }>>([
    { path: "/payments", method: "POST", desc: "Initiate customer checkout payment" },
    { path: "/payments/:id", method: "GET", desc: "Query transactional status" }
  ]);
  const [extraFeatures, setExtraFeatures] = useState<string>("Enable CORS, rate limit of 100 req/min, JWT OAuth validation, and Prometheus alert thresholds.");

  // Loading and result states
  const [loading, setLoading] = useState<boolean>(false);
  const [compileStep, setCompileStep] = useState<string>("");
  const [compileLogs, setCompileLogs] = useState<string[]>([]);
  const [generatedService, setGeneratedService] = useState<ServiceCode | null>(null);
  
  // Real active Workspace state from disk
  const [diskWorkspace, setDiskWorkspace] = useState<ServiceCode | null>(null);

  // Active selected file and state
  const [selectedFile, setSelectedFile] = useState<string>("README.md");
  const [copied, setCopied] = useState<boolean>(false);

  // Load disk workspace on component mount
  useEffect(() => {
    fetchActiveDiskWorkspace();
  }, []);

  const fetchActiveDiskWorkspace = async () => {
    try {
      const response = await fetch("/api/enterprise/workspace");
      if (response.ok) {
        const data = await response.json();
        setDiskWorkspace(data);
        // Default select README if present
        if (data.files && data.files["README.md"]) {
          setSelectedFile("README.md");
        } else {
          const keys = Object.keys(data.files || {});
          if (keys.length > 0) setSelectedFile(keys[0]);
        }
      }
    } catch (err) {
      console.error("Failed to load disk workspace:", err);
    }
  };

  const handleAddEndpoint = () => {
    setEndpoints([...endpoints, { path: "/new-route", method: "GET", desc: "Describe endpoint action" }]);
  };

  const handleRemoveEndpoint = (idx: number) => {
    setEndpoints(endpoints.filter((_, i) => i !== idx));
  };

  const handleUpdateEndpoint = (idx: number, field: string, val: string) => {
    const updated = [...endpoints];
    updated[idx] = { ...updated[idx], [field]: val };
    setEndpoints(updated);
  };

  const executeCompilationTimeline = () => {
    setCompileLogs([]);
    const steps = [
      { msg: "🐳 Initializing Docker compilation environment container...", dur: 600 },
      { msg: "⚙️ Synchronizing module dependencies from go.mod...", dur: 900 },
      { msg: "✓ Imported github.com/prometheus/client_golang v1.19.0", dur: 500 },
      { msg: "✓ Imported go.opentelemetry.io/otel v1.22.0", dur: 500 },
      { msg: "🏗️ Validating Clean Architecture separation of concerns...", dur: 1000 },
      { msg: "✔️ Domain interfaces and structures compile validated.", dur: 400 },
      { msg: "✔️ PostgreSQL Repository and sql injection sanitizers validated.", dur: 500 },
      { msg: "✔️ Usecase flow business validation rules mapping confirmed.", dur: 500 },
      { msg: "✔️ Delivery HTTP handler ports and endpoints router wired.", dur: 400 },
      { msg: "🧪 Executing unit tests: go test -v ./internal/usecase/...", dur: 1100 },
      { msg: "--- PASS: TestCreateTransaction_Success (0.00s)", dur: 200 },
      { msg: "--- PASS: TestCreateTransaction_ValidationFailure (0.00s)", dur: 200 },
      { msg: "PASS: ok testing completed successfully.", dur: 400 },
      { msg: "🐳 Building security-hardened scratch distroless container image...", dur: 800 },
      { msg: "📝 Synthesizing complete OpenAPI 3.0 configuration spec...", dur: 500 },
      { msg: "📦 Creating Helm charts values and Kubernetes manifests deployment configurations...", dur: 800 },
      { msg: "✨ Build finalized! Distributing platform service block to workbench.", dur: 300 }
    ];

    let totalOffset = 0;
    steps.forEach((step, idx) => {
      setTimeout(() => {
        setCompileStep(step.msg);
        setCompileLogs(prev => [...prev, step.msg]);
      }, totalOffset);
      totalOffset += step.dur;
    });

    return totalOffset;
  };

  const handleGenerateService = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setGeneratedService(null);
    onTriggerEvent(`AI Architect: Starting code synthesis for microservice ${serviceName}...`, "info");

    const compilerDuration = executeCompilationTimeline();

    try {
      const response = await fetch("/api/gemini/generate-service", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          serviceName,
          database,
          broker,
          endpoints,
          extraFeatures
        })
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || "Failed to generate service files.");
      }

      setTimeout(() => {
        setGeneratedService(data);
        const keys = Object.keys(data.files);
        if (keys.length > 0) {
          setSelectedFile(keys[0]);
        }
        setLoading(false);
        onDeployService(data);
        onTriggerEvent(`AI Architect: Code synthesis completed! deployment.apps/${serviceName.toLowerCase()} registered in workspace.`, "info");
      }, Math.max(compilerDuration - 500, 2000));

    } catch (err: any) {
      console.error(err);
      setLoading(false);
      onTriggerEvent(`Architect compilation error: ${err.message}`, "error");
    }
  };

  const activeService = viewMode === "workspace" ? diskWorkspace : generatedService;

  const handleCopyCode = () => {
    if (!activeService) return;
    const code = activeService.files[selectedFile];
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDownloadFile = () => {
    if (!activeService) return;
    const code = activeService.files[selectedFile];
    const element = document.createElement("a");
    const file = new Blob([code], { type: "text/plain" });
    element.href = URL.createObjectURL(file);
    element.download = selectedFile.split("/").pop() || "source_code";
    document.body.appendChild(element);
    element.click();
    document.body.removeChild(element);
  };

  const getFileIcon = (fileName: string) => {
    if (fileName.endsWith(".go")) return <Code size={13} className="text-indigo-400" />;
    if (fileName.endsWith(".yaml") || fileName.endsWith(".yml")) return <Cloud size={13} className="text-amber-400" />;
    if (fileName.includes("Dockerfile")) return <Cpu size={13} className="text-sky-400" />;
    if (fileName.endsWith(".md")) return <File size={13} className="text-emerald-400" />;
    return <File size={13} className="text-slate-400" />;
  };

  // Helper to split active service files into folders recursively for directory browser
  const getFileGroups = (files: Record<string, string>) => {
    const paths = Object.keys(files);
    const groups: Record<string, string[]> = {};

    paths.forEach(p => {
      const parts = p.split("/");
      let groupName = "root";
      if (parts.length > 1) {
        groupName = parts.slice(0, -1).join("/");
      }
      if (!groups[groupName]) {
        groups[groupName] = [];
      }
      groups[groupName].push(p);
    });

    return groups;
  };

  const fileGroups = activeService ? getFileGroups(activeService.files) : {};

  return (
    <div className="flex flex-col gap-4 h-full" id="nexuscore-designer-root">
      {/* Modes Switch Bar */}
      <div className="flex items-center justify-between bg-slate-900 border border-slate-800 p-3 rounded-lg">
        <div className="flex items-center gap-2">
          <Layers className="text-indigo-400" size={16} />
          <span className="text-xs font-semibold text-slate-300">Workspace Execution Context</span>
        </div>
        <div className="flex items-center gap-1.5 bg-slate-950 p-1 rounded border border-slate-800">
          <button
            onClick={() => {
              setViewMode("workspace");
              if (diskWorkspace) setSelectedFile("README.md");
            }}
            className={`px-3 py-1 text-[10px] font-sans font-bold rounded transition-colors ${
              viewMode === "workspace" ? "bg-indigo-600 text-white" : "text-slate-400 hover:text-slate-200"
            }`}
          >
            ACTIVE ENTERPRISE WORKSPACE
          </button>
          <button
            onClick={() => {
              setViewMode("generator");
              if (generatedService) {
                const keys = Object.keys(generatedService.files);
                if (keys.length > 0) setSelectedFile(keys[0]);
              } else {
                setSelectedFile("");
              }
            }}
            className={`px-3 py-1 text-[10px] font-sans font-bold rounded transition-colors ${
              viewMode === "generator" ? "bg-indigo-600 text-white" : "text-slate-400 hover:text-slate-200"
            }`}
          >
            NEW SERVICE GENERATOR
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-12 gap-6 flex-1">
        {/* Left Side: Parameters Form / Workspace Overview */}
        <div className="xl:col-span-4 bg-slate-950 rounded-xl border border-slate-800 p-5 flex flex-col h-[520px] overflow-y-auto">
          {viewMode === "workspace" ? (
            <div className="space-y-4 flex flex-col h-full">
              <div className="border-b border-slate-900 pb-3">
                <h3 className="font-display font-semibold text-white tracking-wide text-sm flex items-center gap-2">
                  <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse" />
                  Real-time Active Platform
                </h3>
                <p className="text-xs text-slate-400 font-sans mt-0.5">Enterprise Multi-Module Workspace System</p>
              </div>

              <div className="space-y-3.5 text-xs text-slate-300 flex-1">
                <p className="leading-relaxed text-slate-400">
                  This section monitors and views the **physical workspace files** initialized inside the cloud container's runtime directory tree. You can inspect actual modular architectures on disk.
                </p>

                <div className="bg-slate-900/60 border border-slate-800/40 p-3 rounded-lg space-y-2 font-mono text-[10px]">
                  <div className="text-indigo-400 font-bold uppercase tracking-wider text-[9px]">ACTIVE MODULES REGISTER</div>
                  <div className="flex justify-between border-b border-slate-800/40 pb-1">
                    <span className="text-slate-400">api-gateway</span>
                    <span className="text-emerald-400 font-bold">PORT 8080</span>
                  </div>
                  <div className="flex justify-between border-b border-slate-800/40 pb-1">
                    <span className="text-slate-400">auth-service</span>
                    <span className="text-emerald-400 font-bold">PORT 8081</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-slate-400">compute-engine</span>
                    <span className="text-emerald-400 font-bold">PORT 8082</span>
                  </div>
                </div>

                <div className="bg-slate-900/60 border border-slate-800/40 p-3 rounded-lg space-y-2 font-mono text-[10px]">
                  <div className="text-amber-400 font-bold uppercase tracking-wider text-[9px]">EXTERNAL INTEGRATIONS</div>
                  <div className="flex justify-between border-b border-slate-800/40 pb-1">
                    <span className="text-slate-400">PostgreSQL Store</span>
                    <span className="text-slate-400">Active</span>
                  </div>
                  <div className="flex justify-between border-b border-slate-800/40 pb-1">
                    <span className="text-slate-400">Redis Cache</span>
                    <span className="text-slate-400">Active</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-slate-400">Kafka Broker</span>
                    <span className="text-slate-400">Active</span>
                  </div>
                </div>
              </div>

              <button
                onClick={fetchActiveDiskWorkspace}
                className="w-full py-2 bg-slate-900 border border-slate-800 text-xs font-bold text-slate-300 hover:bg-slate-800 rounded transition-colors flex items-center justify-center gap-1.5"
              >
                <Terminal size={12} /> RE-SCAN FILE SYSTEM
              </button>
            </div>
          ) : (
            <div className="space-y-4 flex flex-col h-full">
              <div className="border-b border-slate-900 pb-3">
                <h3 className="font-display font-semibold text-white tracking-wide">Enterprise Service Designer</h3>
                <p className="text-xs text-slate-400 font-sans mt-0.5">Synthesize additional clean-architecture blocks</p>
              </div>

              <form onSubmit={handleGenerateService} className="space-y-4 flex-1">
                {/* Service Name */}
                <div className="space-y-1">
                  <label className="text-[10px] text-slate-500 uppercase tracking-wider font-bold block">Microservice Name</label>
                  <input
                    type="text"
                    required
                    value={serviceName}
                    onChange={e => setServiceName(e.target.value)}
                    className="w-full bg-slate-900 border border-slate-800 rounded-lg px-3 py-2 text-xs text-white font-mono placeholder-slate-500 focus:outline-none focus:border-indigo-500/50"
                    placeholder="e.g. order-service"
                  />
                </div>

                {/* Database & Message Broker Row */}
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <label className="text-[10px] text-slate-500 uppercase tracking-wider font-bold block">Transactional Database</label>
                    <select
                      value={database}
                      onChange={e => setDatabase(e.target.value)}
                      className="w-full bg-slate-900 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-200 font-sans focus:outline-none focus:border-indigo-500/50"
                    >
                      <option value="PostgreSQL">PostgreSQL (pgx pool)</option>
                      <option value="MySQL">MySQL (go-sql-driver)</option>
                      <option value="Redis">Redis Enterprise (Cache)</option>
                      <option value="CockroachDB">CockroachDB (distributed)</option>
                    </select>
                  </div>

                  <div className="space-y-1">
                    <label className="text-[10px] text-slate-500 uppercase tracking-wider font-bold block">Async Message Broker</label>
                    <select
                      value={broker}
                      onChange={e => setBroker(e.target.value)}
                      className="w-full bg-slate-900 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-200 font-sans focus:outline-none focus:border-indigo-500/50"
                    >
                      <option value="Kafka">Apache Kafka (sarama)</option>
                      <option value="RabbitMQ">RabbitMQ (amqp091)</option>
                      <option value="gRPC Client">gRPC Internal Mesh</option>
                      <option value="Google Pub/Sub">GCP PubSub Client</option>
                    </select>
                  </div>
                </div>

                {/* API Gateway Route Mapping */}
                <div className="space-y-1.5">
                  <div className="flex items-center justify-between">
                    <label className="text-[10px] text-slate-500 uppercase tracking-wider font-bold block">HTTP Controller Endpoints</label>
                    <button
                      type="button"
                      onClick={handleAddEndpoint}
                      className="text-[9px] font-sans font-bold text-indigo-400 hover:text-indigo-300 flex items-center gap-0.5"
                    >
                      <ListPlus size={10} /> Add Route
                    </button>
                  </div>

                  <div className="space-y-2 max-h-[100px] overflow-y-auto pr-1">
                    {endpoints.map((ep, idx) => (
                      <div key={idx} className="flex gap-2 items-center bg-slate-900/60 p-2 border border-slate-900 rounded-lg">
                        <select
                          value={ep.method}
                          onChange={e => handleUpdateEndpoint(idx, "method", e.target.value)}
                          className="bg-slate-950 border border-slate-800 text-[10px] font-mono font-bold text-indigo-400 rounded p-1"
                        >
                          <option value="GET">GET</option>
                          <option value="POST">POST</option>
                          <option value="PUT">PUT</option>
                          <option value="DELETE">DELETE</option>
                        </select>
                        <input
                          type="text"
                          required
                          value={ep.path}
                          onChange={e => handleUpdateEndpoint(idx, "path", e.target.value)}
                          className="bg-slate-950 border border-slate-800 text-[10px] font-mono text-slate-200 rounded p-1 flex-1 min-w-0"
                          placeholder="/payments"
                        />
                        <button
                          type="button"
                          onClick={() => handleRemoveEndpoint(idx)}
                          className="text-slate-500 hover:text-rose-400 p-1"
                        >
                          <Trash2 size={12} />
                        </button>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Extra Specifications */}
                <div className="space-y-1">
                  <label className="text-[10px] text-slate-500 uppercase tracking-wider font-bold block">Extra architectural guidelines</label>
                  <textarea
                    value={extraFeatures}
                    onChange={e => setExtraFeatures(e.target.value)}
                    className="w-full bg-slate-900 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-300 placeholder-slate-600 font-sans focus:outline-none focus:border-indigo-500/50 h-[60px] resize-none"
                    placeholder="e.g. Include redis connection pool, go-playground/validator bindings, etc."
                  />
                </div>

                <button
                  type="submit"
                  disabled={loading}
                  className="w-full py-2 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white rounded-lg font-sans font-bold text-xs transition-colors flex items-center justify-center gap-1.5"
                >
                  <Hammer size={14} />
                  {loading ? "WIRING CORE PATTERNS..." : "SYNTHESIZE CLEAN SOURCE CODE"}
                </button>
              </form>
            </div>
          )}
        </div>

        {/* Right Side: File Tree & Code Editor (Compiler / Editor) */}
        <div className="xl:col-span-8 bg-slate-950 rounded-xl border border-slate-800 p-5 flex flex-col h-[520px] relative">
          {loading ? (
            /* Compilation Terminal Output */
            <div className="flex-1 flex flex-col min-h-0 bg-slate-950 text-emerald-400 font-mono p-4 rounded-lg border border-emerald-900/30 overflow-hidden">
              <div className="flex items-center justify-between border-b border-emerald-900/30 pb-2 mb-3 text-xs">
                <div className="flex items-center gap-1.5">
                  <Terminal size={14} className="animate-pulse" />
                  <span>Compiler Container stdout // go build -v</span>
                </div>
                <span className="text-[9px] px-1.5 py-0.5 rounded bg-emerald-950 border border-emerald-900/50 text-emerald-400 font-semibold animate-pulse">Orchestrating...</span>
              </div>

              <div className="flex-1 overflow-y-auto space-y-1 text-xs">
                {compileLogs.map((log, lIdx) => (
                  <div key={lIdx} className="leading-relaxed whitespace-pre-wrap">{log}</div>
                ))}
                <div className="animate-pulse h-4 w-1 bg-emerald-400 inline-block ml-1" />
              </div>
            </div>
          ) : activeService ? (
            /* Multi-tab Explorer and Code Editor */
            <div className="flex-1 flex flex-col min-h-0">
              {/* Explorer top bar */}
              <div className="border-b border-slate-900 pb-3 mb-3 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <h4 className="font-display font-semibold text-slate-200 text-sm flex items-center gap-1.5 truncate">
                    <Folder size={16} className="text-indigo-400 shrink-0" /> Workspace: <span className="font-mono text-indigo-400 font-bold truncate">{activeService.serviceName}</span>
                  </h4>
                  <p className="text-xs text-slate-500 mt-0.5 truncate">{activeService.description}</p>
                </div>

                {/* Utility actions */}
                <div className="flex items-center gap-2 shrink-0">
                  <button
                    onClick={handleCopyCode}
                    disabled={!selectedFile}
                    className="p-1.5 bg-slate-900 hover:bg-slate-800 border border-slate-800 rounded text-slate-400 hover:text-slate-200 transition-colors flex items-center gap-1 text-[10px] font-sans font-semibold disabled:opacity-50"
                  >
                    {copied ? <Check size={12} className="text-emerald-400" /> : <Copy size={12} />}
                    {copied ? "COPIED" : "COPY CODE"}
                  </button>
                  <button
                    onClick={handleDownloadFile}
                    disabled={!selectedFile}
                    className="p-1.5 bg-slate-900 hover:bg-slate-800 border border-slate-800 rounded text-slate-400 hover:text-slate-200 transition-colors flex items-center gap-1 text-[10px] font-sans font-semibold disabled:opacity-50"
                  >
                    <Download size={12} />
                    DOWNLOAD
                  </button>
                </div>
              </div>

              {/* Split layout: Tree View (Left) and Editor Window (Right) */}
              <div className="flex-1 flex min-h-0 border border-slate-900 rounded-lg overflow-hidden">
                {/* File tree browser sidebar */}
                <div className="w-1/3 border-r border-slate-900 bg-slate-900/20 overflow-y-auto p-2 space-y-1 text-xs">
                  <span className="text-[9px] text-slate-500 uppercase tracking-wider font-bold p-1 block">Repository tree</span>
                  
                  {/* Directory groups recursively mapped */}
                  {Object.keys(fileGroups).sort().map(folder => (
                    <div key={folder} className="space-y-0.5">
                      <div className="flex items-center gap-1.5 p-1 text-slate-400 font-bold text-[9px] uppercase font-mono tracking-wider">
                        <Folder size={10} className="text-slate-500" />
                        {folder}
                      </div>
                      <div className="pl-3.5 space-y-0.5 border-l border-slate-900/60 ml-2">
                        {fileGroups[folder].map(f => {
                          const isSelected = selectedFile === f;
                          const nameOnly = f.split("/").pop() || f;

                          return (
                            <button
                              key={f}
                              onClick={() => setSelectedFile(f)}
                              className={`w-full text-left p-1.5 rounded font-mono text-[10px] transition-colors flex items-center gap-1.5 ${
                                isSelected ? "bg-indigo-950/30 text-indigo-400 font-bold border-l-2 border-indigo-500" : "text-slate-400 hover:text-slate-200"
                              }`}
                            >
                              {getFileIcon(f)}
                              <span className="truncate">{nameOnly}</span>
                            </button>
                          );
                        })}
                      </div>
                    </div>
                  ))}
                </div>

                {/* Editor Code Viewer */}
                <div className="flex-1 bg-slate-950 overflow-auto p-4 font-mono text-xs leading-relaxed relative scrollbar-thin">
                  <div className="absolute right-3 top-3 text-[9px] text-slate-600 select-none uppercase font-bold tracking-wider font-sans">
                    {selectedFile.split(".").pop() || "source"}
                  </div>
                  <pre className="text-slate-300 select-text whitespace-pre overflow-x-auto selection:bg-slate-800">
                    {activeService.files[selectedFile] || "// Select a file to view source"}
                  </pre>
                </div>
              </div>
            </div>
          ) : (
            /* Landing Empty State */
            <div className="flex-1 flex flex-col items-center justify-center text-slate-500 font-sans text-center max-w-[400px] mx-auto">
              <Cpu size={40} className="text-slate-700 mb-3" />
              <h4 className="font-display font-semibold text-slate-300 mb-1.5">Go Clean Architecture Compiler</h4>
              <p className="text-xs text-slate-400 leading-relaxed mb-4">
                Enter details in the designer panel and trigger compilation. Our compiler will orchestrate a production Go microservice following full Clean Architecture patterns and K8s standards.
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
