import { useState, useEffect } from "react";
import { motion, AnimatePresence } from "motion/react";
import {
  Shield, Key, Lock, RefreshCw, UserCheck, FileText, AlertTriangle,
  Server, Globe, Activity, Terminal, Plus, Trash, Play, CheckCircle2,
  XCircle, Eye, EyeOff, LockKeyhole, History, Settings, Database,
  Network, Copy, Check
} from "lucide-react";
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer
} from "recharts";

// TypeScript declarations matching backend models
interface SpiffeSvid {
  spiffeId: string;
  trustDomain: string;
  serviceName: string;
  serialNumber: string;
  validFrom: string;
  validTo: string;
  fingerprint: string;
  status: "active" | "expired" | "rotated";
  certificateChain: string[];
}

interface SvidRotationLog {
  timestamp: string;
  event: string;
  spiffeId: string;
  details: string;
  operator: string;
}

interface VaultSecret {
  path: string;
  version: number;
  data: Record<string, string>;
  updatedAt: string;
  leaseDurationSeconds: number;
  isRotatable: boolean;
  rotationHistory: Array<{
    version: number;
    timestamp: string;
    fields: string[];
    operator: string;
  }>;
}

interface VaultKey {
  name: string;
  type: string;
  status: string;
  createdAt: string;
  lastRotated: string;
}

interface KeycloakUser {
  id: string;
  username: string;
  email: string;
  roles: string[];
  groups: string[];
  mfaEnabled: boolean;
  status: string;
}

interface JwtTokenDetail {
  header: Record<string, string>;
  payload: {
    iss: string;
    sub: string;
    aud: string;
    exp: number;
    iat: number;
    jti: string;
    username: string;
    email: string;
    roles: string[];
    groups: string[];
    context: string;
  };
  signature: string;
  raw: string;
}

interface OpaPolicy {
  id: string;
  name: string;
  regoCode: string;
  isActive: boolean;
  updatedAt: string;
}

interface ApiKeyDetail {
  id: string;
  name: string;
  prefix: string;
  hash: string;
  ownerService: string;
  allowedMethods: string[];
  rateLimitRps: number;
  status: "active" | "revoked";
  createdAt: string;
}

interface SecurityAuditEvent {
  id: string;
  timestamp: string;
  category: "authentication" | "authorization" | "secrets" | "network_mtls" | "rate_limiting";
  action: string;
  actor: string;
  service: string;
  status: "success" | "failure" | "blocked";
  details: string;
  severity: "info" | "warning" | "critical";
  clientIp: string;
}

interface RateLimitMetric {
  timestamp: string;
  clientId: string;
  totalRequests: number;
  blockedRequests: number;
  rps: number;
}

