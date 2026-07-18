import React, { useState, useEffect } from "react";
import { motion, AnimatePresence } from "motion/react";
import { ServiceNode } from "../types";
import { Server, Shield, Activity, Database, Cpu, Plus, Minus, RefreshCw, AlertTriangle, Play, Pause } from "lucide-react";

interface TopologyProps {
  nodes: ServiceNode[];
  onUpdateNode: (nodeId: string, updates: Partial<ServiceNode>) => void;
  onTriggerEvent: (msg: string, type: "info" | "warn" | "error") => void;
  activeWorkload: string;
}

export default function TopologyView({ nodes, onUpdateNode, onTriggerEvent, activeWorkload }: TopologyProps) {
  const [selectedNodeId, setSelectedNodeId] = useState<string>("api-gateway");
  const [particleTrigger, setParticleTrigger] = useState<number>(0);

  // Trigger flowing requests visual based on active workload
  useEffect(() => {
    const interval = setInterval(() => {
      setParticleTrigger(prev => (prev + 1) % 100);
    }, activeWorkload === "high-traffic" ? 400 : activeWorkload === "normal" ? 1000 : 2000);
    return () => clearInterval(interval);
  }, [activeWorkload]);

  const selectedNode = nodes.find(n => n.id === selectedNodeId) || nodes[0];

  const handleScale = (increment: boolean) => {
    if (!selectedNode) return;
    const current = selectedNode.replicas.active;
    const max = selectedNode.replicas.total;
    let next = increment ? current + 1 : current - 1;
    if (next < 0) next = 0;
    if (next > max) next = max;

    if (next !== current) {
      onUpdateNode(selectedNode.id, {
        replicas: { ...selectedNode.replicas, active: next },
        status: next === 0 ? "OFFLINE" : next < max ? "DEGRADED" : "HEALTHY"
      });
      onTriggerEvent(
        `Kubernetes HPA scaled ${selectedNode.name} to ${next}/${max} pods`,
        next === 0 ? "error" : next < max ? "warn" : "info"
      );
    }
  };

  const handleTogglePower = () => {
    if (!selectedNode) return;
    const isOffline = selectedNode.status === "OFFLINE";
    const nextStatus = isOffline ? "HEALTHY" : "OFFLINE";
    const nextActive = isOffline ? selectedNode.replicas.total : 0;

    onUpdateNode(selectedNode.id, {
      status: nextStatus,
      replicas: { ...selectedNode.replicas, active: nextActive }
    });

    onTriggerEvent(
      isOffline 
        ? `Manual recovery initiated for ${selectedNode.name}. Kubernetes launching pods...` 
        : `CRITICAL ALERT: Node ${selectedNode.name} terminated. All pods stopped!`,
      isOffline ? "info" : "error"
    );
  };

  const handleRestart = () => {
    if (!selectedNode) return;
    onUpdateNode(selectedNode.id, { status: "DEGRADED", replicas: { ...selectedNode.replicas, active: 1 } });
    onTriggerEvent(`Rolling restart triggered for deployment/${selectedNode.id}. Commencing rollout...`, "warn");
    
    setTimeout(() => {
      onUpdateNode(selectedNode.id, { status: "HEALTHY", replicas: { ...selectedNode.replicas, active: selectedNode.replicas.total } });
      onTriggerEvent(`Rolling restart completed for deployment/${selectedNode.id}. All pods ready.`, "info");
    }, 4000);
  };

  // Node position helper inside responsive SVG coordinate box
  const nodeCoords: Record<string, { x: number; y: number }> = {
    "ingress": { x: 50, y: 150 },
    "api-gateway": { x: 220, y: 150 },
    "auth-service": { x: 420, y: 70 },
    "compute-engine": { x: 420, y: 230 },
    "postgres-db": { x: 620, y: 230 },
    "telemetry-collector": { x: 420, y: 350 },
  };

  const getNodeIcon = (id: string, size = 20) => {
    switch (id) {
      case "ingress": return <Server size={size} />;
      case "api-gateway": return <Cpu size={size} className="text-indigo-400" />;
      case "auth-service": return <Shield size={size} className="text-amber-400" />;
      case "compute-engine": return <Activity size={size} className="text-emerald-400" />;
      case "postgres-db": return <Database size={size} className="text-purple-400" />;
      case "telemetry-collector": return <Cpu size={size} className="text-rose-400" />;
      default: return <Server size={size} />;
    }
  };

  const getNodeStatusColor = (status: string) => {
    switch (status) {
      case "HEALTHY": return "border-emerald-500/50 text-emerald-400 bg-emerald-950/20";
      case "DEGRADED": return "border-amber-500/50 text-amber-400 bg-amber-950/20";
      case "CRITICAL": return "border-rose-500/50 text-rose-400 bg-rose-950/20";
      case "OFFLINE": return "border-slate-700 text-slate-500 bg-slate-900/40";
      default: return "border-slate-500 text-slate-400 bg-slate-900";
    }
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 h-full" id="nexuscore-topology-root">
      {/* Topology Canvas Panel */}
      <div className="lg:col-span-2 bg-slate-950 rounded-xl border border-slate-800 p-5 relative overflow-hidden flex flex-col justify-between h-[520px]">
        <div>
          <div className="flex items-center justify-between border-b border-slate-900 pb-3">
            <div>
              <h3 className="font-display font-semibold text-white tracking-wide">NexusCore Namespace Topology</h3>
              <p className="text-xs text-slate-400 font-sans mt-0.5">Live Kubernetes namespace: <span className="font-mono text-indigo-400 bg-indigo-950/30 px-1.5 py-0.5 rounded">nexus-core</span></p>
            </div>
            <div className="flex items-center gap-3">
              <span className="flex items-center gap-1.5 text-xs text-slate-400">
                <span className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse" /> Live Mesh Active
              </span>
            </div>
          </div>
        </div>

        {/* Interactive SVG Diagram */}
        <div className="relative flex-1 flex items-center justify-center p-4">
          <svg viewBox="0 0 720 420" className="w-full h-full max-h-[380px] drop-shadow-2xl">
            {/* SVG Filter for gorgeous glows */}
            <defs>
              <filter id="glow" x="-20%" y="-20%" width="140%" height="140%">
                <feGaussianBlur stdDeviation="4" result="blur" />
                <feComposite in="SourceGraphic" in2="blur" operator="over" />
              </filter>
            </defs>

            {/* Network Connections / Lines */}
            {/* Ingress -> API Gateway */}
            <line x1={nodeCoords.ingress.x} y1={nodeCoords.ingress.y} x2={nodeCoords["api-gateway"].x} y2={nodeCoords["api-gateway"].y} stroke="#1e293b" strokeWidth="2" strokeDasharray="4 4" />
            
            {/* Gateway -> Auth */}
            <path d={`M ${nodeCoords["api-gateway"].x} ${nodeCoords["api-gateway"].y} C 320 150, 320 70, ${nodeCoords["auth-service"].x} ${nodeCoords["auth-service"].y}`} fill="none" stroke={nodes.find(n => n.id === "auth-service")?.status === "OFFLINE" ? "#475569" : "#6366f1"} strokeWidth="1.5" opacity="0.6" />
            
            {/* Gateway -> Compute */}
            <path d={`M ${nodeCoords["api-gateway"].x} ${nodeCoords["api-gateway"].y} C 320 150, 320 230, ${nodeCoords["compute-engine"].x} ${nodeCoords["compute-engine"].y}`} fill="none" stroke={nodes.find(n => n.id === "compute-engine")?.status === "OFFLINE" ? "#475569" : "#10b981"} strokeWidth="1.5" opacity="0.6" />
            
            {/* Compute -> DB */}
            <line x1={nodeCoords["compute-engine"].x} y1={nodeCoords["compute-engine"].y} x2={nodeCoords["postgres-db"].x} y2={nodeCoords["postgres-db"].y} stroke={nodes.find(n => n.id === "postgres-db")?.status === "OFFLINE" ? "#475569" : "#a855f7"} strokeWidth="1.5" opacity="0.6" />
            
            {/* Gateway & Services -> Telemetry Collector */}
            <path d={`M ${nodeCoords["api-gateway"].x} ${nodeCoords["api-gateway"].y} Q 320 350, ${nodeCoords["telemetry-collector"].x} ${nodeCoords["telemetry-collector"].y}`} fill="none" stroke="#e11d48" strokeWidth="1" strokeDasharray="3 3" opacity="0.4" />
            <line x1={nodeCoords["compute-engine"].x} y1={nodeCoords["compute-engine"].y} x2={nodeCoords["telemetry-collector"].x} y2={nodeCoords["telemetry-collector"].y} stroke="#e11d48" strokeWidth="1" strokeDasharray="3 3" opacity="0.4" />

            {/* Simulated request particles flowing through the wires */}
            <AnimatePresence>
              {activeWorkload !== "outage" && (
                <>
                  {/* Gateway -> Auth */}
                  {nodes.find(n => n.id === "auth-service")?.status !== "OFFLINE" && (
                    <motion.circle r="3" fill="#818cf8" filter="url(#glow)" initial={{ offset: 0 }} animate={{ offset: 1 }} transition={{ repeat: Infinity, duration: 1.8, ease: "linear" }}>
                      <animateMotion path={`M ${nodeCoords["api-gateway"].x} ${nodeCoords["api-gateway"].y} C 320 150, 320 70, ${nodeCoords["auth-service"].x} ${nodeCoords["auth-service"].y}`} dur="1.8s" repeatCount="indefinite" />
                    </motion.circle>
                  )}

                  {/* Gateway -> Compute */}
                  {nodes.find(n => n.id === "compute-engine")?.status !== "OFFLINE" && (
                    <motion.circle r="3" fill="#34d399" filter="url(#glow)" transition={{ repeat: Infinity, duration: 1.5, ease: "linear" }}>
                      <animateMotion path={`M ${nodeCoords["api-gateway"].x} ${nodeCoords["api-gateway"].y} C 320 150, 320 230, ${nodeCoords["compute-engine"].x} ${nodeCoords["compute-engine"].y}`} dur="1.5s" repeatCount="indefinite" />
                    </motion.circle>
                  )}

                  {/* Compute -> DB */}
                  {nodes.find(n => n.id === "compute-engine")?.status !== "OFFLINE" && nodes.find(n => n.id === "postgres-db")?.status !== "OFFLINE" && (
                    <motion.circle r="2.5" fill="#c084fc" filter="url(#glow)" transition={{ repeat: Infinity, duration: 1.2, ease: "linear" }}>
                      <animateMotion path={`M ${nodeCoords["compute-engine"].x} ${nodeCoords["compute-engine"].y} L ${nodeCoords["postgres-db"].x} ${nodeCoords["postgres-db"].y}`} dur="1.2s" repeatCount="indefinite" />
                    </motion.circle>
                  )}
                </>
              )}
            </AnimatePresence>

            {/* Ingress node */}
            <g transform={`translate(${nodeCoords.ingress.x - 22}, ${nodeCoords.ingress.y - 22})`}>
              <rect width="44" height="44" rx="8" fill="#1e293b" stroke="#334155" strokeWidth="2" className="cursor-default" />
              <foreignObject x="12" y="12" width="20" height="20">
                <div className="text-slate-400 flex items-center justify-center">
                  <Server size={18} />
                </div>
              </foreignObject>
              <text x="22" y="58" fill="#94a3b8" fontSize="10" fontFamily="sans-serif" textAnchor="middle" fontWeight="bold">Ingress</text>
            </g>

            {/* Render interactive Service Nodes */}
            {nodes.map(node => {
              const coords = nodeCoords[node.id];
              if (!coords) return null;
              const isSelected = selectedNodeId === node.id;
              const isOffline = node.status === "OFFLINE";
              const isDegraded = node.status === "DEGRADED";

              return (
                <g key={node.id} transform={`translate(${coords.x - 28}, ${coords.y - 28})`} className="cursor-pointer group" onClick={() => setSelectedNodeId(node.id)}>
                  {/* Selected Pulse Ring */}
                  {isSelected && (
                    <circle cx="28" cy="28" r="34" fill="none" stroke="#6366f1" strokeWidth="1.5" className="animate-pulse" opacity="0.7" />
                  )}

                  {/* Standard Node Box */}
                  <rect width="56" height="56" rx="12" fill={isOffline ? "#0f172a" : "#020617"} stroke={isSelected ? "#6366f1" : isOffline ? "#334155" : isDegraded ? "#f59e0b" : "#1e293b"} strokeWidth={isSelected ? 2.5 : 2} className="transition-all duration-300 group-hover:border-slate-500" />

                  {/* Inner Icon Container */}
                  <rect x="8" y="8" width="40" height="40" rx="8" fill={isOffline ? "#1e293b/50" : "rgba(30, 41, 59, 0.2)"} className="transition-all" />

                  <foreignObject x="18" y="18" width="20" height="20">
                    <div className="flex items-center justify-center">
                      {getNodeIcon(node.id, 20)}
                    </div>
                  </foreignObject>

                  {/* Small Health Status indicator dot */}
                  <circle cx="48" cy="8" r="5" fill={isOffline ? "#64748b" : isDegraded ? "#f59e0b" : "#10b981"} stroke="#020617" strokeWidth="1" />

                  {/* Node Label */}
                  <text x="28" y="70" fill={isSelected ? "#818cf8" : "#e2e8f0"} fontSize="10" fontFamily="sans-serif" textAnchor="middle" fontWeight={isSelected ? "bold" : "normal"}>
                    {node.name.length > 11 ? `${node.name.substring(0, 9)}..` : node.name}
                  </text>

                  {/* Port Info Label */}
                  <text x="28" y="82" fill="#64748b" fontSize="8" fontFamily="monospace" textAnchor="middle">
                    :{node.port}
                  </text>
                </g>
              );
            })}
          </svg>
        </div>

        {/* Node quick overview footer */}
        <div className="bg-slate-900/40 border border-slate-900 rounded-lg p-3 grid grid-cols-3 gap-2">
          <div className="text-center border-r border-slate-950">
            <p className="text-[10px] text-slate-500 font-sans uppercase tracking-wider font-semibold">Active Pods</p>
            <p className="font-mono text-base font-bold text-slate-200 mt-0.5">
              {nodes.reduce((acc, curr) => acc + (curr.status !== "OFFLINE" ? curr.replicas.active : 0), 0)} / {nodes.reduce((acc, curr) => acc + curr.replicas.total, 0)}
            </p>
          </div>
          <div className="text-center border-r border-slate-950">
            <p className="text-[10px] text-slate-500 font-sans uppercase tracking-wider font-semibold">Mesh Health</p>
            <p className="font-mono text-base font-bold text-emerald-400 mt-0.5">
              {nodes.filter(n => n.status === "HEALTHY").length === nodes.length 
                ? "100%" 
                : nodes.filter(n => n.status === "OFFLINE").length > 0 
                  ? `${Math.round((nodes.filter(n => n.status === "HEALTHY").length / nodes.length) * 100)}%`
                  : "DEGRADED"}
            </p>
          </div>
          <div className="text-center">
            <p className="text-[10px] text-slate-500 font-sans uppercase tracking-wider font-semibold">Gateway Ingress</p>
            <p className="font-mono text-base font-bold text-indigo-400 mt-0.5">ACTIVE</p>
          </div>
        </div>
      </div>

      {/* Control Panel Sidebar */}
      <div className="bg-slate-950 rounded-xl border border-slate-800 p-5 flex flex-col justify-between h-[520px]">
        <div>
          <div className="border-b border-slate-900 pb-3 mb-4">
            <h3 className="font-display font-semibold text-white tracking-wide">Deployment Controller</h3>
            <p className="text-xs text-slate-400 font-sans mt-0.5">Manage replica sets and pods state</p>
          </div>

          <div className="space-y-4">
            {/* Service Stats */}
            <div className="bg-slate-900/60 rounded-lg p-4 border border-slate-900">
              <div className="flex items-center gap-2.5">
                <div className="p-1.5 rounded-md bg-slate-800 text-indigo-400 font-bold">
                  {getNodeIcon(selectedNode.id, 18)}
                </div>
                <div>
                  <h4 className="font-semibold text-sm text-slate-200">{selectedNode.name}</h4>
                  <p className="text-[10px] text-slate-500 font-mono">deployment.apps/{selectedNode.id}</p>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3 mt-4 pt-4 border-t border-slate-950">
                <div>
                  <span className="text-[10px] text-slate-500 block uppercase font-semibold">Target Port</span>
                  <span className="text-xs font-mono text-slate-300 font-bold">{selectedNode.port} (TCP)</span>
                </div>
                <div>
                  <span className="text-[10px] text-slate-500 block uppercase font-semibold">Replica Status</span>
                  <span className={`text-xs font-mono font-bold ${selectedNode.status === "OFFLINE" ? "text-rose-400" : "text-emerald-400"}`}>
                    {selectedNode.replicas.active} / {selectedNode.replicas.total} Ready
                  </span>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3 mt-3">
                <div>
                  <span className="text-[10px] text-slate-500 block uppercase font-semibold">CPU Requests</span>
                  <span className="text-xs font-mono text-slate-300 font-bold">{selectedNode.cpu}% of 500m</span>
                </div>
                <div>
                  <span className="text-[10px] text-slate-500 block uppercase font-semibold">Memory Requests</span>
                  <span className="text-xs font-mono text-slate-300 font-bold">{selectedNode.mem}Mi of 512Mi</span>
                </div>
              </div>

              <div className="mt-4">
                <span className="text-[10px] text-slate-500 block uppercase font-semibold mb-1">State Conditions</span>
                <div className="flex flex-wrap gap-1.5">
                  <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded ${selectedNode.status === "OFFLINE" ? "bg-rose-950/40 text-rose-400 border border-rose-900/50" : "bg-emerald-950/40 text-emerald-400 border border-emerald-900/50"}`}>
                    {selectedNode.status === "OFFLINE" ? "PodFailed" : "MinimumReplicasAvailable"}
                  </span>
                  <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-slate-900 text-slate-400 border border-slate-800">
                    Progressing
                  </span>
                </div>
              </div>
            </div>

            {/* Scaling Controller */}
            {selectedNode.id !== "ingress" && (
              <div className="space-y-2.5">
                <label className="text-[10px] text-slate-400 uppercase tracking-wider font-bold">Manual Scaling (HPA Override)</label>
                <div className="flex items-center gap-3 bg-slate-900/40 rounded-lg p-1.5 border border-slate-900">
                  <button onClick={() => handleScale(false)} disabled={selectedNode.status === "OFFLINE" || selectedNode.replicas.active <= 1} className="p-2 bg-slate-800 hover:bg-slate-700 disabled:opacity-40 text-slate-300 rounded transition-colors">
                    <Minus size={14} />
                  </button>
                  <div className="flex-1 text-center">
                    <span className="text-sm font-mono text-white font-bold">{selectedNode.replicas.active}</span>
                    <span className="text-xs text-slate-500 font-sans ml-1">replicas</span>
                  </div>
                  <button onClick={() => handleScale(true)} disabled={selectedNode.status === "OFFLINE" || selectedNode.replicas.active >= selectedNode.replicas.total} className="p-2 bg-slate-800 hover:bg-slate-700 disabled:opacity-40 text-slate-300 rounded transition-colors">
                    <Plus size={14} />
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Action Controls */}
        <div className="space-y-2 pt-4 border-t border-slate-900">
          <button onClick={handleTogglePower} className={`w-full py-2.5 px-4 rounded-lg font-sans font-semibold text-xs transition-all duration-300 flex items-center justify-center gap-2 border ${selectedNode.status === "OFFLINE" ? "bg-emerald-500 hover:bg-emerald-600 text-slate-950 border-emerald-400" : "bg-rose-950/20 hover:bg-rose-950/60 text-rose-400 border-rose-900/50"}`}>
            {selectedNode.status === "OFFLINE" ? <Play size={14} fill="currentColor" /> : <Pause size={14} fill="currentColor" />}
            {selectedNode.status === "OFFLINE" ? `Deploy deployment/${selectedNode.id}` : `Terminate deployment/${selectedNode.id}`}
          </button>
          
          <button onClick={handleRestart} disabled={selectedNode.status === "OFFLINE"} className="w-full py-2.5 px-4 bg-slate-900 hover:bg-slate-850 border border-slate-800 text-slate-300 hover:text-white rounded-lg font-sans font-semibold text-xs transition-colors disabled:opacity-40 flex items-center justify-center gap-2">
            <RefreshCw size={14} className="animate-spin-slow" />
            Perform Rolling Restart
          </button>
        </div>
      </div>
    </div>
  );
}
