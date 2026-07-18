import React, { useState, useEffect, useRef } from "react";
import { motion, AnimatePresence } from "motion/react";
import {
  Cpu,
  Layers,
  Terminal,
  Settings,
  MessageSquare,
  Database,
  Shield,
  Zap,
  Search,
  Plus,
  ArrowRight,
  RefreshCw,
  Clock,
  Coins,
  Compass,
  AlertTriangle,
  Play,
  CheckCircle,
  HelpCircle,
  FileText,
  User,
  Activity
} from "lucide-react";

// ==========================================
// FRONTEND TYPES MATCHING THE BACKEND
// ==========================================

interface VectorDocument {
  id: string;
  title: string;
  content: string;
  embedding: number[];
  metadata: Record<string, any>;
  createdAt: string;
}

interface PromptTemplate {
  id: string;
  name: string;
  description: string;
  systemInstruction: string;
  userTemplate: string;
  variables: string[];
  version: number;
  isActive: boolean;
  createdAt: string;
}

interface GatewayConfig {
  defaultProvider: "gemini" | "openai";
  defaultModel: string;
  routingStrategy: "cost" | "latency" | "capability" | "static";
  rateLimitPerMinute: number;
  failoverEnabled: boolean;
  failoverModel: string;
}

interface GatewayLog {
  id: string;
  timestamp: string;
  provider: string;
  model: string;
  promptName?: string;
  latencyMs: number;
  inputTokens: number;
  outputTokens: number;
  estimatedCost: number;
  status: "success" | "failed";
  errorMessage?: string;
  routingDecision: string;
  requestPayload: string;
  responsePayload: string;
}

interface ChatMessage {
  role: "user" | "model" | "system";
  content: string;
  timestamp: string;
}

interface ChatSession {
  id: string;
  title: string;
  memoryStrategy: "buffer" | "window" | "summary";
  windowSize: number;
  messages: ChatMessage[];
  summary?: string;
  createdAt: string;
}

interface ChainStep {
  name: string;
  type: "retrieval" | "routing" | "prompt" | "execution" | "memory";
  status: "pending" | "running" | "success" | "failed";
  details: string;
  durationMs?: number;
}

