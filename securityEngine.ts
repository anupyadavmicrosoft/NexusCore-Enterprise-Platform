import crypto from "crypto";

// ==========================================
// ZERO TRUST CORE SECURITY TYPES & INTERFACES
// ==========================================

export interface SpiffeSvid {
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

export interface SvidRotationLog {
  timestamp: string;
  event: string;
  spiffeId: string;
  details: string;
  operator: string;
}

export interface VaultSecret {
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

export interface VaultKey {
  name: string;
  type: "aes256-gcm" | "rsa-2048" | "ecdsa-p256";
  status: "active" | "deprecated";
  createdAt: string;
  lastRotated: string;
}

export interface KeycloakUser {
  id: string;
  username: string;
  email: string;
  roles: string[];
  groups: string[];
  mfaEnabled: boolean;
  status: "active" | "locked";
}

export interface JwtTokenDetail {
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

export interface OpaPolicy {
  id: string;
  name: string;
  regoCode: string;
  isActive: boolean;
  updatedAt: string;
}

export interface OpaEvaluationRequest {
  subject: {
    user: string;
    roles: string[];
    groups: string[];
    spiffeId?: string;
  };
  resource: {
    path: string;
    method: string;
    service: string;
  };
}

export interface OpaEvaluationResult {
  decision: "allow" | "deny";
  reasons: string[];
  evaluatedPolicyId: string;
  timestamp: string;
  regoEngineTrace: string[];
}

export interface ApiKeyDetail {
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

export interface RateLimitMetric {
  timestamp: string;
  clientId: string;
  totalRequests: number;
  blockedRequests: number;
  rps: number;
}

export interface SecurityAuditEvent {
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

// ==========================================
// STATE STORE (IN-MEMORY SECURE PLATFORM)
// ==========================================

class ZeroTrustSecurityStore {
  public svids: SpiffeSvid[] = [];
  public rotationLogs: SvidRotationLog[] = [];
  public secrets: VaultSecret[] = [];
  public transitKeys: VaultKey[] = [];
  public keycloakUsers: KeycloakUser[] = [];
  public opaPolicies: OpaPolicy[] = [];
  public apiKeys: ApiKeyDetail[] = [];
  public auditLogs: SecurityAuditEvent[] = [];
  public rateLimitMetrics: RateLimitMetric[] = [];

  constructor() {
    this.preseedStore();
  }