export default function ZeroTrustDashboard() {
  // Navigation states
  const [activeSubTab, setActiveSubTab] = useState<"spiffe" | "vault" | "keycloak" | "opa" | "ratelimit" | "audit">("spiffe");

  // Domain state variables
  const [svids, setSvids] = useState<SpiffeSvid[]>([]);
  const [svidLogs, setSvidLogs] = useState<SvidRotationLog[]>([]);
  const [selectedSvid, setSelectedSvid] = useState<SpiffeSvid | null>(null);
  const [isRotatingSvid, setIsRotatingSvid] = useState<string | null>(null);
  const [svidTraceLogs, setSvidTraceLogs] = useState<string[]>([]);

  // Vault state variables
  const [secrets, setSecrets] = useState<VaultSecret[]>([]);
  const [selectedSecret, setSelectedSecret] = useState<VaultSecret | null>(null);
  const [transitKeys, setTransitKeys] = useState<VaultKey[]>([]);
  const [isRotatingSecret, setIsRotatingSecret] = useState<string | null>(null);
  const [secretTraceLogs, setSecretTraceLogs] = useState<string[]>([]);
  const [isAddingSecret, setIsAddingSecret] = useState(false);
  const [newSecretPath, setNewSecretPath] = useState("/secret/data/");
  const [newSecretData, setNewSecretData] = useState<Array<{ k: string; v: string }>>([{ k: "", v: "" }]);
  
  // Transit Crypto sandbox states
  const [selectedTransitKey, setSelectedTransitKey] = useState<string>("payment-token-key");
  const [plainCryptoText, setPlainCryptoText] = useState("4532-7182-9018-4412");
  const [cipherCryptoText, setCipherCryptoText] = useState("");
  const [decryptedText, setDecryptedText] = useState("");

  // Keycloak states
  const [kcUsers, setKcUsers] = useState<KeycloakUser[]>([]);
  const [selectedKcUser, setSelectedKcUser] = useState<KeycloakUser | null>(null);
  const [activeJwt, setActiveJwt] = useState<JwtTokenDetail | null>(null);
  const [isIssuingJwt, setIsIssuingJwt] = useState(false);

  // OPA states
  const [opaPolicies, setOpaPolicies] = useState<OpaPolicy[]>([]);
  const [selectedOpaPolicy, setSelectedOpaPolicy] = useState<OpaPolicy | null>(null);
  const [isEditingRego, setIsEditingRego] = useState(false);
  const [editedRegoCode, setEditedRegoCode] = useState("");
  
  // OPA Evaluator inputs
  const [opaSubjectUser, setOpaSubjectUser] = useState("bob_developer");
  const [opaSubjectRoles, setOpaSubjectRoles] = useState<string[]>(["developer"]);
  const [opaResourcePath, setOpaResourcePath] = useState("/secret/data/database/credentials");
  const [opaResourceMethod, setOpaResourceMethod] = useState("POST");
  const [opaEvaluationResult, setOpaEvaluationResult] = useState<any | null>(null);
  const [isEvaluatingOpa, setIsEvaluatingOpa] = useState(false);

  // API Key & Rate Limit states
  const [apiKeys, setApiKeys] = useState<ApiKeyDetail[]>([]);
  const [newApiKeyName, setNewApiKeyName] = useState("");
  const [newApiKeyService, setNewApiKeyService] = useState("order-service");
  const [newApiKeyLimit, setNewApiKeyLimit] = useState(15);
  const [createdPlainKey, setCreatedPlainKey] = useState<string | null>(null);
  const [rateLimitMetrics, setRateLimitMetrics] = useState<RateLimitMetric[]>([]);
  const [rateLimitConfig, setRateLimitConfig] = useState<any>(null);
  const [isSimulatingRateLimit, setIsSimulatingRateLimit] = useState(false);

  // Audit Logs
  const [auditLogs, setAuditLogs] = useState<SecurityAuditEvent[]>([]);
  const [auditFilterCategory, setAuditFilterCategory] = useState<string>("all");
  const [auditFilterSeverity, setAuditFilterSeverity] = useState<string>("all");

  // Global triggers / helpers
  const [copiedText, setCopiedText] = useState<string | null>(null);

  useEffect(() => {
    fetchInitialData();
    const interval = setInterval(refreshTelemetry, 5000);
    return () => clearInterval(interval);
  }, []);

  const fetchInitialData = async () => {
    try {
      const [
        svidRes, secretsRes, keysRes, usersRes, policiesRes, apiKeysRes, logsRes, limitRes
      ] = await Promise.all([
        fetch("/api/security/svids"),
        fetch("/api/security/secrets"),
        fetch("/api/security/transit-keys"),
        fetch("/api/security/users"),
        fetch("/api/security/opa/policies"),
        fetch("/api/security/api-keys"),
        fetch("/api/security/audit-logs"),
        fetch("/api/security/rate-limit/metrics")
      ]);

      const svidData = await svidRes.json();
      setSvids(svidData.svids);
      setSvidLogs(svidData.rotationLogs);
      if (svidData.svids.length > 0) setSelectedSvid(svidData.svids[0]);

      const secretsData = await secretsRes.json();
      setSecrets(secretsData);
      if (secretsData.length > 0) setSelectedSecret(secretsData[0]);

      setTransitKeys(await keysRes.json());
      
      const usersData = await usersRes.json();
      setKcUsers(usersData);
      if (usersData.length > 0) {
        setSelectedKcUser(usersData[1]); // default to developer
        handleIssueJwt(usersData[1].id);
      }

      const policiesData = await policiesRes.json();
      setOpaPolicies(policiesData);
      if (policiesData.length > 0) {
        setSelectedOpaPolicy(policiesData[0]);
        setEditedRegoCode(policiesData[0].regoCode);
      }

      setApiKeys(await apiKeysRes.json());
      setAuditLogs(await logsRes.json());
      
      const limitData = await limitRes.json();
      setRateLimitMetrics(limitData.metrics);
      setRateLimitConfig(limitData.config);
    } catch (err) {
      console.error("Failed to fetch initial security configuration data", err);
    }
  };

  const refreshTelemetry = async () => {
    try {
      const [logsRes, limitRes] = await Promise.all([
        fetch("/api/security/audit-logs"),
        fetch("/api/security/rate-limit/metrics")
      ]);
      setAuditLogs(await logsRes.json());
      const limitData = await limitRes.json();
      setRateLimitMetrics(limitData.metrics);
    } catch (err) {
      console.error("Failed telemetry refresh", err);
    }
  };

  // Helper copy to clipboard
  const handleCopy = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedText(id);
    setTimeout(() => setCopiedText(null), 2000);
  };

  // Rotate SVID
  const handleRotateSvid = async (spiffeId: string) => {
    setIsRotatingSvid(spiffeId);
    setSvidTraceLogs([]);
    try {
      const response = await fetch("/api/security/svids/rotate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ spiffeId })
      });
      const data = await response.json();
      
      // Animate execution log traces incrementally
      for (let i = 0; i < data.trace.length; i++) {
        await new Promise(resolve => setTimeout(resolve, 350));
        setSvidTraceLogs(prev => [...prev, data.trace[i]]);
      }
      
      // Update local mapping
      const updatedSvids = svids.map(s => s.spiffeId === spiffeId ? data.svid : s);
      setSvids(updatedSvids);
      if (selectedSvid?.spiffeId === spiffeId) {
        setSelectedSvid(data.svid);
      }

      // Refresh log list
      const svidDataRes = await fetch("/api/security/svids");
      const svidData = await svidDataRes.json();
      setSvidLogs(svidData.rotationLogs);
      
      // Reload Audit logs
      const logsRes = await fetch("/api/security/audit-logs");
      setAuditLogs(await logsRes.json());
    } catch (err) {
      console.error(err);
    } finally {
      setIsRotatingSvid(null);
    }
  };

  // Rotate Secret
  const handleRotateSecret = async (path: string, payload: Record<string, string>) => {
    setIsRotatingSecret(path);
    setSecretTraceLogs([]);
    try {
      const response = await fetch("/api/security/secrets/rotate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path, data: payload })
      });
      const data = await response.json();

      for (let i = 0; i < data.trace.length; i++) {
        await new Promise(resolve => setTimeout(resolve, 350));
        setSecretTraceLogs(prev => [...prev, data.trace[i]]);
      }

      const updatedSecrets = secrets.map(s => s.path === path ? data.secret : s);
      setSecrets(updatedSecrets);
      setSelectedSecret(data.secret);

      const logsRes = await fetch("/api/security/audit-logs");
      setAuditLogs(await logsRes.json());
    } catch (err) {
      console.error(err);
    } finally {
      setIsRotatingSecret(null);
    }
  };

  // Add custom path secret
  const handleAddSecret = async () => {
    const payload: Record<string, string> = {};
    newSecretData.forEach(item => {
      if (item.k.trim()) payload[item.k.trim()] = item.v;
    });

    if (!newSecretPath.startsWith("/secret/data/") || newSecretPath.trim() === "/secret/data/") {
      alert("Secret path must be a qualified vault URI matching structure '/secret/data/...'");
      return;
    }

    try {
      const response = await fetch("/api/security/secrets/rotate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: newSecretPath.trim(), data: payload })
      });
      const data = await response.json();
      
      setSecrets(prev => [...prev.filter(s => s.path !== data.secret.path), data.secret]);
      setSelectedSecret(data.secret);
      setIsAddingSecret(false);
      setNewSecretPath("/secret/data/");
      setNewSecretData([{ k: "", v: "" }]);

      const logsRes = await fetch("/api/security/audit-logs");
      setAuditLogs(await logsRes.json());
    } catch (err) {
      console.error(err);
    }
  };

  // Rotate Transit Key
  const handleRotateTransitKey = async (name: string) => {
    try {
      const response = await fetch("/api/security/transit-keys/rotate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name })
      });
      const data = await response.json();
      setTransitKeys(transitKeys.map(k => k.name === name ? data.key : k));

      const logsRes = await fetch("/api/security/audit-logs");
      setAuditLogs(await logsRes.json());
    } catch (err) {
      console.error(err);
    }
  };

  // Interactive local Cryptography simulation using SHA256/AES key rotation concepts
  const handleEncryptPlaintext = () => {
    if (!plainCryptoText) return;
    const key = transitKeys.find(k => k.name === selectedTransitKey);
    const lastRotatedMs = key ? new Date(key.lastRotated).getTime() : Date.now();
    
    // Simulate high-fidelity Vault Transit ciphertext structure: vault:v[version]:[base64-payload]
    const hash = btoa(encodeURIComponent(plainCryptoText + lastRotatedMs));
    const vaultCiphertext = `vault:v1:${hash.replace(/[^a-zA-Z0-9]/g, "").substring(0, 32)}`;
    
    setCipherCryptoText(vaultCiphertext);
    setDecryptedText("");
  };

  const handleDecryptCiphertext = () => {
    if (!cipherCryptoText) return;
    setDecryptedText(plainCryptoText); // Symmetric retrieval simulation
  };

  // Issue Keycloak JWT
  const handleIssueJwt = async (userId: string) => {
    setIsIssuingJwt(true);
    try {
      const response = await fetch("/api/security/users/jwt", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ userId })
      });
      const tokenDetail = await response.json();
      setActiveJwt(tokenDetail);

      // Map roles/paths dynamically to test sandbox
      const user = kcUsers.find(u => u.id === userId);
      if (user) {
        setOpaSubjectUser(user.username);
        setOpaSubjectRoles(user.roles);
      }

      const logsRes = await fetch("/api/security/audit-logs");
      setAuditLogs(await logsRes.json());
    } catch (err) {
      console.error(err);
    } finally {
      setIsIssuingJwt(false);
    }
  };

  // Save modified Rego policies
  const handleSaveOpaPolicy = async () => {
    if (!selectedOpaPolicy) return;
    try {
      const response = await fetch("/api/security/opa/policies", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id: selectedOpaPolicy.id, regoCode: editedRegoCode })
      });
      const data = await response.json();
      setOpaPolicies(opaPolicies.map(p => p.id === selectedOpaPolicy.id ? data.policy : p));
      setSelectedOpaPolicy(data.policy);
      setIsEditingRego(false);

      const logsRes = await fetch("/api/security/audit-logs");
      setAuditLogs(await logsRes.json());
    } catch (err) {
      console.error(err);
    }
  };

  // Evaluate OPA policies
  const handleEvaluateOpa = async () => {
    if (!selectedOpaPolicy) return;
    setIsEvaluatingOpa(true);
    try {
      const response = await fetch("/api/security/opa/evaluate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          policyId: selectedOpaPolicy.id,
          request: {
            subject: {
              user: opaSubjectUser,
              roles: opaSubjectRoles,
              groups: opaSubjectRoles.includes("admin") ? ["secops-team"] : ["checkout-team"],
              spiffeId: `spiffe://nexuscore.io/ns/prod/sa/${opaSubjectUser.replace("_", "-")}`
            },
            resource: {
              path: opaResourcePath,
              method: opaResourceMethod,
              service: "ingress-gateway"
            }
          }
        })
      });
      const result = await response.json();
      setOpaEvaluationResult(result);

      const logsRes = await fetch("/api/security/audit-logs");
      setAuditLogs(await logsRes.json());
    } catch (err) {
      console.error(err);
    } finally {
      setIsEvaluatingOpa(false);
    }
  };

  // Issue dynamic client API keys
  const handleCreateApiKey = async () => {
    if (!newApiKeyName) return;
    try {
      const response = await fetch("/api/security/api-keys", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: newApiKeyName,
          ownerService: newApiKeyService,
          allowedMethods: ["GET", "POST"],
          rateLimitRps: newApiKeyLimit
        })
      });
      const data = await response.json();
      setApiKeys(prev => [...prev, data.key]);
      setCreatedPlainKey(data.plainKey);
      setNewApiKeyName("");

      const logsRes = await fetch("/api/security/audit-logs");
      setAuditLogs(await logsRes.json());
    } catch (err) {
      console.error(err);
    }
  };

  const handleRevokeApiKey = async (id: string) => {
    try {
      const response = await fetch("/api/security/api-keys/revoke", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id })
      });
      const data = await response.json();
      setApiKeys(apiKeys.map(k => k.id === id ? data.key : k));

      const logsRes = await fetch("/api/security/audit-logs");
      setAuditLogs(await logsRes.json());
    } catch (err) {
      console.error(err);
    }
  };

  // Simulate Rate Limiter Throttling flow
  const handleSimulateRateLimit = async () => {
    setIsSimulatingRateLimit(true);
    try {
      const response = await fetch("/api/security/rate-limit/simulate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ clientId: "auth-gateway" })
      });
      const data = await response.json();
      
      // Refresh metrics immediately
      const limitRes = await fetch("/api/security/rate-limit/metrics");
      const limitData = await limitRes.json();
      setRateLimitMetrics(limitData.metrics);

      const logsRes = await fetch("/api/security/audit-logs");
      setAuditLogs(await logsRes.json());
    } catch (err) {
      console.error(err);
    } finally {
      setIsSimulatingRateLimit(false);
    }
  };

  // Filter logs safely
  const filteredAuditLogs = auditLogs.filter(log => {
    const matchesCat = auditFilterCategory === "all" || log.category === auditFilterCategory;
    const matchesSev = auditFilterSeverity === "all" || log.severity === auditFilterSeverity;
    return matchesCat && matchesSev;
  });

  return (
    <div id="zero-trust-dashboard" className="space-y-6 text-slate-100 font-sans">
      {/* SECTION HEADER: METRICS GRID */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-4 flex items-center justify-between shadow-xl">
          <div className="space-y-1">
            <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider block">Active SVIDs (mTLS)</span>
            <span className="text-2xl font-bold font-mono text-emerald-400">{svids.filter(s => s.status === "active").length} / {svids.length}</span>
            <span className="text-[10px] text-slate-500 font-mono block">SPIRE Workload Attestations</span>
          </div>
          <div className="bg-emerald-500/10 text-emerald-400 p-2.5 rounded-lg border border-emerald-500/20">
            <Network size={20} />
          </div>
        </div>

        <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-4 flex items-center justify-between shadow-xl">
          <div className="space-y-1">
            <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider block">Vault Secret Version</span>
            <span className="text-2xl font-bold font-mono text-indigo-400">v{secrets.reduce((acc, curr) => acc + curr.version, 0)}</span>
            <span className="text-[10px] text-slate-500 font-mono block">Active secret pathways</span>
          </div>
          <div className="bg-indigo-500/10 text-indigo-400 p-2.5 rounded-lg border border-indigo-500/20">
            <LockKeyhole size={20} />
          </div>
        </div>

        <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-4 flex items-center justify-between shadow-xl">
          <div className="space-y-1">
            <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider block">OPA Policy Enforcement</span>
            <span className="text-2xl font-bold font-mono text-cyan-400">Enforcing</span>
            <span className="text-[10px] text-slate-500 font-mono block">Rego policy engines active</span>
          </div>
          <div className="bg-cyan-500/10 text-cyan-400 p-2.5 rounded-lg border border-cyan-500/20">
            <Shield size={20} />
          </div>
        </div>

        <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-4 flex items-center justify-between shadow-xl">
          <div className="space-y-1">
            <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider block">Gateways Rate Limit</span>
            <span className="text-2xl font-bold font-mono text-amber-500">100 rps</span>
            <span className="text-[10px] text-slate-500 font-mono block">Symmetric shield active</span>
          </div>
          <div className="bg-amber-500/10 text-amber-400 p-2.5 rounded-lg border border-amber-500/20">
            <Activity size={20} />
          </div>
        </div>
      </div>

      {/* CORE WORKBENCH NAVIGATION */}
      <div className="flex border-b border-slate-800/80 overflow-x-auto whitespace-nowrap scrollbar-hide">
        {[
          { id: "spiffe", name: "SPIFFE mTLS Identity", icon: Network },
          { id: "vault", name: "Vault Key Management", icon: Lock },
          { id: "keycloak", name: "Keycloak OIDC & JWT", icon: UserCheck },
          { id: "opa", name: "OPA Policy Sandbox", icon: Shield },
          { id: "ratelimit", name: "Gateways & Shields", icon: Activity },
          { id: "audit", name: "Security Audit Logs", icon: FileText }
        ].map(subTab => {
          const Icon = subTab.icon;
          return (
            <button
              key={subTab.id}
              onClick={() => setActiveSubTab(subTab.id as any)}
              className={`px-5 py-3 text-xs font-medium flex items-center space-x-2 border-b-2 transition-all cursor-pointer ${
                activeSubTab === subTab.id
                  ? "border-indigo-500 text-indigo-400 bg-indigo-500/5 font-semibold"
                  : "border-transparent text-slate-400 hover:text-slate-200 hover:bg-slate-800/20"
              }`}
            >
              <Icon size={14} />
              <span>{subTab.name}</span>
            </button>
          );
        })}
      </div>

      {/* DYNAMIC TAB COMPONENT CONTROLLER */}
      <div className="min-h-[580px]">
        <AnimatePresence mode="wait">
          {activeSubTab === "spiffe" && (
            <motion.div
              key="spiffe"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              transition={{ duration: 0.15 }}
              className="grid grid-cols-1 lg:grid-cols-3 gap-6"
            >
              {/* SVID WORKLOAD LIST */}
              <div className="lg:col-span-2 space-y-6">
                <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-5 shadow-xl">
                  <div className="flex items-center justify-between mb-4">
                    <div className="space-y-0.5">
                      <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
                        <Network className="text-indigo-400" size={16} />
                        Active SPIFFE SVID Identifiers (mTLS)
                      </h3>
                      <p className="text-xs text-slate-400">Cryptographically verifiable workload identities running under mutual TLS</p>
                    </div>
                  </div>

                  <div className="divide-y divide-slate-800/60">
                    {svids.map(svid => (
                      <div
                        key={svid.spiffeId}
                        onClick={() => setSelectedSvid(svid)}
                        className={`py-4 flex flex-col md:flex-row md:items-center justify-between gap-4 cursor-pointer transition-all rounded-lg px-2 -mx-2 ${
                          selectedSvid?.spiffeId === svid.spiffeId ? "bg-slate-800/40" : "hover:bg-slate-800/20"
                        }`}
                      >
                        <div className="space-y-1 max-w-md">
                          <div className="flex items-center space-x-2">
                            <span className="text-xs font-semibold text-indigo-400 font-mono break-all">{svid.spiffeId}</span>
                            <span className="text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 px-1.5 py-0.5 rounded font-bold uppercase font-mono">
                              {svid.status}
                            </span>
                          </div>
                          <p className="text-xs text-slate-400">Assigned Workload Cluster: <span className="font-semibold text-slate-300">{svid.serviceName}</span></p>
                          <div className="flex items-center text-[10px] text-slate-500 space-x-4 font-mono">
                            <span>S/N: {svid.serialNumber}</span>
                            <span>Valid until: {new Date(svid.validTo).toLocaleDateString()}</span>
                          </div>
                        </div>

                        <div className="flex items-center space-x-3">
                          <button
                            disabled={isRotatingSvid !== null}
                            onClick={(e) => {
                              e.stopPropagation();
                              handleRotateSvid(svid.spiffeId);
                            }}
                            className="bg-slate-800 hover:bg-slate-700 disabled:opacity-40 text-slate-200 border border-slate-700/80 px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center space-x-1.5 transition-all cursor-pointer"
                          >
                            <RefreshCw size={13} className={isRotatingSvid === svid.spiffeId ? "animate-spin text-indigo-400" : "text-slate-400"} />
                            <span>{isRotatingSvid === svid.spiffeId ? "Renewing SVID..." : "Rotate Cert"}</span>
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                {/* HISTORICAL SVID EMISSIONS */}
                <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-5 shadow-xl">
                  <h3 className="text-xs font-bold text-slate-400 uppercase tracking-wider mb-4 flex items-center gap-2">
                    <History size={14} className="text-indigo-400" />
                    SPIRE Certificate Emission Log Ledger
                  </h3>
                  <div className="max-h-48 overflow-y-auto space-y-3 pr-2 scrollbar-thin">
                    {svidLogs.map((log, index) => (
                      <div key={index} className="bg-slate-950/50 rounded-lg p-3 border border-slate-800/40 text-xs font-mono space-y-1.5">
                        <div className="flex items-center justify-between text-slate-500">
                          <span>{new Date(log.timestamp).toLocaleTimeString()}</span>
                          <span className="text-indigo-400 text-[10px] font-bold">{log.operator}</span>
                        </div>
                        <div className="text-slate-300 flex items-center gap-2">
                          <span className="text-emerald-400 font-bold">●</span>
                          <span className="font-semibold text-indigo-300">{log.event}</span>
                        </div>
                        <div className="text-[10px] text-slate-400 break-all">{log.spiffeId}</div>
                        <p className="text-[11px] text-slate-500">{log.details}</p>
                      </div>
                    ))}
                  </div>
                </div>
              </div>

              {/* SVID DETAILS & CERTIFICATE BLOCK */}
              <div className="space-y-6">
                <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-5 shadow-xl">
                  <h3 className="text-sm font-semibold text-slate-100 mb-3 flex items-center gap-2">
                    <FileText className="text-indigo-400" size={16} />
                    Active SVID Cryptographic Details
                  </h3>

                  {selectedSvid ? (
                    <div className="space-y-4">
                      <div className="bg-slate-950/60 rounded-lg p-4 border border-slate-800/80 space-y-3 font-mono text-[11px]">
                        <div>
                          <span className="text-slate-500 block text-[10px]">TRUST DOMAIN</span>
                          <span className="text-slate-300 font-semibold">{selectedSvid.trustDomain}</span>
                        </div>
                        <div>
                          <span className="text-slate-500 block text-[10px]">SVID FINGERPRINT (SHA-256)</span>
                          <span className="text-indigo-400 break-all text-[10px]">{selectedSvid.fingerprint}</span>
                        </div>
                        <div className="grid grid-cols-2 gap-2">
                          <div>
                            <span className="text-slate-500 block text-[10px]">VALID FROM</span>
                            <span className="text-slate-400 text-[10px]">{new Date(selectedSvid.validFrom).toLocaleDateString()}</span>
                          </div>
                          <div>
                            <span className="text-slate-500 block text-[10px]">VALID UNTIL</span>
                            <span className="text-amber-400 text-[10px]">{new Date(selectedSvid.validTo).toLocaleDateString()}</span>
                          </div>
                        </div>
                      </div>

                      <div className="space-y-1.5">
                        <span className="text-[10px] font-bold text-slate-500 uppercase tracking-wider block">X.509 Certificate Chain PEM</span>
                        <div className="relative group">
                          <pre className="bg-slate-950 p-3 rounded-lg border border-slate-800/80 text-[9px] font-mono text-emerald-400/90 overflow-x-auto max-h-48 scrollbar-thin">
                            {selectedSvid.certificateChain[0]}
                          </pre>
                          <button
                            onClick={() => handleCopy(selectedSvid.certificateChain[0], `chain_${selectedSvid.serialNumber}`)}
                            className="absolute top-2 right-2 bg-slate-900/80 text-slate-300 hover:text-white p-1.5 rounded border border-slate-700/50 opacity-0 group-hover:opacity-100 transition-all cursor-pointer"
                          >
                            {copiedText === `chain_${selectedSvid.serialNumber}` ? <Check size={11} className="text-emerald-400" /> : <Copy size={11} />}
                          </button>
                        </div>
                      </div>
                    </div>
                  ) : (
                    <p className="text-xs text-slate-500">Select an SVID Workload to audit details</p>
                  )}
                </div>

                {/* SVID ROTATION REALTIME EXECUTION TRACER */}
                {svidTraceLogs.length > 0 && (
                  <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-5 shadow-xl">
                    <div className="flex items-center space-x-2 text-indigo-400 font-mono text-xs mb-3">
                      <Terminal size={14} className="animate-pulse" />
                      <span>SPIFFE Certificate Generation Trace</span>
                    </div>
                    <div className="bg-slate-950 p-4 rounded-lg border border-slate-900 text-[11px] font-mono text-slate-300 space-y-2 max-h-60 overflow-y-auto scrollbar-thin">
                      {svidTraceLogs.map((trace, i) => (
                        <div key={i} className="leading-relaxed">
                          <span className="text-slate-500 select-none mr-2">{`>`}</span>
                          {trace}
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </motion.div>
          )}

          {activeSubTab === "vault" && (
            <motion.div
              key="vault"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              transition={{ duration: 0.15 }}
              className="grid grid-cols-1 lg:grid-cols-3 gap-6"
            >
              {/* VAULT PATHS LIST */}
              <div className="lg:col-span-2 space-y-6">
                <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-5 shadow-xl">
                  <div className="flex items-center justify-between mb-4">
                    <div className="space-y-0.5">
                      <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
                        <Lock className="text-indigo-400" size={16} />
                        HashiCorp Vault KV Secrets Engines
                      </h3>
                      <p className="text-xs text-slate-400">High-security encrypted version-controlled paths with leasing rules</p>
                    </div>

                    <button
                      onClick={() => setIsAddingSecret(!isAddingSecret)}
                      className="bg-indigo-600 hover:bg-indigo-500 text-white border border-indigo-500/30 px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center space-x-1.5 transition-all cursor-pointer"
                    >
                      <Plus size={13} />
                      <span>{isAddingSecret ? "Cancel" : "Add Path"}</span>
                    </button>
                  </div>

                  {isAddingSecret && (
                    <div className="bg-slate-950/60 rounded-xl p-4 border border-indigo-500/20 mb-4 space-y-4">
                      <div className="space-y-1">
                        <label className="text-[10px] font-bold text-slate-400 uppercase font-mono block">Vault Path URI</label>
                        <input
                          type="text"
                          value={newSecretPath}
                          onChange={(e) => setNewSecretPath(e.target.value)}
                          className="w-full bg-slate-900 border border-slate-800 rounded px-3 py-1.5 text-xs font-mono focus:border-indigo-500 outline-none text-slate-100"
                        />
                      </div>

                      <div className="space-y-2">
                        <span className="text-[10px] font-bold text-slate-400 uppercase font-mono block">Secret Fields (Key-Value Keyring)</span>
                        {newSecretData.map((field, index) => (
                          <div key={index} className="flex items-center space-x-2">
                            <input
                              type="text"
                              placeholder="Key Name"
                              value={field.k}
                              onChange={(e) => {
                                const copy = [...newSecretData];
                                copy[index].k = e.target.value;
                                setNewSecretData(copy);
                              }}
                              className="w-1/3 bg-slate-900 border border-slate-800 rounded px-3 py-1 text-xs font-mono text-slate-200"
                            />
                            <input
                              type="text"
                              placeholder="Secret Value"
                              value={field.v}
                              onChange={(e) => {
                                const copy = [...newSecretData];
                                copy[index].v = e.target.value;
                                setNewSecretData(copy);
                              }}
                              className="w-2/3 bg-slate-900 border border-slate-800 rounded px-3 py-1 text-xs font-mono text-slate-200"
                            />
                            <button
                              onClick={() => {
                                if (newSecretData.length > 1) {
                                  setNewSecretData(newSecretData.filter((_, i) => i !== index));
                                }
                              }}
                              className="p-1.5 bg-slate-800 hover:bg-slate-700 rounded border border-slate-700/50 cursor-pointer"
                            >
                              <Trash size={12} className="text-rose-400" />
                            </button>
                          </div>
                        ))}
                        <button
                          onClick={() => setNewSecretData([...newSecretData, { k: "", v: "" }])}
                          className="text-[11px] text-indigo-400 font-semibold flex items-center space-x-1 hover:text-indigo-300 mt-1 cursor-pointer"
                        >
                          <Plus size={11} />
                          <span>Add Row</span>
                        </button>
                      </div>

                      <div className="flex justify-end space-x-2">
                        <button
                          onClick={() => handleAddSecret()}
                          className="bg-emerald-600 hover:bg-emerald-500 text-white px-3 py-1.5 rounded text-xs font-semibold cursor-pointer"
                        >
                          Save Secure Path
                        </button>
                      </div>
                    </div>
                  )}

                  <div className="divide-y divide-slate-800/60">
                    {secrets.map(secret => (
                      <div
                        key={secret.path}
                        onClick={() => setSelectedSecret(secret)}
                        className={`py-4 flex flex-col md:flex-row md:items-center justify-between gap-4 cursor-pointer transition-all rounded-lg px-2 -mx-2 ${
                          selectedSecret?.path === secret.path ? "bg-slate-800/40" : "hover:bg-slate-800/20"
                        }`}
                      >
                        <div className="space-y-1">
                          <div className="flex items-center space-x-2">
                            <span className="text-xs font-semibold text-indigo-300 font-mono">{secret.path}</span>
                            <span className="text-[10px] bg-indigo-500/10 text-indigo-400 border border-indigo-500/20 px-1.5 py-0.5 rounded font-bold font-mono">
                              v{secret.version}
                            </span>
                          </div>
                          <p className="text-xs text-slate-400">Lease window: <span className="font-semibold text-slate-300 font-mono">{secret.leaseDurationSeconds}s</span> (30 Days)</p>
                          <div className="flex items-center text-[10px] text-slate-500 space-x-2 font-mono">
                            <span>Keys active: {Object.keys(secret.data).join(", ")}</span>
                            <span>•</span>
                            <span>Updated: {new Date(secret.updatedAt).toLocaleTimeString()}</span>
                          </div>
                        </div>

                        {secret.isRotatable && (
                          <div className="flex items-center space-x-3">
                            <button
                              disabled={isRotatingSecret !== null}
                              onClick={(e) => {
                                e.stopPropagation();
                                const rotatedPayload = { ...secret.data };
                                Object.keys(rotatedPayload).forEach(k => {
                                  if (k === "password") {
                                    rotatedPayload[k] = `NXS_Secure_DB_Rotated_${Math.random().toString(36).substring(2, 8).toUpperCase()}!`;
                                  }
                                });
                                handleRotateSecret(secret.path, rotatedPayload);
                              }}
                              className="bg-slate-800 hover:bg-slate-700 disabled:opacity-40 text-slate-200 border border-slate-700/80 px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center space-x-1.5 transition-all cursor-pointer"
                            >
                              <RefreshCw size={13} className={isRotatingSecret === secret.path ? "animate-spin text-indigo-400" : "text-slate-400"} />
                              <span>{isRotatingSecret === secret.path ? "Rotating credentials..." : "Trigger Rotation"}</span>
                            </button>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                </div>

                {/* HISTORICAL VAULT ROTATION PATHS */}
                {selectedSecret && selectedSecret.rotationHistory.length > 0 && (
                  <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-5 shadow-xl">
                    <h3 className="text-xs font-bold text-slate-400 uppercase tracking-wider mb-4 flex items-center gap-2">
                      <History size={14} className="text-indigo-400" />
                      Vault Secret Pathway Version Ledger (v{selectedSecret.version} history)
                    </h3>
                    <div className="max-h-48 overflow-y-auto space-y-3 pr-2 scrollbar-thin">
                      {selectedSecret.rotationHistory.map((history, idx) => (
                        <div key={idx} className="bg-slate-950/50 rounded-lg p-3 border border-slate-800/40 text-xs font-mono space-y-1 flex justify-between items-start">
                          <div className="space-y-1">
                            <div className="flex items-center space-x-2">
                              <span className="text-emerald-400 font-bold">●</span>
                              <span className="font-semibold text-indigo-300">Secret Version v{history.version} Issued</span>
                            </div>
                            <p className="text-[10px] text-slate-500">Mutated Key Fields: {history.fields.join(", ")}</p>
                          </div>
                          <div className="text-right space-y-1">
                            <span className="text-indigo-400 text-[10px] font-bold block">{history.operator}</span>
                            <span className="text-slate-500 text-[10px]">{new Date(history.timestamp).toLocaleTimeString()}</span>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>

              {/* VAULT TRANSIT KEY CRYPTOGRAPHY SANDBOX */}
              <div className="space-y-6">
                {/* TRANSIT CRYPTO */}
                <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-5 shadow-xl space-y-4">
                  <div className="space-y-0.5">
                    <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
                      <Key className="text-indigo-400" size={16} />
                      Vault Transit Cryptography Sandbox
                    </h3>
                    <p className="text-xs text-slate-400">Cryptographically encrypt plaintext into secure ciphertext via AES-256 GCM transit rings</p>
                  </div>

                  <div className="bg-slate-950/60 rounded-lg p-4 border border-slate-800/80 space-y-4">
                    <div className="space-y-1.5">
                      <label className="text-[10px] font-bold text-slate-500 uppercase tracking-wider font-mono block">SELECT ACTIVE TRANSIT KEY</label>
                      <select
                        value={selectedTransitKey}
                        onChange={(e) => setSelectedTransitKey(e.target.value)}
                        className="w-full bg-slate-900 border border-slate-800 rounded px-2.5 py-1.5 text-xs font-semibold focus:border-indigo-500 outline-none text-slate-200"
                      >
                        {transitKeys.map(k => (
                          <option key={k.name} value={k.name}>{k.name} ({k.type})</option>
                        ))}
                      </select>
                    </div>

                    <div className="space-y-1.5">
                      <label className="text-[10px] font-bold text-slate-500 uppercase tracking-wider font-mono block">PLAINTEXT (e.g. Card Credentials)</label>
                      <input
                        type="text"
                        value={plainCryptoText}
                        onChange={(e) => setPlainCryptoText(e.target.value)}
                        className="w-full bg-slate-900 border border-slate-800 rounded px-3 py-1.5 text-xs font-mono focus:border-indigo-500 outline-none text-slate-200"
                      />
                    </div>

                    <div className="flex gap-2">
                      <button
                        onClick={handleEncryptPlaintext}
                        className="flex-1 bg-indigo-600 hover:bg-indigo-500 text-white font-semibold py-1.5 rounded text-xs transition-all cursor-pointer"
                      >
                        Encrypt
                      </button>
                      <button
                        disabled={!cipherCryptoText}
                        onClick={handleDecryptCiphertext}
                        className="flex-1 bg-slate-800 hover:bg-slate-700 disabled:opacity-40 text-slate-300 font-semibold py-1.5 rounded border border-slate-700/80 text-xs transition-all cursor-pointer"
                      >
                        Decrypt
                      </button>
                    </div>

                    {cipherCryptoText && (
                      <div className="space-y-1.5">
                        <span className="text-[10px] font-bold text-slate-500 uppercase tracking-wider font-mono block">VAULT TRANSIT CIPHERTEXT</span>
                        <div className="relative group">
                          <pre className="bg-slate-950 p-2.5 rounded border border-slate-900 text-[10px] font-mono text-cyan-400 break-all leading-normal whitespace-pre-wrap">
                            {cipherCryptoText}
                          </pre>
                          <button
                            onClick={() => handleCopy(cipherCryptoText, "cipherText")}
                            className="absolute top-1.5 right-1.5 bg-slate-900/80 text-slate-300 hover:text-white p-1 rounded border border-slate-700/50 opacity-0 group-hover:opacity-100 transition-all cursor-pointer"
                          >
                            {copiedText === "cipherText" ? <Check size={10} className="text-emerald-400" /> : <Copy size={10} />}
                          </button>
                        </div>
                      </div>
                    )}

                    {decryptedText && (
                      <div className="space-y-1.5">
                        <span className="text-[10px] font-bold text-slate-500 uppercase tracking-wider font-mono block">DECRYPTED PLAINTEXT RESPONSE</span>
                        <pre className="bg-slate-950 p-2.5 rounded border border-slate-900 text-[10px] font-mono text-emerald-400 break-all">
                          {decryptedText}
                        </pre>
                      </div>
                    )}
                  </div>
                </div>

                {/* SECRET ROTATION REALTIME EXECUTION TRACER */}
                {secretTraceLogs.length > 0 && (
                  <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-5 shadow-xl">
                    <div className="flex items-center space-x-2 text-indigo-400 font-mono text-xs mb-3">
                      <Terminal size={14} className="animate-pulse" />
                      <span>Vault Master Rotation Pipeline Trace</span>
                    </div>
                    <div className="bg-slate-950 p-4 rounded-lg border border-slate-900 text-[11px] font-mono text-slate-300 space-y-2 max-h-60 overflow-y-auto scrollbar-thin">
                      {secretTraceLogs.map((trace, i) => (
                        <div key={i} className="leading-relaxed">
                          <span className="text-slate-500 select-none mr-2">{`>`}</span>
                          {trace}
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </motion.div>
          )}

          {activeSubTab === "keycloak" && (
            <motion.div
              key="keycloak"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              transition={{ duration: 0.15 }}
              className="grid grid-cols-1 lg:grid-cols-3 gap-6"
            >
              {/* OIDC USER PROFILES LIST */}
              <div className="space-y-6">
                <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-5 shadow-xl">
                  <div className="space-y-0.5 mb-4">
                    <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
                      <UserCheck className="text-indigo-400" size={16} />
                      Keycloak Identities (OIDC Client Realm)
                    </h3>
                    <p className="text-xs text-slate-400">Authorized user database with Multi-Factor Authentication</p>
                  </div>

                  <div className="space-y-3">
                    {kcUsers.map(user => (
                      <div
                        key={user.id}
                        onClick={() => {
                          setSelectedKcUser(user);
                          handleIssueJwt(user.id);
                        }}
                        className={`p-4 border rounded-xl cursor-pointer transition-all ${
                          selectedKcUser?.id === user.id
                            ? "bg-indigo-950/20 border-indigo-500/50"
                            : "bg-slate-950/40 border-slate-800/60 hover:bg-slate-800/20"
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <div className="space-y-0.5">
                            <span className="text-xs font-semibold text-slate-200">{user.username}</span>
                            <span className="text-[10px] text-slate-400 block">{user.email}</span>
                          </div>
                          <span className="text-[9px] bg-emerald-500/15 text-emerald-400 border border-emerald-500/30 px-1.5 py-0.5 rounded font-bold font-mono">
                            MFA ENABLED
                          </span>
                        </div>

                        <div className="mt-3 flex flex-wrap gap-1">
                          {user.roles.map(r => (
                            <span key={r} className="text-[9px] bg-slate-800 text-slate-300 px-2 py-0.5 rounded-full border border-slate-700 font-mono">
                              role:{r}
                            </span>
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>

              {/* JWT ENCODER & COLORIZED DECODER */}
              <div className="lg:col-span-2 space-y-6">
                <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-5 shadow-xl">
                  <div className="flex items-center justify-between mb-4">
                    <div className="space-y-0.5">
                      <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
                        <FileText className="text-indigo-400" size={16} />
                        OIDC JWT Cryptographic Verification Sandbox
                      </h3>
                      <p className="text-xs text-slate-400 font-mono">RS256 JWT Signed by realm Keycloak-CA</p>
                    </div>

                    <button
                      disabled={!selectedKcUser || isIssuingJwt}
                      onClick={() => handleIssueJwt(selectedKcUser!.id)}
                      className="bg-indigo-600 hover:bg-indigo-500 disabled:opacity-40 text-white px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center space-x-1.5 transition-all cursor-pointer"
                    >
                      <RefreshCw size={13} className={isIssuingJwt ? "animate-spin" : ""} />
                      <span>Re-sign Token</span>
                    </button>
                  </div>

                  {activeJwt ? (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      {/* RAW CRYPTO STRING */}
                      <div className="space-y-3">
                        <span className="text-[10px] font-bold text-slate-500 uppercase tracking-wider font-mono block">RAW OIDC ACCESS TOKEN</span>
                        <div className="bg-slate-950 p-4 rounded-xl border border-slate-900 text-[10px] font-mono break-all h-96 overflow-y-auto scrollbar-thin flex flex-col justify-between leading-normal relative group">
                          <div>
                            {/* Color encode the raw JWT parts: Header.Payload.Signature */}
                            <span className="text-rose-400 select-all">{activeJwt.raw.split(".")[0]}</span>
                            <span className="text-indigo-300">.</span>
                            <span className="text-sky-300 select-all">{activeJwt.raw.split(".")[1]}</span>
                            <span className="text-indigo-300">.</span>
                            <span className="text-emerald-400 select-all">{activeJwt.raw.split(".")[2]}</span>
                          </div>
                          
                          <div className="border-t border-slate-900 mt-4 pt-3 flex justify-between items-center">
                            <span className="text-emerald-400 font-bold flex items-center gap-1">
                              <CheckCircle2 size={11} /> Cryptographic Signature OK
                            </span>
                            <button
                              onClick={() => handleCopy(activeJwt.raw, "jwt_raw")}
                              className="bg-slate-900/80 text-slate-300 hover:text-white p-1.5 rounded border border-slate-700/50 cursor-pointer"
                            >
                              {copiedText === "jwt_raw" ? <Check size={11} className="text-emerald-400" /> : <Copy size={11} />}
                            </button>
                          </div>
                        </div>
                      </div>

                      {/* DECODED CLAIMS SANDBOX */}
                      <div className="space-y-4">
                        <div className="space-y-2">
                          <span className="text-[10px] font-bold text-rose-400 uppercase tracking-wider font-mono block">JWT HEADER (ALGORITHM & KEY ID)</span>
                          <pre className="bg-slate-950 p-3 rounded-lg border border-slate-900 text-[10px] font-mono text-rose-300 overflow-x-auto">
                            {JSON.stringify(activeJwt.header, null, 2)}
                          </pre>
                        </div>

                        <div className="space-y-2">
                          <span className="text-[10px] font-bold text-sky-400 uppercase tracking-wider font-mono block">JWT PAYLOAD (CLAIMS & POLICIES)</span>
                          <pre className="bg-slate-950 p-4 rounded-lg border border-slate-900 text-[10px] font-mono text-sky-200 overflow-y-auto max-h-64 scrollbar-thin">
                            {JSON.stringify(activeJwt.payload, null, 2)}
                          </pre>
                        </div>
                      </div>
                    </div>
                  ) : (
                    <p className="text-xs text-slate-500">Select user to forge OIDC session token</p>
                  )}
                </div>
              </div>
            </motion.div>
          )}

          {activeSubTab === "opa" && (
            <motion.div
              key="opa"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              transition={{ duration: 0.15 }}
              className="grid grid-cols-1 lg:grid-cols-2 gap-6"
            >
              {/* OPA REGO EDITOR */}
              <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-5 shadow-xl space-y-4 flex flex-col justify-between">
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <div className="space-y-0.5">
                      <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
                        <Shield className="text-indigo-400" size={16} />
                        Active Open Policy Agent (OPA) Rules
                      </h3>
                      <p className="text-xs text-slate-400">Rego declarative policy validating cluster ingress permissions</p>
                    </div>

                    <button
                      onClick={() => {
                        if (isEditingRego) {
                          handleSaveOpaPolicy();
                        } else {
                          setIsEditingRego(true);
                        }
                      }}
                      className="bg-indigo-600 hover:bg-indigo-500 text-white px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center space-x-1.5 transition-all cursor-pointer"
                    >
                      <Settings size={13} />
                      <span>{isEditingRego ? "Save Policy" : "Edit Rego"}</span>
                    </button>
                  </div>

                  <div className="relative">
                    <textarea
                      disabled={!isEditingRego}
                      value={editedRegoCode}
                      onChange={(e) => setEditedRegoCode(e.target.value)}
                      className={`w-full h-96 bg-slate-950 font-mono text-xs text-indigo-300 p-4 rounded-xl border focus:border-indigo-500 outline-none leading-relaxed resize-none ${
                        isEditingRego ? "border-indigo-500/50" : "border-slate-900"
                      }`}
                    />
                  </div>
                </div>

                <div className="bg-slate-950/60 rounded-lg p-3 border border-slate-900 text-xs text-slate-400 space-y-1 leading-normal font-mono">
                  <span className="text-indigo-400 font-bold block">REGO DOCUMENT REFERENCE</span>
                  <span>Allows developers GET access on paths. Blocks developer writes on credentials pathway matching wildcards (`**/credentials**`).</span>
                </div>
              </div>

              {/* OPA PLAYGROUND TEST BED */}
              <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-5 shadow-xl space-y-6">
                <div className="space-y-0.5">
                  <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
                    <Play className="text-indigo-400" size={16} />
                    Dynamic OPA Evaluation Ingress Playground
                  </h3>
                  <p className="text-xs text-slate-400">Audit Rego evaluation results dynamically against custom request envelopes</p>
                </div>

                <div className="bg-slate-950/60 rounded-xl p-4 border border-slate-800/80 space-y-4 text-xs">
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-1.5">
                      <label className="text-[10px] font-bold text-slate-500 uppercase font-mono block">SUBJECT USER</label>
                      <select
                        value={opaSubjectUser}
                        onChange={(e) => {
                          setOpaSubjectUser(e.target.value);
                          const user = kcUsers.find(u => u.username === e.target.value);
                          if (user) setOpaSubjectRoles(user.roles);
                        }}
                        className="w-full bg-slate-900 border border-slate-800 rounded px-2.5 py-1.5 focus:border-indigo-500 outline-none text-slate-200"
                      >
                        {kcUsers.map(u => (
                          <option key={u.id} value={u.username}>{u.username}</option>
                        ))}
                      </select>
                    </div>

                    <div className="space-y-1.5">
                      <label className="text-[10px] font-bold text-slate-500 uppercase font-mono block">ASSIGNED ROLES</label>
                      <input
                        type="text"
                        disabled
                        value={opaSubjectRoles.join(", ")}
                        className="w-full bg-slate-950 border border-slate-800 rounded px-3 py-1.5 font-mono text-slate-400"
                      />
                    </div>
                  </div>

                  <div className="grid grid-cols-3 gap-4">
                    <div className="col-span-2 space-y-1.5">
                      <label className="text-[10px] font-bold text-slate-500 uppercase font-mono block">TARGET HTTP PATH</label>
                      <input
                        type="text"
                        value={opaResourcePath}
                        onChange={(e) => setOpaResourcePath(e.target.value)}
                        className="w-full bg-slate-900 border border-slate-800 rounded px-3 py-1.5 font-mono focus:border-indigo-500 outline-none text-slate-200"
                      />
                    </div>

                    <div className="space-y-1.5">
                      <label className="text-[10px] font-bold text-slate-500 uppercase font-mono block">METHOD</label>
                      <select
                        value={opaResourceMethod}
                        onChange={(e) => setOpaResourceMethod(e.target.value)}
                        className="w-full bg-slate-900 border border-slate-800 rounded px-2 py-1.5 focus:border-indigo-500 outline-none text-slate-200"
                      >
                        {["GET", "POST", "PUT", "DELETE"].map(m => (
                          <option key={m} value={m}>{m}</option>
                        ))}
                      </select>
                    </div>
                  </div>

                  <button
                    onClick={handleEvaluateOpa}
                    className="w-full bg-indigo-600 hover:bg-indigo-500 text-white font-semibold py-2 rounded-lg text-xs transition-all cursor-pointer flex items-center justify-center space-x-1.5"
                  >
                    <Play size={13} />
                    <span>Run OPA Rego Evaluation</span>
                  </button>

                  {/* EVALUATION OUTCOMES */}
                  {opaEvaluationResult && (
                    <div className="mt-4 pt-4 border-t border-slate-800/80 space-y-4">
                      <div className="flex items-center justify-between">
                        <span className="text-[11px] font-bold text-slate-400 font-mono">OPA EVALUATION DECISION</span>
                        {opaEvaluationResult.decision === "allow" ? (
                          <span className="bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 px-3 py-1 rounded font-extrabold uppercase font-mono flex items-center gap-1.5 text-xs">
                            <CheckCircle2 size={13} /> Access Granted
                          </span>
                        ) : (
                          <span className="bg-rose-500/10 text-rose-400 border border-rose-500/20 px-3 py-1 rounded font-extrabold uppercase font-mono flex items-center gap-1.5 text-xs">
                            <XCircle size={13} /> Access Denied
                          </span>
                        )}
                      </div>

                      <div className="bg-slate-950 p-4 rounded-lg border border-slate-900 space-y-2">
                        <span className="text-[10px] font-bold text-slate-500 uppercase tracking-wider block font-mono">Rule Trace Logic Reasons</span>
                        <ul className="list-disc pl-4 space-y-1.5 text-slate-300 text-[11px]">
                          {opaEvaluationResult.reasons.map((r: string, i: number) => (
                            <li key={i}>{r}</li>
                          ))}
                        </ul>
                      </div>

                      <div className="space-y-1.5">
                        <span className="text-[10px] font-bold text-slate-500 uppercase tracking-wider block font-mono">OPA Rego Evaluation Path Graph</span>
                        <pre className="bg-slate-950 p-3 rounded-lg border border-slate-900 text-[10px] font-mono text-slate-300 leading-relaxed overflow-x-auto max-h-40 overflow-y-auto scrollbar-thin">
                          {opaEvaluationResult.regoEngineTrace.map((line: string, index: number) => (
                            <div key={index}>{line}</div>
                          ))}
                        </pre>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </motion.div>
          )}

          {activeSubTab === "ratelimit" && (
            <motion.div
              key="ratelimit"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              transition={{ duration: 0.15 }}
              className="grid grid-cols-1 lg:grid-cols-3 gap-6"
            >
              {/* API KEYS MANAGER */}
              <div className="lg:col-span-2 space-y-6">
                <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-5 shadow-xl">
                  <div className="flex items-center justify-between mb-4">
                    <div className="space-y-0.5">
                      <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
                        <Key className="text-indigo-400" size={16} />
                        Client Ingress API Credentials Store
                      </h3>
                      <p className="text-xs text-slate-400">Verifiable client keychains mapping to cluster rate limiting policies</p>
                    </div>
                  </div>

                  {/* Forge key */}
                  <div className="bg-slate-950/60 rounded-xl p-4 border border-slate-800/80 mb-6 space-y-4">
                    <span className="text-xs font-semibold text-indigo-400 font-mono block">Issue Live Gateway Credentials</span>
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                      <div className="space-y-1.5">
                        <label className="text-[10px] font-bold text-slate-500 uppercase font-mono block">KEY NAME / PURPOSE</label>
                        <input
                          type="text"
                          placeholder="Datadog metrics integration"
                          value={newApiKeyName}
                          onChange={(e) => setNewApiKeyName(e.target.value)}
                          className="w-full bg-slate-900 border border-slate-800 rounded px-3 py-1.5 text-xs focus:border-indigo-500 outline-none text-slate-200"
                        />
                      </div>

                      <div className="space-y-1.5">
                        <label className="text-[10px] font-bold text-slate-500 uppercase font-mono block">OWNER WORKLOAD SERVICE</label>
                        <select
                          value={newApiKeyService}
                          onChange={(e) => setNewApiKeyService(e.target.value)}
                          className="w-full bg-slate-900 border border-slate-800 rounded px-2 py-1.5 text-xs focus:border-indigo-500 outline-none text-slate-200"
                        >
                          {["order-service", "payment-service", "data-analytics", "auth-gateway"].map(svc => (
                            <option key={svc} value={svc}>{svc}</option>
                          ))}
                        </select>
                      </div>

                      <div className="space-y-1.5">
                        <label className="text-[10px] font-bold text-slate-500 uppercase font-mono block">RATE LIMIT (MAX RPS)</label>
                        <input
                          type="number"
                          value={newApiKeyLimit}
                          onChange={(e) => setNewApiKeyLimit(Number(e.target.value))}
                          className="w-full bg-slate-900 border border-slate-800 rounded px-3 py-1.5 text-xs font-mono focus:border-indigo-500 outline-none text-slate-200"
                        />
                      </div>
                    </div>

                    <div className="flex justify-end pt-2">
                      <button
                        onClick={handleCreateApiKey}
                        disabled={!newApiKeyName}
                        className="bg-indigo-600 hover:bg-indigo-500 disabled:opacity-40 text-white font-semibold px-4 py-1.5 rounded-lg text-xs cursor-pointer transition-all"
                      >
                        Issue New Key
                      </button>
                    </div>

                    {createdPlainKey && (
                      <div className="bg-amber-500/10 border border-amber-500/20 rounded-lg p-3 space-y-1.5 text-xs">
                        <span className="font-bold text-amber-400 font-mono block">⚠️ IMPORTANT: COPY PLAIN-TEXT SECRET KEY ONCE</span>
                        <div className="relative group">
                          <pre className="bg-slate-950 p-2.5 rounded border border-slate-900 text-[11px] font-mono text-amber-200 break-all select-all">
                            {createdPlainKey}
                          </pre>
                          <button
                            onClick={() => handleCopy(createdPlainKey, "plainKey")}
                            className="absolute top-1.5 right-1.5 bg-slate-900/80 text-slate-300 hover:text-white p-1 rounded border border-slate-700/50 transition-all cursor-pointer"
                          >
                            {copiedText === "plainKey" ? <Check size={11} className="text-emerald-400" /> : <Copy size={11} />}
                          </button>
                        </div>
                        <p className="text-[10px] text-slate-400">This key is cryptographic SHA-256 hashed. We never display plaintext again.</p>
                      </div>
                    )}
                  </div>

                  {/* List active keys */}
                  <div className="divide-y divide-slate-800/60">
                    {apiKeys.map(key => (
                      <div key={key.id} className="py-3 flex flex-col md:flex-row md:items-center justify-between gap-4">
                        <div className="space-y-1">
                          <div className="flex items-center space-x-2">
                            <span className="text-xs font-semibold text-slate-200">{key.name}</span>
                            <span className={`text-[10px] border px-1.5 py-0.5 rounded font-bold font-mono ${
                              key.status === "active" ? "bg-emerald-500/15 text-emerald-400 border-emerald-500/20" : "bg-rose-500/15 text-rose-400 border-rose-500/20"
                            }`}>
                              {key.status}
                            </span>
                          </div>
                          <div className="flex items-center text-[10px] text-slate-500 font-mono space-x-4">
                            <span>Prefix: {key.prefix}...</span>
                            <span>Service: {key.ownerService}</span>
                            <span>Limit: {key.rateLimitRps} RPS</span>
                          </div>
                        </div>

                        {key.status === "active" && (
                          <button
                            onClick={() => handleRevokeApiKey(key.id)}
                            className="bg-slate-800 hover:bg-slate-700/60 text-slate-300 border border-slate-700/60 hover:text-rose-400 hover:border-rose-500/30 px-3 py-1.5 rounded text-xs font-semibold transition-all cursor-pointer"
                          >
                            Revoke Key
                          </button>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              </div>

              {/* LIVE SYMMETRIC RATE LIMIT VISUALIZER */}
              <div className="space-y-6">
                <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-5 shadow-xl space-y-4">
                  <div className="flex items-center justify-between">
                    <div className="space-y-0.5">
                      <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
                        <Activity className="text-indigo-400" size={16} />
                        Ingress Rate Limiter & Shield
                      </h3>
                      <p className="text-xs text-slate-400">Sliding Window traffic metrics & symmetric 429 blocks</p>
                    </div>

                    <button
                      disabled={isSimulatingRateLimit}
                      onClick={handleSimulateRateLimit}
                      className="bg-amber-600 hover:bg-amber-500 disabled:opacity-40 text-white font-semibold px-3 py-1.5 rounded-lg text-xs cursor-pointer transition-all flex items-center space-x-1"
                    >
                      <AlertTriangle size={12} />
                      <span>Trigger Spike</span>
                    </button>
                  </div>

                  {rateLimitMetrics.length > 0 && (
                    <div className="bg-slate-950 p-4 rounded-xl border border-slate-900 space-y-4">
                      {/* RPS Chart */}
                      <div className="h-40">
                        <ResponsiveContainer width="100%" height="100%">
                          <AreaChart data={[...rateLimitMetrics].reverse()}>
                            <defs>
                              <linearGradient id="colorRequests" x1="0" y1="0" x2="0" y2="1">
                                <stop offset="5%" stopColor="#818cf8" stopOpacity={0.4}/>
                                <stop offset="95%" stopColor="#818cf8" stopOpacity={0}/>
                              </linearGradient>
                              <linearGradient id="colorBlocked" x1="0" y1="0" x2="0" y2="1">
                                <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.4}/>
                                <stop offset="95%" stopColor="#f59e0b" stopOpacity={0}/>
                              </linearGradient>
                            </defs>
                            <CartesianGrid stroke="#1e293b" strokeDasharray="3 3" vertical={false} />
                            <XAxis dataKey="timestamp" stroke="#64748b" tickFormatter={(v) => new Date(v).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})} tick={{fontSize: 9}} />
                            <YAxis stroke="#64748b" tick={{fontSize: 9}} />
                            <Tooltip
                              contentStyle={{ backgroundColor: "#0f172a", border: "1px solid #334155" }}
                              labelStyle={{ color: "#94a3b8", fontSize: "10px" }}
                              itemStyle={{ fontSize: "11px" }}
                            />
                            <Area type="monotone" dataKey="totalRequests" name="Allowed reqs" stroke="#818cf8" fillOpacity={1} fill="url(#colorRequests)" strokeWidth={1.5} />
                            <Area type="monotone" dataKey="blockedRequests" name="Throttled (429)" stroke="#f59e0b" fillOpacity={1} fill="url(#colorBlocked)" strokeWidth={1.5} />
                          </AreaChart>
                        </ResponsiveContainer>
                      </div>

                      <div className="grid grid-cols-2 gap-2 text-center font-mono text-[11px] text-slate-400 pt-2 border-t border-slate-900">
                        <div className="bg-slate-900/60 p-2 rounded border border-slate-800">
                          <span className="text-slate-500 block text-[9px] uppercase">ALGORITHM</span>
                          <span className="font-semibold text-slate-300">SLIDING WINDOW</span>
                        </div>
                        <div className="bg-slate-900/60 p-2 rounded border border-slate-800">
                          <span className="text-slate-500 block text-[9px] uppercase">BURST CAPACITY</span>
                          <span className="font-semibold text-amber-400">150 REQS/SEC</span>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </motion.div>
          )}

          {activeSubTab === "audit" && (
            <motion.div
              key="audit"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              transition={{ duration: 0.15 }}
              className="space-y-4"
            >
              {/* AUDIT LOG FILTERS CONTROL BAR */}
              <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl p-4 shadow-xl flex flex-col md:flex-row gap-4 items-center justify-between text-xs">
                <div className="space-y-0.5">
                  <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
                    <FileText className="text-indigo-400" size={16} />
                    Immutable Decentralized Security Audit Ledger
                  </h3>
                  <p className="text-xs text-slate-400 font-mono">Consolidated telemetry audit trail mapping to Zero Trust execution components</p>
                </div>

                <div className="flex flex-wrap items-center gap-3">
                  <div className="flex items-center space-x-1.5">
                    <span className="text-slate-400 uppercase font-bold text-[10px] font-mono">Category:</span>
                    <select
                      value={auditFilterCategory}
                      onChange={(e) => setAuditFilterCategory(e.target.value)}
                      className="bg-slate-950 border border-slate-800 rounded px-2.5 py-1 focus:border-indigo-500 outline-none text-slate-200"
                    >
                      <option value="all">All Category</option>
                      <option value="authentication">Authentication</option>
                      <option value="authorization">Authorization</option>
                      <option value="secrets">Secrets Store</option>
                      <option value="network_mtls">Network & mTLS</option>
                      <option value="rate_limiting">Rate Limiting</option>
                    </select>
                  </div>

                  <div className="flex items-center space-x-1.5">
                    <span className="text-slate-400 uppercase font-bold text-[10px] font-mono">Severity:</span>
                    <select
                      value={auditFilterSeverity}
                      onChange={(e) => setAuditFilterSeverity(e.target.value)}
                      className="bg-slate-950 border border-slate-800 rounded px-2.5 py-1 focus:border-indigo-500 outline-none text-slate-200"
                    >
                      <option value="all">All Severity</option>
                      <option value="info">Info</option>
                      <option value="warning">Warning</option>
                      <option value="critical">Critical</option>
                    </select>
                  </div>
                </div>
              </div>

              {/* AUDIT TIMELINE TABLE */}
              <div className="bg-slate-900/60 backdrop-blur border border-slate-800/80 rounded-xl overflow-hidden shadow-xl">
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-xs border-collapse">
                    <thead>
                      <tr className="bg-slate-950/60 border-b border-slate-800 font-mono text-[10px] text-slate-400 uppercase tracking-wider">
                        <th className="p-4">Timestamp</th>
                        <th className="p-4">Category</th>
                        <th className="p-4">Action</th>
                        <th className="p-4">Actor</th>
                        <th className="p-4">Service</th>
                        <th className="p-4">Status</th>
                        <th className="p-4">IP Address</th>
                        <th className="p-4 max-w-sm">Telemetry Trace details</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-800/40">
                      {filteredAuditLogs.length > 0 ? (
                        filteredAuditLogs.map(log => (
                          <tr key={log.id} className="hover:bg-slate-800/10 font-mono transition-all text-slate-300">
                            <td className="p-4 whitespace-nowrap text-slate-500">
                              {new Date(log.timestamp).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" })}
                            </td>
                            <td className="p-4 whitespace-nowrap">
                              <span className="text-[10px] uppercase font-bold text-indigo-400">{log.category}</span>
                            </td>
                            <td className="p-4 whitespace-nowrap font-semibold text-slate-200">{log.action}</td>
                            <td className="p-4 whitespace-nowrap text-slate-400">{log.actor}</td>
                            <td className="p-4 whitespace-nowrap text-indigo-300">{log.service}</td>
                            <td className="p-4 whitespace-nowrap">
                              <span className={`text-[10px] font-bold px-1.5 py-0.5 rounded border uppercase ${
                                log.status === "success"
                                  ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/20"
                                  : "bg-rose-500/10 text-rose-400 border-rose-500/20"
                              }`}>
                                {log.status}
                              </span>
                            </td>
                            <td className="p-4 whitespace-nowrap text-slate-500">{log.clientIp}</td>
                            <td className="p-4 max-w-xs break-words text-[11px] text-slate-400 leading-normal">{log.details}</td>
                          </tr>
                        ))
                      ) : (
                        <tr>
                          <td colSpan={8} className="p-8 text-center text-slate-500 font-mono">
                            No security audit logs match the selected filter criteria.
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}