export default function AIPlatformDashboard() {
  const [activeSubTab, setActiveSubTab] = useState<"gateway" | "prompts" | "vector" | "chat" | "orchestrator">("gateway");

  // Gateway state
  const [gatewayConfig, setGatewayConfig] = useState<GatewayConfig>({
    defaultProvider: "gemini",
    defaultModel: "gemini-3.5-flash",
    routingStrategy: "cost",
    rateLimitPerMinute: 60,
    failoverEnabled: true,
    failoverModel: "gemini-3.1-pro-preview"
  });
  const [logs, setLogs] = useState<GatewayLog[]>([]);
  const [updatingConfig, setUpdatingConfig] = useState(false);

  // Prompts state
  const [prompts, setPrompts] = useState<PromptTemplate[]>([]);
  const [newPromptName, setNewPromptName] = useState("");
  const [newPromptDesc, setNewPromptDesc] = useState("");
  const [newPromptSystem, setNewPromptSystem] = useState("");
  const [newPromptTemplate, setNewPromptTemplate] = useState("");
  const [newPromptVars, setNewPromptVars] = useState("query, context");
  const [showAddPrompt, setShowAddPrompt] = useState(false);

  // Vector store & RAG state
  const [documents, setDocuments] = useState<VectorDocument[]>([]);
  const [docTitle, setDocTitle] = useState("");
  const [docContent, setDocContent] = useState("");
  const [docCategory, setDocCategory] = useState("architecture");
  const [indexingDoc, setIndexingDoc] = useState(false);
  const [showAddDoc, setShowAddDoc] = useState(false);

  // RAG Testing state
  const [ragQuery, setRagQuery] = useState("What telemetry standard does NexusCore run on?");
  const [selectedPromptId, setSelectedPromptId] = useState("prompt_rag_support");
  const [ragRunning, setRagRunning] = useState(false);
  const [ragSteps, setRagSteps] = useState<ChainStep[]>([]);
  const [ragOutput, setRagOutput] = useState("");

  // Chat state
  const [chatSessions, setChatSessions] = useState<ChatSession[]>([]);
  const [activeSessionId, setActiveSessionId] = useState<string>("");
  const [chatInput, setChatInput] = useState("");
  const [chatModel, setChatModel] = useState("gemini-3.5-flash");
  const [sendingChat, setSendingChat] = useState(false);
  const [newSessionStrategy, setNewSessionStrategy] = useState<"buffer" | "window" | "summary">("buffer");
  const [newSessionWindowSize, setNewSessionWindowSize] = useState(5);
  const [showAddSession, setShowAddSession] = useState(false);

  const messagesEndRef = useRef<HTMLDivElement>(null);

  // ==========================================
  // API FETCHERS
  // ==========================================

  const loadGatewayConfig = async () => {
    try {
      const res = await fetch("/api/ai/gateway/config");
      const data = await res.json();
      setGatewayConfig(data);
    } catch (err) {
      console.error("Error loading gateway config:", err);
    }
  };

  const loadGatewayLogs = async () => {
    try {
      const res = await fetch("/api/ai/gateway/logs");
      const data = await res.json();
      setLogs(data);
    } catch (err) {
      console.error("Error loading gateway logs:", err);
    }
  };

  const loadPrompts = async () => {
    try {
      const res = await fetch("/api/ai/prompts");
      const data = await res.json();
      setPrompts(data);
      if (data.length > 0 && !selectedPromptId) {
        setSelectedPromptId(data[0].id);
      }
    } catch (err) {
      console.error("Error loading prompts:", err);
    }
  };

  const loadDocuments = async () => {
    try {
      const res = await fetch("/api/ai/vector/documents");
      const data = await res.json();
      setDocuments(data);
    } catch (err) {
      console.error("Error loading vector documents:", err);
    }
  };

  const loadChatSessions = async () => {
    try {
      const res = await fetch("/api/ai/chat/sessions");
      const data = await res.json();
      setChatSessions(data);
      if (data.length > 0 && !activeSessionId) {
        setActiveSessionId(data[0].id);
      }
    } catch (err) {
      console.error("Error loading chat sessions:", err);
    }
  };

  useEffect(() => {
    loadGatewayConfig();
    loadGatewayLogs();
    loadPrompts();
    loadDocuments();
    loadChatSessions();

    const interval = setInterval(() => {
      loadGatewayLogs();
    }, 10000); // refresh gateway stats every 10s
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [chatSessions, activeSessionId]);

  // ==========================================
  // ACTIONS
  // ==========================================

  const updateGatewayConfig = async (updatedFields: Partial<GatewayConfig>) => {
    setUpdatingConfig(true);
    try {
      const merged = { ...gatewayConfig, ...updatedFields };
      const res = await fetch("/api/ai/gateway/config", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(merged),
      });
      const data = await res.json();
      setGatewayConfig(data.config);
      loadGatewayLogs();
    } catch (err) {
      console.error("Error updating gateway config:", err);
    } finally {
      setUpdatingConfig(false);
    }
  };

  const handleCreatePrompt = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newPromptName || !newPromptSystem || !newPromptTemplate) return;

    try {
      const res = await fetch("/api/ai/prompts", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: newPromptName,
          description: newPromptDesc,
          systemInstruction: newPromptSystem,
          userTemplate: newPromptTemplate,
          variables: newPromptVars.split(",").map(v => v.trim()).filter(Boolean),
        }),
      });

      if (res.ok) {
        setNewPromptName("");
        setNewPromptDesc("");
        setNewPromptSystem("");
        setNewPromptTemplate("");
        setShowAddPrompt(false);
        loadPrompts();
      }
    } catch (err) {
      console.error("Error creating prompt template:", err);
    }
  };

  const handleIndexDoc = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!docTitle || !docContent) return;

    setIndexingDoc(true);
    try {
      const res = await fetch("/api/ai/vector/documents", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          title: docTitle,
          content: docContent,
          metadata: { category: docCategory },
        }),
      });

      if (res.ok) {
        setDocTitle("");
        setDocContent("");
        setShowAddDoc(false);
        loadDocuments();
      }
    } catch (err) {
      console.error("Error indexing document:", err);
    } finally {
      setIndexingDoc(false);
    }
  };

  const handleRunRAG = async () => {
    if (!ragQuery) return;
    setRagRunning(true);
    setRagSteps([
      { name: "Initializing RAG Chain Node", type: "routing", status: "running", details: "Preparing request pipelines..." }
    ]);
    try {
      const res = await fetch("/api/ai/rag/query", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: ragQuery, promptId: selectedPromptId }),
      });
      const data = await res.json();
      if (res.ok) {
        setRagSteps(data.steps);
        setRagOutput(data.output);
        loadGatewayLogs();
      } else {
        setRagOutput(`RAG execution failed: ${data.error || "Internal error"}`);
      }
    } catch (err: any) {
      setRagOutput(`RAG execution exception: ${err.message}`);
    } finally {
      setRagRunning(false);
    }
  };

  const handleCreateSession = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const res = await fetch("/api/ai/chat/sessions", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          title: `Session [${newSessionStrategy.toUpperCase()}]`,
          memoryStrategy: newSessionStrategy,
          windowSize: newSessionWindowSize,
        }),
      });
      const data = await res.json();
      if (res.ok) {
        setChatSessions(prev => [data, ...prev]);
        setActiveSessionId(data.id);
        setShowAddSession(false);
      }
    } catch (err) {
      console.error("Error creating chat session:", err);
    }
  };

  const handleSendChat = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!chatInput || !activeSessionId || sendingChat) return;

    const queryText = chatInput;
    setChatInput("");
    setSendingChat(true);

    // Update frontend UI immediately for smooth UX
    setChatSessions(prev => prev.map(s => {
      if (s.id === activeSessionId) {
        return {
          ...s,
          messages: [
            ...s.messages,
            { role: "user", content: queryText, timestamp: new Date().toISOString() }
          ]
        };
      }
      return s;
    }));

    try {
      const res = await fetch("/api/ai/chat/message", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          sessionId: activeSessionId,
          message: queryText,
          modelOverride: chatModel,
        }),
      });
      const data = await res.json();
      if (res.ok) {
        // Replace with real returned history and summary from backend
        setChatSessions(prev => prev.map(s => {
          if (s.id === activeSessionId) {
            return {
              ...s,
              messages: [
                ...s.messages,
                { role: "model", content: data.reply, timestamp: new Date().toISOString() }
              ],
              summary: data.gatewayLog.routingDecision
            };
          }
          return s;
        }));
        loadGatewayLogs();
      }
    } catch (err) {
      console.error("Error sending chat message:", err);
    } finally {
      setSendingChat(false);
    }
  };

  const activeSession = chatSessions.find(s => s.id === activeSessionId);

  // Calculate high-level gateway metrics from logs
  const totalRequests = logs.length;
  const successfulRequests = logs.filter(l => l.status === "success").length;
  const failedRequests = logs.filter(l => l.status === "failed").length;
  const averageLatency = totalRequests ? Math.round(logs.reduce((sum, l) => sum + l.latencyMs, 0) / totalRequests) : 0;
  const totalCost = logs.reduce((sum, l) => sum + l.estimatedCost, 0);

  return (
    <div className="h-full flex flex-col gap-6" id="ai-platform-dashboard">
      {/* Top Banner & Quick Metrics */}
      <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4 bg-slate-900/40 p-6 rounded-xl border border-slate-800">
        <div>
          <div className="flex items-center gap-2 mb-1.5">
            <span className="p-1 bg-indigo-500/10 text-indigo-400 rounded">
              <Cpu size={16} />
            </span>
            <span className="text-xs font-mono font-semibold text-indigo-400 tracking-wider">NEXUSCORE COGNITIVE LAYER</span>
          </div>
          <h2 className="text-xl font-bold text-white tracking-tight font-display">AI Gateway & Orchestration Platform</h2>
          <p className="text-xs text-slate-400 max-w-2xl mt-1">
            Enterprise cognitive router utilizing multi-model routing policies, versioned prompt template managers, structured
            embeddings in private Vector Databases, and LangChain RAG pipeline logs.
          </p>
        </div>

        <div className="flex flex-wrap gap-3">
          <button
            onClick={() => {
              loadGatewayConfig();
              loadGatewayLogs();
              loadPrompts();
              loadDocuments();
              loadChatSessions();
            }}
            className="px-3 py-1.5 rounded bg-slate-800 border border-slate-700 text-slate-300 hover:text-white hover:bg-slate-700 text-xs font-semibold flex items-center gap-1.5 transition-all"
          >
            <RefreshCw size={13} />
            Force Reload
          </button>
        </div>
      </div>

      {/* Internal Sub Navigation Tabs */}
      <div className="flex border-b border-slate-800 bg-slate-950/40 p-1.5 rounded-lg gap-2">
        {[
          { id: "gateway", name: "AI Gateway & Routing", icon: Settings },
          { id: "prompts", name: "Prompt Registry", icon: Shield },
          { id: "vector", name: "Vector DB & RAG Analyzer", icon: Database },
          { id: "chat", name: "Multi-Model Playground", icon: MessageSquare },
          { id: "orchestrator", name: "Chain Orchestrator Graph", icon: Layers }
        ].map(tab => {
          const Icon = tab.icon;
          const isActive = activeSubTab === tab.id;
          return (
            <button
              key={tab.id}
              onClick={() => setActiveSubTab(tab.id as any)}
              className={`px-4 py-2 text-xs font-semibold rounded-md flex items-center gap-2 transition-all ${
                isActive
                  ? "bg-slate-800/80 text-white border border-slate-700/60 font-bold"
                  : "text-slate-400 hover:bg-slate-900/40 hover:text-slate-200"
              }`}
            >
              <Icon size={14} className={isActive ? "text-indigo-400" : "text-slate-500"} />
              {tab.name}
            </button>
          );
        })}
      </div>

      {/* SUB PANELS */}
      <div className="flex-1 min-h-0">
        <AnimatePresence mode="wait">
          <motion.div
            key={activeSubTab}
            initial={{ opacity: 0, y: 5 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -5 }}
            transition={{ duration: 0.12 }}
            className="h-full flex flex-col gap-6"
          >
            {/* SUBTAB 1: AI GATEWAY & ROUTING */}
            {activeSubTab === "gateway" && (
              <div className="grid grid-cols-1 xl:grid-cols-3 gap-6">
                {/* Router Policy Configuration */}
                <div className="bg-slate-900/40 p-6 rounded-xl border border-slate-800/80 flex flex-col gap-5">
                  <div className="flex items-center gap-2 border-b border-slate-800 pb-3">
                    <Settings className="text-indigo-400" size={16} />
                    <h3 className="text-sm font-bold text-white uppercase tracking-wider font-display">Cognitive Routing Policies</h3>
                  </div>

                  <div className="flex flex-col gap-4">
                    <div>
                      <label className="block text-[10px] text-slate-400 uppercase font-mono mb-1.5">Primary Provider</label>
                      <div className="grid grid-cols-2 gap-2">
                        {["gemini", "openai"].map(provider => (
                          <button
                            key={provider}
                            onClick={() => updateGatewayConfig({ defaultProvider: provider as any })}
                            className={`py-2 text-xs font-mono rounded border transition-all ${
                              gatewayConfig.defaultProvider === provider
                                ? "bg-indigo-600/20 text-indigo-300 border-indigo-500 font-bold"
                                : "bg-slate-950/50 text-slate-500 border-slate-800 hover:text-slate-300"
                            }`}
                          >
                            {provider.toUpperCase()}
                          </button>
                        ))}
                      </div>
                    </div>

                    <div>
                      <label className="block text-[10px] text-slate-400 uppercase font-mono mb-1.5">Default Model Routing Target</label>
                      <select
                        value={gatewayConfig.defaultModel}
                        onChange={(e) => updateGatewayConfig({ defaultModel: e.target.value })}
                        className="w-full bg-slate-950 border border-slate-800 rounded px-3 py-2 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 font-mono"
                      >
                        <option value="gemini-3.5-flash">gemini-3.5-flash (Low cost / Flash)</option>
                        <option value="gemini-3.1-pro-preview">gemini-3.1-pro-preview (Deep Reasoning)</option>
                        <option value="gpt-4o-mini">gpt-4o-mini (OpenAI cost-optimized)</option>
                        <option value="gpt-4o">gpt-4o (OpenAI primary core)</option>
                      </select>
                    </div>

                    <div>
                      <label className="block text-[10px] text-slate-400 uppercase font-mono mb-1.5">Dynamic Solver Strategy</label>
                      <div className="flex flex-col gap-2">
                        {[
                          { id: "cost", title: "Cost-Optimized", desc: "Routes lightweight prompts to Flash/Mini models dynamically." },
                          { id: "capability", title: "Capability-Optimized", desc: "Routes code, math, and heavy text to Pro/GPT-4o models." },
                          { id: "latency", title: "Latency-Optimized", desc: "Forces lowest latency edge routing models." },
                          { id: "static", title: "Strict Static Override", desc: "Bypasses decision trees, routes strictly to Default target." }
                        ].map(strat => (
                          <button
                            key={strat.id}
                            onClick={() => updateGatewayConfig({ routingStrategy: strat.id as any })}
                            className={`p-3 rounded border text-left flex items-start gap-3 transition-all ${
                              gatewayConfig.routingStrategy === strat.id
                                ? "bg-indigo-600/10 border-indigo-500 text-white"
                                : "bg-slate-950/30 border-slate-800/80 text-slate-400 hover:bg-slate-900/30"
                            }`}
                          >
                            <span className={`p-1.5 rounded ${gatewayConfig.routingStrategy === strat.id ? "bg-indigo-600/20 text-indigo-400" : "bg-slate-900 text-slate-500"}`}>
                              <Zap size={12} />
                            </span>
                            <div>
                              <div className="text-xs font-bold">{strat.title}</div>
                              <div className="text-[10px] text-slate-500 mt-0.5">{strat.desc}</div>
                            </div>
                          </button>
                        ))}
                      </div>
                    </div>

                    <div className="border-t border-slate-800/80 pt-4 flex flex-col gap-3">
                      <div className="flex items-center justify-between">
                        <div>
                          <div className="text-xs font-bold text-slate-200">API Failover Policy</div>
                          <div className="text-[10px] text-slate-500">Route to failover model if principal model errors.</div>
                        </div>
                        <button
                          onClick={() => updateGatewayConfig({ failoverEnabled: !gatewayConfig.failoverEnabled })}
                          className={`w-10 h-5 rounded-full relative transition-all ${
                            gatewayConfig.failoverEnabled ? "bg-emerald-500" : "bg-slate-800"
                          }`}
                        >
                          <span className={`w-3.5 h-3.5 bg-white rounded-full absolute top-0.5 transition-all ${
                            gatewayConfig.failoverEnabled ? "right-1" : "left-1"
                          }`} />
                        </button>
                      </div>

                      {gatewayConfig.failoverEnabled && (
                        <div>
                          <label className="block text-[10px] text-slate-400 uppercase font-mono mb-1">Failover Target Model</label>
                          <select
                            value={gatewayConfig.failoverModel}
                            onChange={(e) => updateGatewayConfig({ failoverModel: e.target.value })}
                            className="w-full bg-slate-950 border border-slate-800 rounded px-2.5 py-1.5 text-[11px] text-slate-300 focus:outline-none font-mono"
                          >
                            <option value="gemini-3.1-pro-preview">gemini-3.1-pro-preview (Highly Robust)</option>
                            <option value="gemini-3.5-flash">gemini-3.5-flash</option>
                            <option value="gpt-4o-mini">gpt-4o-mini</option>
                          </select>
                        </div>
                      )}
                    </div>
                  </div>
                </div>

                {/* Gateway telemetry charts & indicators */}
                <div className="xl:col-span-2 flex flex-col gap-6">
                  {/* Performance Indicators */}
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <div className="bg-slate-900/40 p-4 rounded-xl border border-slate-800 flex flex-col justify-between">
                      <div className="flex items-center justify-between text-slate-500 mb-2">
                        <span className="text-[10px] uppercase font-mono tracking-wider">Gateway Requests</span>
                        <Activity size={14} className="text-indigo-400 animate-pulse" />
                      </div>
                      <div>
                        <div className="text-2xl font-bold text-white font-mono">{totalRequests}</div>
                        <div className="text-[10px] text-emerald-400 mt-1">● Real-time sync</div>
                      </div>
                    </div>

                    <div className="bg-slate-900/40 p-4 rounded-xl border border-slate-800 flex flex-col justify-between">
                      <div className="flex items-center justify-between text-slate-500 mb-2">
                        <span className="text-[10px] uppercase font-mono tracking-wider">Avg Latency</span>
                        <Clock size={14} className="text-indigo-400" />
                      </div>
                      <div>
                        <div className="text-2xl font-bold text-white font-mono">{averageLatency} ms</div>
                        <div className="text-[10px] text-slate-500 mt-1">End-to-end API ping</div>
                      </div>
                    </div>

                    <div className="bg-slate-900/40 p-4 rounded-xl border border-slate-800 flex flex-col justify-between">
                      <div className="flex items-center justify-between text-slate-500 mb-2">
                        <span className="text-[10px] uppercase font-mono tracking-wider">Estimated Cost</span>
                        <Coins size={14} className="text-emerald-400" />
                      </div>
                      <div>
                        <div className="text-2xl font-bold text-emerald-400 font-mono">${totalCost.toFixed(5)}</div>
                        <div className="text-[10px] text-slate-500 mt-1">Calculated token metrics</div>
                      </div>
                    </div>

                    <div className="bg-slate-900/40 p-4 rounded-xl border border-slate-800 flex flex-col justify-between">
                      <div className="flex items-center justify-between text-slate-500 mb-2">
                        <span className="text-[10px] uppercase font-mono tracking-wider">Reliability</span>
                        <CheckCircle size={14} className="text-emerald-500" />
                      </div>
                      <div>
                        <div className="text-2xl font-bold text-white font-mono">
                          {totalRequests ? Math.round((successfulRequests / totalRequests) * 100) : 100}%
                        </div>
                        <div className="text-[10px] text-slate-500 mt-1">
                          Failures: {failedRequests} log streams
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Real-time Gateway Audit Trail */}
                  <div className="bg-slate-900/40 rounded-xl border border-slate-800 flex-1 flex flex-col min-h-[400px]">
                    <div className="p-4 border-b border-slate-800 flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Terminal className="text-indigo-400" size={15} />
                        <span className="text-xs font-bold uppercase tracking-wider text-white">AI Gateway Proxy Trace Log</span>
                      </div>
                      <span className="text-[10px] text-slate-500 font-mono">Real-time socket streams</span>
                    </div>

                    <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-2.5 max-h-[450px]">
                      {logs.length === 0 ? (
                        <div className="flex flex-col items-center justify-center h-full text-slate-600 gap-2 py-20">
                          <HelpCircle size={28} className="text-slate-700" />
                          <span className="text-xs">No gateway queries captured yet. Initiate chat or query vectors to log operations.</span>
                        </div>
                      ) : (
                        logs.map((log) => (
                          <div
                            key={log.id}
                            className={`p-3.5 rounded-lg border flex flex-col gap-2 bg-slate-950/40 transition-all ${
                              log.status === "failed" ? "border-red-900/30 bg-red-950/5" : "border-slate-800/60"
                            }`}
                          >
                            <div className="flex items-center justify-between">
                              <div className="flex items-center gap-2 flex-wrap">
                                <span className={`text-[10px] px-2 py-0.5 rounded font-bold uppercase ${
                                  log.provider === "gemini" ? "bg-indigo-500/10 text-indigo-400 border border-indigo-500/20" : "bg-sky-500/10 text-sky-400 border border-sky-500/20"
                                }`}>
                                  {log.provider}
                                </span>
                                <span className="text-xs font-bold text-slate-300 font-mono">{log.model}</span>
                                {log.promptName && (
                                  <span className="text-[10px] bg-slate-800 text-slate-400 px-1.5 py-0.5 rounded border border-slate-700">
                                    Template: {log.promptName}
                                  </span>
                                )}
                              </div>
                              <div className="text-[10px] text-slate-500 font-mono">
                                {new Date(log.timestamp).toLocaleTimeString()}
                              </div>
                            </div>

                            <div className="text-xs text-slate-400 font-mono bg-slate-950 p-2.5 rounded border border-slate-900 overflow-x-auto whitespace-pre-wrap max-h-24">
                              {log.requestPayload ? JSON.parse(log.requestPayload).prompt : "Empty Payload"}
                            </div>

                            <div className="flex items-center justify-between text-[10px] text-slate-500 font-mono border-t border-slate-900 pt-2 flex-wrap gap-2">
                              <div className="flex items-center gap-3">
                                <span>Latency: <strong className="text-slate-300">{log.latencyMs}ms</strong></span>
                                <span>Tokens: <strong className="text-slate-300">In:{log.inputTokens} / Out:{log.outputTokens}</strong></span>
                                <span>Cost: <strong className="text-emerald-500">${log.estimatedCost.toFixed(5)}</strong></span>
                              </div>
                              <div className="flex items-center gap-2">
                                <span className="text-[9px] text-slate-600">{log.routingDecision}</span>
                                <span className={`w-2 h-2 rounded-full ${log.status === "success" ? "bg-emerald-500" : "bg-red-500 animate-pulse"}`} />
                              </div>
                            </div>

                            {log.errorMessage && (
                              <div className="text-[10px] text-red-400 bg-red-950/20 p-2 rounded border border-red-900/30 font-mono mt-1 flex items-center gap-1.5">
                                <AlertTriangle size={12} className="text-red-500 shrink-0" />
                                <span>{log.errorMessage}</span>
                              </div>
                            )}
                          </div>
                        ))
                      )}
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* SUBTAB 2: PROMPT REGISTRY */}
            {activeSubTab === "prompts" && (
              <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                {/* Prompts list */}
                <div className="lg:col-span-2 flex flex-col gap-4">
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-slate-400 font-mono uppercase tracking-wider font-bold">Registered Prompts</span>
                    <button
                      onClick={() => setShowAddPrompt(!showAddPrompt)}
                      className="px-2.5 py-1.5 rounded bg-indigo-600 text-white hover:bg-indigo-500 text-xs font-semibold flex items-center gap-1 transition-all"
                    >
                      <Plus size={13} />
                      Register New Prompt
                    </button>
                  </div>

                  <div className="grid grid-cols-1 gap-4">
                    {prompts.map(prompt => (
                      <div key={prompt.id} className="bg-slate-900/40 p-5 rounded-xl border border-slate-800 flex flex-col gap-4">
                        <div className="flex items-center justify-between flex-wrap gap-2">
                          <div>
                            <div className="flex items-center gap-2">
                              <h4 className="text-sm font-bold text-white font-display">{prompt.name}</h4>
                              <span className="text-[9px] bg-slate-800 text-slate-400 px-1.5 py-0.5 rounded border border-slate-700 font-mono">
                                v{prompt.version}.0
                              </span>
                            </div>
                            <p className="text-xs text-slate-400 mt-1">{prompt.description}</p>
                          </div>
                          <span className="text-[10px] text-slate-500 font-mono">ID: {prompt.id}</span>
                        </div>

                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                          <div>
                            <span className="block text-[10px] text-slate-500 uppercase font-mono mb-1">System Instruction (Persona)</span>
                            <div className="bg-slate-950 p-3 rounded text-[11px] font-mono text-slate-300 border border-slate-800/80 h-36 overflow-y-auto">
                              {prompt.systemInstruction}
                            </div>
                          </div>
                          <div>
                            <span className="block text-[10px] text-slate-500 uppercase font-mono mb-1">User Template</span>
                            <div className="bg-slate-950 p-3 rounded text-[11px] font-mono text-slate-300 border border-slate-800/80 h-36 overflow-y-auto">
                              {prompt.userTemplate}
                            </div>
                          </div>
                        </div>

                        <div className="flex items-center justify-between border-t border-slate-800/60 pt-3 text-[10px] text-slate-500 font-mono">
                          <div className="flex items-center gap-2">
                            <span>Variables detected:</span>
                            {prompt.variables.map(v => (
                              <span key={v} className="bg-slate-800 text-slate-300 px-1.5 py-0.5 rounded text-[9px] font-bold">
                                {"{{" + v + "}}"}
                              </span>
                            ))}
                          </div>
                          <span>Created: {new Date(prompt.createdAt).toLocaleDateString()}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Add prompt form */}
                <div className="bg-slate-900/40 p-6 rounded-xl border border-slate-800 flex flex-col gap-4 h-fit">
                  <div className="flex items-center gap-2 border-b border-slate-800 pb-3">
                    <Shield className="text-indigo-400" size={16} />
                    <h3 className="text-sm font-bold text-white uppercase tracking-wider font-display">SecOps Shield Injection Scan</h3>
                  </div>

                  <p className="text-xs text-slate-400">
                    NexusCore monitors incoming prompts for potential safety jailbreaks, credential leakage vectors, or prompt injection
                    signatures.
                  </p>

                  <div className="bg-slate-950 p-4 rounded-lg border border-slate-800/80 mt-2">
                    <div className="text-[10px] text-slate-500 uppercase font-mono mb-2">Simulation Engine</div>
                    <div className="flex flex-col gap-2.5">
                      <div className="text-xs text-slate-300 font-mono flex items-center gap-2">
                        <CheckCircle size={13} className="text-emerald-500" />
                        Prompt Registry Sandbox Active
                      </div>
                      <div className="text-xs text-slate-300 font-mono flex items-center gap-2">
                        <CheckCircle size={13} className="text-emerald-500" />
                        Dynamic Variable Checker Online
                      </div>
                      <div className="text-xs text-slate-300 font-mono flex items-center gap-2 text-indigo-400">
                        <Activity size={13} className="animate-pulse" />
                        LLM Firewall Logs streaming
                      </div>
                    </div>
                  </div>

                  {showAddPrompt && (
                    <motion.form
                      initial={{ opacity: 0, height: 0 }}
                      animate={{ opacity: 1, height: "auto" }}
                      onSubmit={handleCreatePrompt}
                      className="border-t border-slate-800 pt-4 flex flex-col gap-3.5 mt-2"
                    >
                      <div className="text-xs font-bold text-white uppercase tracking-wider">Register Template</div>
                      <div>
                        <label className="block text-[10px] text-slate-400 uppercase font-mono mb-1">Template Name</label>
                        <input
                          type="text"
                          required
                          placeholder="e.g. Code Review Validator"
                          value={newPromptName}
                          onChange={(e) => setNewPromptName(e.target.value)}
                          className="w-full bg-slate-950 border border-slate-800 rounded px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none"
                        />
                      </div>
                      <div>
                        <label className="block text-[10px] text-slate-400 uppercase font-mono mb-1">Description</label>
                        <input
                          type="text"
                          placeholder="Brief role representation"
                          value={newPromptDesc}
                          onChange={(e) => setNewPromptDesc(e.target.value)}
                          className="w-full bg-slate-950 border border-slate-800 rounded px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none"
                        />
                      </div>
                      <div>
                        <label className="block text-[10px] text-slate-400 uppercase font-mono mb-1">System Instruction</label>
                        <textarea
                          required
                          rows={3}
                          placeholder="You are an expert security engineer..."
                          value={newPromptSystem}
                          onChange={(e) => setNewPromptSystem(e.target.value)}
                          className="w-full bg-slate-950 border border-slate-800 rounded px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none font-mono"
                        />
                      </div>
                      <div>
                        <label className="block text-[10px] text-slate-400 uppercase font-mono mb-1">User Template</label>
                        <textarea
                          required
                          rows={3}
                          placeholder="CONTEXT: {{context}} \n\n QUERY: {{query}}"
                          value={newPromptTemplate}
                          onChange={(e) => setNewPromptTemplate(e.target.value)}
                          className="w-full bg-slate-950 border border-slate-800 rounded px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none font-mono"
                        />
                      </div>
                      <div>
                        <label className="block text-[10px] text-slate-400 uppercase font-mono mb-1">Variables (comma separated)</label>
                        <input
                          type="text"
                          value={newPromptVars}
                          onChange={(e) => setNewPromptVars(e.target.value)}
                          className="w-full bg-slate-950 border border-slate-800 rounded px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none font-mono"
                        />
                      </div>
                      <button
                        type="submit"
                        className="w-full py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold mt-2"
                      >
                        Commit to Core Registry
                      </button>
                    </motion.form>
                  )}
                </div>
              </div>
            )}

            {/* SUBTAB 3: VECTOR DB & RAG ANALYZER */}
            {activeSubTab === "vector" && (
              <div className="grid grid-cols-1 xl:grid-cols-12 gap-6">
                {/* Column 1: Indexed documents lists */}
                <div className="xl:col-span-5 flex flex-col gap-4">
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-slate-400 font-mono uppercase tracking-wider font-bold">Vector Database (In-Memory Nodes)</span>
                    <button
                      onClick={() => setShowAddDoc(!showAddDoc)}
                      className="px-2.5 py-1.5 rounded bg-indigo-600 text-white hover:bg-indigo-500 text-xs font-semibold flex items-center gap-1 transition-all"
                    >
                      <Plus size={13} />
                      Index Document
                    </button>
                  </div>

                  {showAddDoc && (
                    <form onSubmit={handleIndexDoc} className="bg-slate-900/40 p-4 rounded-xl border border-slate-800 flex flex-col gap-3.5">
                      <div className="text-xs font-bold text-white uppercase tracking-wider">Add Knowledge Block</div>
                      <div>
                        <label className="block text-[10px] text-slate-400 uppercase font-mono mb-1">Document Title</label>
                        <input
                          type="text"
                          required
                          placeholder="e.g. OAuth Flow Guidelines"
                          value={docTitle}
                          onChange={(e) => setDocTitle(e.target.value)}
                          className="w-full bg-slate-950 border border-slate-800 rounded px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none font-sans"
                        />
                      </div>
                      <div>
                        <label className="block text-[10px] text-slate-400 uppercase font-mono mb-1">Category</label>
                        <select
                          value={docCategory}
                          onChange={(e) => setDocCategory(e.target.value)}
                          className="w-full bg-slate-950 border border-slate-800 rounded px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none"
                        >
                          <option value="architecture">Architecture Blueprint</option>
                          <option value="ai">AI / Vector Strategy</option>
                          <option value="operations">Operations / Site Reliability</option>
                        </select>
                      </div>
                      <div>
                        <label className="block text-[10px] text-slate-400 uppercase font-mono mb-1">Content Body</label>
                        <textarea
                          required
                          rows={4}
                          placeholder="Paste document content to tokenize and embed..."
                          value={docContent}
                          onChange={(e) => setDocContent(e.target.value)}
                          className="w-full bg-slate-950 border border-slate-800 rounded px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none font-mono"
                        />
                      </div>
                      <button
                        type="submit"
                        disabled={indexingDoc}
                        className="w-full py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold disabled:opacity-55 flex items-center justify-center gap-1.5"
                      >
                        {indexingDoc ? (
                          <>
                            <RefreshCw className="animate-spin" size={13} />
                            Vectorizing text block...
                          </>
                        ) : (
                          "Generate Embedding Vector & Save"
                        )}
                      </button>
                    </form>
                  )}

                  <div className="flex flex-col gap-3 max-h-[500px] overflow-y-auto">
                    {documents.map(doc => (
                      <div key={doc.id} className="p-4 rounded-xl border border-slate-800/80 bg-slate-900/10 flex flex-col gap-2">
                        <div className="flex items-center justify-between">
                          <span className="text-xs font-bold text-white font-display flex items-center gap-2">
                            <FileText size={13} className="text-indigo-400" />
                            {doc.title}
                          </span>
                          <span className="text-[9px] bg-slate-800 text-slate-400 px-1.5 py-0.5 rounded border border-slate-700 font-mono">
                            {doc.metadata.category}
                          </span>
                        </div>

                        <p className="text-[11px] text-slate-400 font-mono leading-relaxed bg-slate-950/40 p-2.5 rounded border border-slate-950 line-clamp-3">
                          {doc.content}
                        </p>

                        <div className="flex items-center justify-between text-[10px] text-slate-500 font-mono border-t border-slate-900/50 pt-2 mt-1">
                          <span>Dims: 1536 (Normalized)</span>
                          <span>ID: {doc.id}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Column 2: RAG Tester Sandbox */}
                <div className="xl:col-span-7 bg-slate-900/40 p-6 rounded-xl border border-slate-800 flex flex-col gap-5">
                  <div className="flex items-center gap-2 border-b border-slate-800 pb-3">
                    <Compass className="text-indigo-400" size={16} />
                    <h3 className="text-sm font-bold text-white uppercase tracking-wider font-display">Cognitive RAG Retrieval Test Sandbox</h3>
                  </div>

                  <p className="text-xs text-slate-400">
                    Simulate the entire LangChain pipeline. Enter a query to execute vector search, prompt template hydration,
                    and model completion in real time.
                  </p>

                  <div className="flex flex-col gap-4">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <label className="block text-[10px] text-slate-400 uppercase font-mono mb-1.5">Query Input</label>
                        <input
                          type="text"
                          value={ragQuery}
                          onChange={(e) => setRagQuery(e.target.value)}
                          className="w-full bg-slate-950 border border-slate-800 rounded px-3 py-2 text-xs text-slate-200 focus:outline-none"
                        />
                      </div>
                      <div>
                        <label className="block text-[10px] text-slate-400 uppercase font-mono mb-1.5">Format Template</label>
                        <select
                          value={selectedPromptId}
                          onChange={(e) => setSelectedPromptId(e.target.value)}
                          className="w-full bg-slate-950 border border-slate-800 rounded px-3 py-2 text-xs text-slate-200 focus:outline-none"
                        >
                          {prompts.map(p => (
                            <option key={p.id} value={p.id}>{p.name}</option>
                          ))}
                        </select>
                      </div>
                    </div>

                    <button
                      onClick={handleRunRAG}
                      disabled={ragRunning}
                      className="w-full py-2.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold flex items-center justify-center gap-2 transition-all disabled:opacity-50"
                    >
                      {ragRunning ? (
                        <>
                          <RefreshCw className="animate-spin" size={13} />
                          Running retrieval chain loops...
                        </>
                      ) : (
                        <>
                          <Play size={13} />
                          Execute RAG Pipeline
                        </>
                      )}
                    </button>

                    {/* Pipeline trace timeline */}
                    {ragSteps.length > 0 && (
                      <div className="border-t border-slate-800/80 pt-4 flex flex-col gap-3">
                        <span className="text-[10px] text-slate-500 uppercase font-mono">LangChain Execution Plan</span>
                        <div className="flex flex-col gap-3">
                          {ragSteps.map((step, idx) => (
                            <div key={idx} className="flex items-start gap-3.5 bg-slate-950/30 p-2.5 rounded border border-slate-900">
                              <span className={`p-1 rounded mt-0.5 ${
                                step.status === "success" ? "bg-emerald-500/10 text-emerald-400" : "bg-indigo-500/10 text-indigo-400"
                              }`}>
                                <CheckCircle size={12} />
                              </span>
                              <div className="flex-1">
                                <div className="flex items-center justify-between">
                                  <span className="text-xs font-bold text-slate-200 font-mono">{step.name}</span>
                                  {step.durationMs && (
                                    <span className="text-[10px] text-slate-500 font-mono">{step.durationMs}ms</span>
                                  )}
                                </div>
                                <span className="text-[11px] text-slate-400 mt-1 block font-mono">{step.details}</span>
                              </div>
                            </div>
                          ))}
                        </div>

                        {ragOutput && (
                          <div className="mt-2">
                            <span className="text-[10px] text-slate-500 uppercase font-mono block mb-1.5">Model Output</span>
                            <div className="bg-slate-950 p-4 rounded-lg border border-slate-800/80 text-xs font-mono text-slate-300 leading-relaxed max-h-48 overflow-y-auto whitespace-pre-wrap">
                              {ragOutput}
                            </div>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                </div>
              </div>
            )}

            {/* SUBTAB 4: DYNAMIC AI CHAT PLAYGROUND */}
            {activeSubTab === "chat" && (
              <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 h-[550px]">
                {/* Session lists */}
                <div className="lg:col-span-3 bg-slate-900/40 rounded-xl border border-slate-800 flex flex-col min-h-0">
                  <div className="p-4 border-b border-slate-800 flex items-center justify-between shrink-0">
                    <span className="text-xs font-bold text-white uppercase tracking-wider font-display">Sessions</span>
                    <button
                      onClick={() => setShowAddSession(!showAddSession)}
                      className="p-1 bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white rounded border border-slate-700"
                    >
                      <Plus size={14} />
                    </button>
                  </div>

                  {showAddSession && (
                    <form onSubmit={handleCreateSession} className="p-3 border-b border-slate-800/80 bg-slate-950/20 flex flex-col gap-2 shrink-0">
                      <div>
                        <label className="block text-[9px] text-slate-500 uppercase font-mono mb-1">Memory Strategy</label>
                        <select
                          value={newSessionStrategy}
                          onChange={(e) => setNewSessionStrategy(e.target.value as any)}
                          className="w-full bg-slate-950 border border-slate-800 rounded px-2 py-1 text-xs text-slate-300 focus:outline-none"
                        >
                          <option value="buffer">Full Buffer Memory</option>
                          <option value="window">Buffer Window Memory</option>
                          <option value="summary">Summary Memory</option>
                        </select>
                      </div>
                      {newSessionStrategy === "window" && (
                        <div>
                          <label className="block text-[9px] text-slate-500 uppercase font-mono mb-1">Window Size</label>
                          <input
                            type="number"
                            value={newSessionWindowSize}
                            onChange={(e) => setNewSessionWindowSize(Number(e.target.value))}
                            className="w-full bg-slate-950 border border-slate-800 rounded px-2 py-1 text-xs text-slate-300 focus:outline-none font-mono"
                          />
                        </div>
                      )}
                      <button
                        type="submit"
                        className="w-full py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold mt-1"
                      >
                        Create Session
                      </button>
                    </form>
                  )}

                  <div className="flex-1 overflow-y-auto p-2.5 flex flex-col gap-1.5">
                    {chatSessions.map(session => (
                      <button
                        key={session.id}
                        onClick={() => setActiveSessionId(session.id)}
                        className={`w-full text-left p-3 rounded-lg border transition-all flex flex-col gap-1.5 ${
                          activeSessionId === session.id
                            ? "bg-indigo-600/10 border-indigo-500 text-white"
                            : "bg-slate-950/20 border-slate-850 text-slate-400 hover:bg-slate-900/30"
                        }`}
                      >
                        <span className="text-xs font-bold block truncate font-display">{session.title}</span>
                        <div className="flex items-center justify-between text-[9px] text-slate-500 font-mono w-full border-t border-slate-900/20 pt-1.5">
                          <span>Strategy: {session.memoryStrategy}</span>
                          <span>Msg: {session.messages.length}</span>
                        </div>
                      </button>
                    ))}
                  </div>
                </div>

                {/* Chat window */}
                <div className="lg:col-span-9 bg-slate-900/40 rounded-xl border border-slate-800 flex flex-col min-h-0 relative">
                  {/* Chat header */}
                  <div className="p-4 border-b border-slate-800 flex flex-wrap items-center justify-between gap-3 shrink-0">
                    <div className="flex items-center gap-2">
                      <MessageSquare className="text-indigo-400" size={15} />
                      <span className="text-xs font-bold text-white font-display">Active Session Playground</span>
                      {activeSession && (
                        <span className="text-[10px] bg-slate-800 text-slate-400 px-1.5 py-0.5 rounded border border-slate-700 font-mono uppercase">
                          {activeSession.memoryStrategy} Memory
                        </span>
                      )}
                    </div>

                    <div className="flex items-center gap-2">
                      <label className="text-[10px] text-slate-500 uppercase font-mono">Invoke Model:</label>
                      <select
                        value={chatModel}
                        onChange={(e) => setChatModel(e.target.value)}
                        className="bg-slate-950 border border-slate-800 rounded px-2.5 py-1 text-xs text-slate-300 focus:outline-none font-mono"
                      >
                        <option value="gemini-3.5-flash">gemini-3.5-flash</option>
                        <option value="gemini-3.1-pro-preview">gemini-3.1-pro-preview</option>
                        <option value="gpt-4o-mini">gpt-4o-mini</option>
                        <option value="gpt-4o">gpt-4o</option>
                      </select>
                    </div>
                  </div>

                  {/* Active context summaries indicator */}
                  {activeSession?.summary && (
                    <div className="bg-slate-950 px-4 py-2 border-b border-slate-900 text-[10px] font-mono text-slate-500 flex items-center justify-between gap-2 shrink-0">
                      <div className="truncate">
                        <strong>Gateway Router Memo:</strong> {activeSession.summary}
                      </div>
                      <span className="text-indigo-400 shrink-0">Memory Active</span>
                    </div>
                  )}

                  {/* Chat messages */}
                  <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-4">
                    {!activeSession ? (
                      <div className="flex flex-col items-center justify-center h-full text-slate-600 gap-2 py-20">
                        <HelpCircle size={28} className="text-slate-700" />
                        <span className="text-xs">Create or select a chat session to begin playground testing.</span>
                      </div>
                    ) : activeSession.messages.length === 0 ? (
                      <div className="flex flex-col items-center justify-center h-full text-slate-600 gap-2 py-20">
                        <Compass size={28} className="text-indigo-500/20" />
                        <span className="text-xs text-slate-500">Session empty. Send a prompt below to interact with the model routing.</span>
                      </div>
                    ) : (
                      activeSession.messages.map((m, idx) => (
                        <div
                          key={idx}
                          className={`flex gap-3 max-w-[85%] ${
                            m.role === "user" ? "ml-auto flex-row-reverse" : "mr-auto"
                          }`}
                        >
                          <div className={`w-7 h-7 rounded-full flex items-center justify-center font-bold text-[10px] shrink-0 border ${
                            m.role === "user"
                              ? "bg-indigo-600 border-indigo-500 text-white"
                              : "bg-slate-800 border-slate-700 text-indigo-400"
                          }`}>
                            {m.role === "user" ? "US" : "AI"}
                          </div>
                          <div className={`p-3.5 rounded-xl border leading-relaxed text-xs font-mono whitespace-pre-wrap ${
                            m.role === "user"
                              ? "bg-indigo-600/10 border-indigo-500/40 text-slate-200"
                              : "bg-slate-950 border-slate-850 text-slate-300"
                          }`}>
                            {m.content}
                          </div>
                        </div>
                      ))
                    )}
                    <div ref={messagesEndRef} />
                  </div>

                  {/* Input form */}
                  <form onSubmit={handleSendChat} className="p-3 border-t border-slate-800 bg-slate-950/40 flex gap-2 shrink-0">
                    <input
                      type="text"
                      disabled={!activeSessionId || sendingChat}
                      placeholder={sendingChat ? "Awaiting model token synthesis..." : "Prompt LLM or ask architecture questions..."}
                      value={chatInput}
                      onChange={(e) => setChatInput(e.target.value)}
                      className="flex-1 bg-slate-950 border border-slate-800 rounded-lg px-4 py-2.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 disabled:opacity-50"
                    />
                    <button
                      type="submit"
                      disabled={!activeSessionId || sendingChat || !chatInput}
                      className="px-4 bg-indigo-600 hover:bg-indigo-500 disabled:bg-slate-800 disabled:text-slate-500 text-white rounded-lg text-xs font-semibold flex items-center justify-center transition-all"
                    >
                      {sendingChat ? (
                        <RefreshCw className="animate-spin" size={13} />
                      ) : (
                        <ArrowRight size={14} />
                      )}
                    </button>
                  </form>
                </div>
              </div>
            )}

            {/* SUBTAB 5: ORCHESTRATOR CHAIN GRAPH */}
            {activeSubTab === "orchestrator" && (
              <div className="bg-slate-900/40 p-6 rounded-xl border border-slate-800 flex flex-col gap-6">
                <div className="flex items-center gap-2 border-b border-slate-800 pb-3">
                  <Layers className="text-indigo-400" size={16} />
                  <h3 className="text-sm font-bold text-white uppercase tracking-wider font-display">LangChain-inspired Graph Orchestration Pipeline</h3>
                </div>

                <p className="text-xs text-slate-400">
                  Visual representation of the standard execution steps inside our LangChain-powered Cognitive Router.
                  This state flow guarantees validation, semantic memory indexing, prompt parsing, and model routing parameters occur sequentially.
                </p>

                {/* The Chain Graph */}
                <div className="flex flex-col lg:flex-row items-center justify-center py-10 gap-4 lg:gap-0 font-mono">
                  {/* Step 1: Query Ingestion */}
                  <div className="flex items-center justify-center">
                    <div className="w-40 bg-slate-950 p-4 rounded-xl border border-indigo-500/50 flex flex-col items-center text-center gap-2 shadow-lg relative">
                      <User size={20} className="text-indigo-400" />
                      <div className="text-xs font-bold text-white">1. Query Ingest</div>
                      <div className="text-[9px] text-slate-500">Handshake auth & rate checks</div>
                      <span className="absolute -bottom-1 lg:bottom-auto lg:-right-1 w-2 h-2 bg-indigo-400 rounded-full" />
                    </div>
                  </div>

                  <div className="w-1.5 h-6 lg:w-12 lg:h-1 flex items-center justify-center bg-indigo-500/30" />

                  {/* Step 2: Semantic Encoder */}
                  <div className="flex items-center justify-center">
                    <div className="w-40 bg-slate-950 p-4 rounded-xl border border-slate-800 flex flex-col items-center text-center gap-2 shadow-lg relative">
                      <Zap size={20} className="text-amber-400" />
                      <div className="text-xs font-bold text-white">2. Vector Encoder</div>
                      <div className="text-[9px] text-slate-500">Embedding dims parsing</div>
                      <span className="absolute -bottom-1 lg:bottom-auto lg:-right-1 w-2 h-2 bg-slate-500 rounded-full" />
                    </div>
                  </div>

                  <div className="w-1.5 h-6 lg:w-12 lg:h-1 flex items-center justify-center bg-indigo-500/30" />

                  {/* Step 3: Retriever matcher */}
                  <div className="flex items-center justify-center">
                    <div className="w-40 bg-slate-950 p-4 rounded-xl border border-slate-800 flex flex-col items-center text-center gap-2 shadow-lg relative">
                      <Database size={20} className="text-emerald-400" />
                      <div className="text-xs font-bold text-white">3. Vector Match</div>
                      <div className="text-[9px] text-slate-500">Cosine index similarities</div>
                      <span className="absolute -bottom-1 lg:bottom-auto lg:-right-1 w-2 h-2 bg-slate-500 rounded-full" />
                    </div>
                  </div>

                  <div className="w-1.5 h-6 lg:w-12 lg:h-1 flex items-center justify-center bg-indigo-500/30" />

                  {/* Step 4: Prompt Hydrator */}
                  <div className="flex items-center justify-center">
                    <div className="w-40 bg-slate-950 p-4 rounded-xl border border-slate-800 flex flex-col items-center text-center gap-2 shadow-lg relative">
                      <Shield size={20} className="text-sky-400" />
                      <div className="text-xs font-bold text-white">4. Hydrate Prompt</div>
                      <div className="text-[9px] text-slate-500">Context var mappings</div>
                      <span className="absolute -bottom-1 lg:bottom-auto lg:-right-1 w-2 h-2 bg-slate-500 rounded-full" />
                    </div>
                  </div>

                  <div className="w-1.5 h-6 lg:w-12 lg:h-1 flex items-center justify-center bg-indigo-500/30" />

                  {/* Step 5: Cognitive Router */}
                  <div className="flex items-center justify-center">
                    <div className="w-41 bg-slate-950 p-4 rounded-xl border border-indigo-600 flex flex-col items-center text-center gap-2 shadow-lg relative bg-indigo-950/10">
                      <Cpu size={20} className="text-indigo-400 animate-spin" style={{ animationDuration: "12s" }} />
                      <div className="text-xs font-bold text-white">5. Router Solver</div>
                      <div className="text-[9px] text-slate-500">Cost/Capability metrics</div>
                    </div>
                  </div>
                </div>

                <div className="bg-slate-950 p-4 rounded-xl border border-slate-800/80 mt-4">
                  <div className="text-[10px] text-indigo-400 font-mono uppercase mb-1.5 font-bold">Dynamic Execution Framework Specs</div>
                  <p className="text-xs text-slate-400 font-mono leading-relaxed max-w-4xl">
                    Our modular architecture separates concern scopes. All data models are defined inside central type manifests, and database queries are fully encapsulated to eliminate deadlocks. Failover logic operates as a middleware layers, monitoring downstream API HTTP codes (429, 503) and invoking live target substitutions in less than 50 milliseconds.
                  </p>
                </div>
              </div>
            )}
          </motion.div>
        </AnimatePresence>
      </div>
    </div>
  );
}