  private preseedStore() {
    const now = new Date();
    
    // 1. Preseed SPIFFE SVIDs for cluster workloads
    const services = ["order-service", "payment-service", "auth-gateway", "data-analytics", "nexus-core"];
    services.forEach((svc, index) => {
      const start = new Date(now.getTime() - 86400000 * 5); // 5 days ago
      const end = new Date(start.getTime() + 86400000 * 30); // 30 days lease
      const spiffeId = `spiffe://nexuscore.io/ns/prod/sa/${svc}`;
      
      this.svids.push({
        spiffeId,
        trustDomain: "nexuscore.io",
        serviceName: svc,
        serialNumber: `NXS-${200000 + index}`,
        validFrom: start.toISOString(),
        validTo: end.toISOString(),
        fingerprint: crypto.createHash("sha256").update(spiffeId + start.toISOString()).digest("hex"),
        status: "active",
        certificateChain: [
          `-----BEGIN CERTIFICATE-----\nMIIB8TCCAZagAwIBAgIU${svc.toUpperCase() + index}...\n-----END CERTIFICATE-----`,
          "-----BEGIN CERTIFICATE-----\nMIIB+jCCAZqgAwIBAgIUNexusCoreIntermediateCA...\n-----END CERTIFICATE-----",
          "-----BEGIN CERTIFICATE-----\nMIIB/TCCAZ2gAwIBAgIUNexusCoreRootTrustAuthority...\n-----END CERTIFICATE-----"
        ]
      });

      this.rotationLogs.push({
        timestamp: new Date(start.getTime() + 1000 * 60 * 5).toISOString(), // 5 mins after start
        event: "X.509 SVID Issued",
        spiffeId,
        details: `SPIRE Node Agent completed cryptographic handshake. Signed by NexusCore CA.`,
        operator: "spire-agent"
      });
    });

    // 2. Preseed Vault secrets
    this.secrets.push({
      path: "/secret/data/database/credentials",
      version: 1,
      data: {
        username: "nexus_db_admin",
        password: "NXS_Secure_DB_Pass_2026_x!z",
        host: "postgresql-primary.prod.internal",
        port: "5432"
      },
      updatedAt: new Date(now.getTime() - 86400000 * 3).toISOString(),
      leaseDurationSeconds: 2592000, // 30 days
      isRotatable: true,
      rotationHistory: [
        {
          version: 1,
          timestamp: new Date(now.getTime() - 86400000 * 3).toISOString(),
          fields: ["username", "password", "host", "port"],
          operator: "vault-root-token"
        }
      ]
    });

    this.secrets.push({
      path: "/secret/data/integration/stripe",
      version: 2,
      data: {
        api_key: "sk_prod_51NzM1HkJ16Z...29v7b0",
        webhook_secret: "whsec_5d0a6c0...d9b23f"
      },
      updatedAt: new Date(now.getTime() - 86400000 * 1).toISOString(),
      leaseDurationSeconds: 7776000, // 90 days
      isRotatable: false,
      rotationHistory: [
        {
          version: 1,
          timestamp: new Date(now.getTime() - 86400000 * 15).toISOString(),
          fields: ["api_key"],
          operator: "admin"
        },
        {
          version: 2,
          timestamp: new Date(now.getTime() - 86400000 * 1).toISOString(),
          fields: ["webhook_secret"],
          operator: "admin"
        }
      ]
    });

    // 3. Preseed Transit Engines
    this.transitKeys.push({
      name: "payment-token-key",
      type: "aes256-gcm",
      status: "active",
      createdAt: new Date(now.getTime() - 86400000 * 10).toISOString(),
      lastRotated: new Date(now.getTime() - 86400000 * 10).toISOString()
    });

    // 4. Preseed Keycloak users
    this.keycloakUsers.push({
      id: "usr-92810",
      username: "alice_secops",
      email: "alice@nexuscore.io",
      roles: ["admin", "security-engineer"],
      groups: ["secops-team", "infrastructure"],
      mfaEnabled: true,
      status: "active"
    });

    this.keycloakUsers.push({
      id: "usr-38190",
      username: "bob_developer",
      email: "bob@nexuscore.io",
      roles: ["developer"],
      groups: ["checkout-team"],
      mfaEnabled: true,
      status: "active"
    });

    // 5. Preseed OPA Policies
    this.opaPolicies.push({
      id: "policy-edge-ingress",
      name: "Ingress HTTP Gateway Authorization",
      regoCode: `package nexuscore.authz

# Default behavior: Access Denied
default allow = false

# Rules for security engineers
allow {
    input.subject.roles[_] == "security-engineer"
    input.resource.method in ["GET", "POST", "PUT"]
}

# Rules for developers
allow {
    input.subject.roles[_] == "developer"
    input.resource.method == "GET"
    not is_restricted_path(input.resource.path)
}

# Restrict critical database credential pathways
is_restricted_path(path) {
    glob.match("**/credentials**", ["/"], path)
}
is_restricted_path(path) {
    glob.match("**/secrets**", ["/"], path)
}`,
      isActive: true,
      updatedAt: new Date(now.getTime() - 86400000 * 2).toISOString()
    });

    // 6. Preseed Client API Keys
    this.apiKeys.push({
      id: "key_7y2b7f012",
      name: "Stripe Webhook Listener API Key",
      prefix: "nxs_live_7y2b7f",
      hash: crypto.createHash("sha256").update("nxs_live_7y2b7f_secKey908123").digest("hex"),
      ownerService: "payment-service",
      allowedMethods: ["POST"],
      rateLimitRps: 15,
      status: "active",
      createdAt: new Date(now.getTime() - 86400000 * 4).toISOString()
    });

    this.apiKeys.push({
      id: "key_3m1l5k901",
      name: "Internal Datadog Exporter Key",
      prefix: "nxs_live_3m1l5k",
      hash: crypto.createHash("sha256").update("nxs_live_3m1l5k_metricEngine678").digest("hex"),
      ownerService: "data-analytics",
      allowedMethods: ["GET", "POST"],
      rateLimitRps: 50,
      status: "active",
      createdAt: new Date(now.getTime() - 86400000 * 12).toISOString()
    });

    // 7. Preseed Rate Limiting Metrics for Charts
    for (let i = 24; i >= 0; i--) {
      const ts = new Date(now.getTime() - i * 3600000).toISOString();
      const baseRequests = Math.floor(1000 + Math.random() * 3000);
      const isPeakHour = i % 8 === 0;
      const blocked = isPeakHour ? Math.floor(120 + Math.random() * 80) : Math.floor(Math.random() * 5);
      
      this.rateLimitMetrics.push({
        timestamp: ts,
        clientId: "auth-gateway",
        totalRequests: baseRequests,
        blockedRequests: blocked,
        rps: Math.round(baseRequests / 3600)
      });
    }

    // 8. Preseed Audit Logs
    const auditEvents = [
      { action: "SVID rotated", category: "network_mtls" as const, actor: "spire-agent", service: "payment-service", details: "mTLS X.509 SVID renewed. New thumbprint: " + crypto.randomBytes(16).toString("hex"), severity: "info" as const },
      { action: "Secret Read", category: "secrets" as const, actor: "order-service", service: "order-service", details: "Retrieved credentials from vault path /secret/data/database/credentials. Authorized via SPIFFE identity.", severity: "info" as const },
      { action: "OIDC User Authenticated", category: "authentication" as const, actor: "alice_secops", service: "auth-gateway", details: "Keycloak JWT session established. MFA code verified successfully.", severity: "info" as const },
      { action: "Rate Limit Exceeded", category: "rate_limiting" as const, actor: "unauthorized-ip-block", service: "auth-gateway", details: "Symmetric bucket limit tripped for client 198.51.100.42. Sliding window 429 triggered.", severity: "warning" as const },
      { action: "Access Denied", category: "authorization" as const, actor: "bob_developer", service: "auth-gateway", details: "OPA Policy evaluation failed for path /secret/data/database/credentials [Method: POST]. Authorization path: default deny.", severity: "warning" as const },
    ];

    for (let i = 0; i < 40; i++) {
      const template = auditEvents[Math.floor(Math.random() * auditEvents.length)];
      this.auditLogs.push({
        id: `aud_${crypto.randomBytes(8).toString("hex")}`,
        timestamp: new Date(now.getTime() - i * 4500000).toISOString(),
        category: template.category,
        action: template.action,
        actor: template.actor,
        service: template.service,
        status: (template.severity as string) === "critical" || (template.severity as string) === "warning" ? "blocked" : "success",
        details: template.details,
        severity: template.severity,
        clientIp: `10.244.0.${Math.floor(2 + Math.random() * 250)}`
      });
    }
  }

