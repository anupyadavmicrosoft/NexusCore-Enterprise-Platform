# NexusCore Identity Platform - Redis Architecture Specification
## Sprint 3 High-Performance Caching & Distributed State Blueprint

This document details the configuration, key-space design, eviction strategies, and operational models for the **NexusCore Redis Cluster**. Serving as the ultra-low latency state store for authentication sessions, token families, policy decisions, and active rate limiters, the Redis cluster is structured to scale to millions of active operations per second.

---

## 1. Topographical Topology & Deployment Pattern

To achieve sub-millisecond response times, high availability, and horizontal scalability, the Redis architecture employs a **Multi-Master Redis Cluster** with master-replica replication.

```
                  +----------------------------------------------+
                  |              SaaS Client Traffic             |
                  +----------------------+-----------------------+
                                         |
                                         ▼
                  +----------------------------------------------+
                  |                 API Gateway                  |
                  +----------------------+-----------------------+
                                         |
                                         ▼
                  +----------------------------------------------+
                  |               Go Microservices               |
                  +-------+--------------+---------------+-------+
                          |              |               |
             Read/Write   |              |               |
             (Dynamic)    ▼              ▼               ▼
                  +--------------+ +--------------+ +--------------+
                  |  Node 1      | |  Node 2      | |  Node 3      |
                  |  Master (0)  | |  Master (1)  | |  Master (2)  |
                  +------+-------+ +------+-------+ +------+-------+
                         |                |                |
             Replication |                |                |
             (Asynchronous)               ▼                ▼
                         ▼         +--------------+ +--------------+
                  +--------------+ |  Node 5      | |  Node 6      |
                  |  Node 4      | |  Replica     | |  Replica     |
                  |  Replica     | +--------------+ +--------------+
                  +--------------+
```

### 1.1 Deployment Details
- **Cluster Sizing:** 3 Master nodes (active sharding) + 3 Replica nodes (auto-failover quorum).
- **Cluster Connection Mode:** Dynamic topology scanning using smart-client routing protocols to directly interface with individual slot owners.
- **Max Memory Limit:** Configured with specific limits per pod matching the SRE memory footprint standard (`maxmemory 4gb`).

---

## 2. Eviction & Memory Isolation Strategies

Because Redis holds both transient data (Rate Limits) and critical session security records (Active Login Sessions), a single default eviction strategy is insufficient. 

To prevent catastrophic session losses, we define **Memory Isolation Zones**:

### 2.1 Zone A: Security State Cache (Persistent Sessions / Token Families)
- **Eviction Policy:** `noeviction`
- **Behavior:** If memory limits are breached, Redis returns write errors instead of evicting active authentication sessions or security markers.
- **Mitigation:** Vertical cluster autoscaling is triggered when memory utilization breaches 75%.

### 2.2 Zone B: Transient Access Cache (Rate Limits / Temporary OTPS)
- **Eviction Policy:** `volatile-lru`
- **Behavior:** Redis automatically drops the least recently used keys among those with active TTL values configured.

---

## 3. Key-Space Mappings, Formats, and TTL Matrix

Every cached element is restricted to strict prefix namespaces to avoid naming collisions and optimize lookup queries.

| Namespace Domain | Key Pattern Schema | Value Structure | Target TTL | Eviction Zone |
| --- | --- | --- | --- | --- |
| **Session Store** | `nc:session:{sess_uuid}` | JSON Object (Identity & Device Context) | `24 Hours` | Zone A (`noeviction`) |
| **Token Rotation family** | `nc:token_fam:{family_uuid}` | Hash Map (`token_hash` -> Status) | `30 Days` | Zone A (`noeviction`) |
| **Permissions Cache** | `nc:perm:{tenant_id}:{user_id}` | String Array (Permissions codes) | `15 Minutes` | Zone B (`volatile-lru`) |
| **Role Metadata Cache** | `nc:role:{tenant_id}:{role_id}` | JSON Object (Role Metadata) | `1 Hour` | Zone B (`volatile-lru`) |
| **Rate Limit Bucket** | `nc:rate:{ip_address}:{endpoint}` | Counter Integer | `1 Minute` | Zone B (`volatile-lru`) |
| **One-Time Passwords** | `nc:otp:{user_id}` | Hash (`otp_hash` -> Attempts count) | `5 Minutes` | Zone B (`volatile-lru`) |
| **Device Trust Context** | `nc:device:{user_id}:{fingerprint}` | JSON Object (Device Profiles) | `90 Days` | Zone A (`noeviction`) |
| **Distributed Lock** | `nc:lock:{resource_id}` | String (Unique Lock Owner ID) | `10 Seconds` | Zone B (`volatile-lru`) |

---

## 4. Architectural Patterns & Cryptographic Verification Flows

### 4.1 Token Replay Attack Verification Flow
To secure sliding refresh windows, Refresh Tokens are organized into **Token Families**.

```
Client App                    API Gateway                     auth-service                     Redis Cluster
    |                              |                               |                                 |
    |--- POST /auth/refresh ------>|                               |                                 |
    |    (Expired Token, Family)   |--- Inspect Context Payload -->|                                 |
    |                              |                               |--- Query Token Family --------->|
    |                              |                               |    HGET nc:token_fam:{family}   |
    |                              |                               |                                 |
    |                              |                               |<-- Returns Token State (USED) --|
    |                              |                               |                                 |
    |                              |                               |-- [COMPROMISE DETECTED] --------+
    |                              |                               |   If status is already "USED",  |
    |                              |                               |   instantly purge whole family  |
    |                              |                               |--- DEL nc:token_fam:{family} -->|
    |                              |                               |                                 |
    |                              |                               |--- Revoke All Active Sessions ->|
    |                              |<-- Return Revocation Error ---|                                 |
    |<-- Session Terminated -------|                                                                 |
```

### 4.2 Distributed Lock Implementation (Redlock Protocol)
To synchronize updates and prevent race conditions when updating multi-tenant configurations or user entities, we use the **Redlock Distributed Locking** pattern:
- **Lock Acquisition:** `SET nc:lock:{resource_id} {owner_uuid} NX PX 10000` (Acquires lock if not exists with 10s auto-expiry).
- **Lock Release:** Executed via Lua script to ensure safe deletion matching only the authorized owner:
```lua
if redis.call("get",KEYS[1]) == ARGV[1] then
    return redis.call("del",KEYS[1])
else
    return 0
end
```

---

### 🏆 Sprint 3 Redis Architecture Review Sign-Off
- **Engine Standard:** Redis v7.2+ compatible.
- **SLA Metrics:** Max connection delay < 1ms, query latency < 1.5ms (p99).
- **Security Posture:** Mandated TLS 1.3 encryption-in-transit, access managed via isolated ACL accounts.