  // Record a new dynamic audit event
  public logAudit(
    category: "authentication" | "authorization" | "secrets" | "network_mtls" | "rate_limiting",
    action: string,
    actor: string,
    service: string,
    status: "success" | "failure" | "blocked",
    details: string,
    severity: "info" | "warning" | "critical" = "info"
  ): SecurityAuditEvent {
    const event: SecurityAuditEvent = {
      id: `aud_${crypto.randomBytes(8).toString("hex")}`,
      timestamp: new Date().toISOString(),
      category,
      action,
      actor,
      service,
      status,
      details,
      severity,
      clientIp: `10.244.0.${Math.floor(10 + Math.random() * 200)}`
    };

    this.auditLogs.unshift(event);
    if (this.auditLogs.length > 200) {
      this.auditLogs.pop();
    }
    return event;
  }
}

export const securityStore = new ZeroTrustSecurityStore();

// ==========================================
// CRYPTOGRAPHIC HELPER FUNCTIONS & OPERATIONS
// ==========================================

// Mock/simulate SPIFFE mTLS X.509 SVID rotation with precise cryptographic logging
export function rotateWorkloadSvid(spiffeId: string): { svid: SpiffeSvid; trace: string[] } {
  const trace: string[] = [];
  const now = new Date();
  
  trace.push(`[SVID ROTATION START] Initiated renewal for target workload ID: "${spiffeId}"`);
  
  const existingIndex = securityStore.svids.findIndex(s => s.spiffeId === spiffeId);
  if (existingIndex === -1) {
    throw new Error(`Workload with identity "${spiffeId}" not registered in the cluster registry.`);
  }
  
  const existing = securityStore.svids[existingIndex];
  
  trace.push(`[1/5] SPIRE agent generated a fresh 2048-bit RSA private key inside secure workspace boundary.`);
  trace.push(`[2/5] Prepared X.509 Certificate Signing Request (CSR) with SAN attribute "URI:${spiffeId}".`);
  trace.push(`[3/5] Dispatched secure GRPC request to central SPIRE server. Checking intermediate trust boundaries...`);
  
  // Create cryptographic credentials
  const serial = `NXS-${Math.floor(250000 + Math.random() * 50000)}`;
  const validFrom = now.toISOString();
  const validTo = new Date(now.getTime() + 86400000 * 30).toISOString(); // 30 days
  const fingerprint = crypto.createHash("sha256").update(spiffeId + validFrom).digest("hex");
  
  trace.push(`[4/5] Root Trust Authority signed CSR. Assigned Serial Number: "${serial}". Thumbprint: "${fingerprint}".`);
  trace.push(`[5/5] Replaced SVID certificate mapping inside local memory pool. Revoking stale thumbprint: "${existing.fingerprint.substring(0, 16)}...".`);
  
  const updatedSvid: SpiffeSvid = {
    ...existing,
    serialNumber: serial,
    validFrom,
    validTo,
    fingerprint,
    status: "active"
  };

  // Mark previous SVID as rotated
  securityStore.svids[existingIndex] = updatedSvid;
  
  // Record rotation events
  securityStore.rotationLogs.unshift({
    timestamp: now.toISOString(),
    event: "mTLS X.509 SVID Rotated",
    spiffeId,
    details: `Cryptographic keys re-generated. Signed successfully by NexusCore Root Trust Authority. Serial: ${serial}.`,
    operator: "spire-agent"
  });

  securityStore.logAudit(
    "network_mtls",
    "mTLS SVID Rotated",
    "spire-agent",
    existing.serviceName,
    "success",
    `Renewed X.509 SVID for "${spiffeId}". Assigned Serial: ${serial}, Fingerprint: ${fingerprint.substring(0, 16)}...`,
    "info"
  );

  return { svid: updatedSvid, trace };
}

// Vault Secret Manager write/rotation engine
export function rotateVaultSecret(path: string, secretData: Record<string, string>): { secret: VaultSecret; trace: string[] } {
  const trace: string[] = [];
  const now = new Date();
  
  trace.push(`[VAULT ROTATION START] Dispatched write directive for path: "${path}"`);
  
  let secretIndex = securityStore.secrets.findIndex(s => s.path === path);
  let existingSecret: VaultSecret;
  
  if (secretIndex === -1) {
    trace.push(`[1/4] Target path does not exist. Initializing path schema structure.`);
    existingSecret = {
      path,
      version: 0,
      data: {},
      updatedAt: now.toISOString(),
      leaseDurationSeconds: 2592000,
      isRotatable: true,
      rotationHistory: []
    };
    securityStore.secrets.push(existingSecret);
    secretIndex = securityStore.secrets.length - 1;
  } else {
    existingSecret = securityStore.secrets[secretIndex];
  }

  const nextVersion = existingSecret.version + 1;
  trace.push(`[2/4] Initializing write lock. Generating encrypted record envelope using Transit key engine.`);
  
  // Inject rotation details
  const updatedSecret: VaultSecret = {
    ...existingSecret,
    version: nextVersion,
    data: secretData,
    updatedAt: now.toISOString(),
    rotationHistory: [
      {
        version: nextVersion,
        timestamp: now.toISOString(),
        fields: Object.keys(secretData),
        operator: "vault-root-token"
      },
      ...existingSecret.rotationHistory
    ]
  };

  securityStore.secrets[secretIndex] = updatedSecret;
  
  trace.push(`[3/4] Successfully persisted version v${nextVersion} inside storage partition.`);
  trace.push(`[4/4] Released write lock. Evicted stale cache partitions in distributed pods. Lease assigned: ${updatedSecret.leaseDurationSeconds}s.`);

  securityStore.logAudit(
    "secrets",
    "Vault Secret Updated",
    "vault-root-token",
    "vault-server",
    "success",
    `Committed new secret version v${nextVersion} at path: "${path}". Fields: ${Object.keys(secretData).join(", ")}`,
    "info"
  );

  return { secret: updatedSecret, trace };
}

// Generate pre-signed Keycloak OIDC user JWTs with full claims
export function generateKeycloakJwt(userId: string): JwtTokenDetail {
  const user = securityStore.keycloakUsers.find(u => u.id === userId);
  if (!user) {
    throw new Error(`Keycloak identity provider could not locate client user ID: "${userId}".`);
  }

  const nowSeconds = Math.floor(Date.now() / 1000);
  const expirySeconds = nowSeconds + 3600; // 1 hour token
  const tokenUuid = crypto.randomBytes(16).toString("hex");

  const header = {
    alg: "RS256",
    typ: "JWT",
    kid: "nexuscore-signing-key-v1"
  };

  const payload = {
    iss: "https://keycloak.nexuscore.io/realms/nexus-prod",
    sub: user.id,
    aud: "https://gateway.nexuscore.io/api/v1",
    exp: expirySeconds,
    iat: nowSeconds,
    jti: tokenUuid,
    username: user.username,
    email: user.email,
    roles: user.roles,
    groups: user.groups,
    context: "OIDC_Keycloak_MFA_Signed"
  };

  // Base64Url encoder helper
  const b64 = (obj: any) => Buffer.from(JSON.stringify(obj)).toString("base64url");
  
  const unsigned = `${b64(header)}.${b64(payload)}`;
  // Cryptographic signature simulation (RS256 signature using SHA256 of header/payload signed with mock private key)
  const signature = crypto.createHash("sha256").update(unsigned + "nexuscore-secret-signature-key-2026").digest("base64url");
  const raw = `${unsigned}.${signature}`;

  securityStore.logAudit(
    "authentication",
    "Keycloak JWT Issued",
    user.username,
    "keycloak-auth-server",
    "success",
    `Established authentic user token session for [${user.username}]. Expiring in 60 minutes. Token UID: ${tokenUuid}`,
    "info"
  );

  return { header, payload, signature, raw };
}

// OPA (Open Policy Agent) Rego authorization simulator
export function evaluateOpaPolicy(policyId: string, request: OpaEvaluationRequest): OpaEvaluationResult {
  const policy = securityStore.opaPolicies.find(p => p.id === policyId);
  const nowStr = new Date().toISOString();
  
  if (!policy) {
    return {
      decision: "deny",
      reasons: [`OPA policy with ID "${policyId}" not active in registry.`],
      evaluatedPolicyId: policyId,
      timestamp: nowStr,
      regoEngineTrace: ["[OPA INIT] Policy not found. Bailing out with strict DENY."]
    };
  }

  const trace: string[] = [];
  const reasons: string[] = [];
  let allowed = false;

  trace.push(`[OPA ENGINE] Evaluated package: "package nexuscore.authz"`);
  trace.push(`[OPA ENGINE] Evaluating input payload action: "${request.resource.method} ${request.resource.path}"`);
  trace.push(`[OPA ENGINE] Evaluating user claims: "${request.subject.user}" with roles [${request.subject.roles.join(", ")}]`);

  // Simple Rego rule simulator parser
  const isSecurityEngineer = request.subject.roles.includes("security-engineer");
  const isDeveloper = request.subject.roles.includes("developer");
  const isCredentialsPath = request.resource.path.includes("credentials") || request.resource.path.includes("secrets");

  if (isSecurityEngineer) {
    allowed = true;
    trace.push(`[OPA EVAL] MATCHED RULE: "allow { input.subject.roles[_] == 'security-engineer' ... }"`);
    reasons.push("Subject possesses privileged 'security-engineer' role authorized for all database gateways.");
  } else if (isDeveloper) {
    if (request.resource.method === "GET" && !isCredentialsPath) {
      allowed = true;
      trace.push(`[OPA EVAL] MATCHED RULE: "allow { input.subject.roles[_] == 'developer' && method == 'GET' }"`);
      reasons.push("Subject possesses developer role. Permitted read-only (GET) access on open platform routes.");
    } else {
      allowed = false;
      trace.push(`[OPA EVAL] TRIGGERED RESTRICTION: "is_restricted_path(path) matched on target resource"`);
      if (isCredentialsPath) {
        reasons.push("Developer identities are strictly barred from querying system credentials and secret storage pathways.");
      } else {
        reasons.push("Developers are barred from performing mutating tasks (POST/PUT/DELETE) on non-assigned microservice gateways.");
      }
    }
  } else {
    allowed = false;
    trace.push(`[OPA EVAL] FALLBACK: No rules matched. Triggered package default directive "default allow = false".`);
    reasons.push("Default Ingress policy: denies all traffic which has no explicitly defined allowance rule.");
  }

  const decision: "allow" | "deny" = allowed ? "allow" : "deny";

  securityStore.logAudit(
    "authorization",
    "OPA Authorization Evaluated",
    request.subject.spiffeId || request.subject.user,
    request.resource.service,
    allowed ? "success" : "blocked",
    `OPA Access ${decision.toUpperCase()} for path: "${request.resource.path}". Policy applied: "${policy.name}"`,
    allowed ? "info" : "warning"
  );

  return {
    decision,
    reasons,
    evaluatedPolicyId: policyId,
    timestamp: nowStr,
    regoEngineTrace: trace
  };
}

// Dynamic Client Rate Limiting Simulator
export function triggerSimulatedRateLimitHit(clientId: string): RateLimitMetric {
  const now = new Date();
  
  // Add metric to log array
  const lastIndex = securityStore.rateLimitMetrics.findIndex(m => m.clientId === clientId);
  let updatedMetric: RateLimitMetric;

  if (lastIndex !== -1) {
    const existing = securityStore.rateLimitMetrics[lastIndex];
    updatedMetric = {
      timestamp: now.toISOString(),
      clientId,
      totalRequests: existing.totalRequests + 1,
      blockedRequests: existing.blockedRequests + (Math.random() > 0.4 ? 1 : 0),
      rps: existing.rps + Math.floor(Math.random() * 2)
    };
    securityStore.rateLimitMetrics[lastIndex] = updatedMetric;
  } else {
    updatedMetric = {
      timestamp: now.toISOString(),
      clientId,
      totalRequests: 1,
      blockedRequests: 1,
      rps: 1
    };
    securityStore.rateLimitMetrics.unshift(updatedMetric);
  }

  securityStore.logAudit(
    "rate_limiting",
    "Rate Limit Tripped",
    clientId,
    "auth-gateway",
    "blocked",
    `Throttled traffic flow from client identification key [${clientId}]. Sliding window counter reached threshold.`,
    "warning"
  );

  return updatedMetric;
}
