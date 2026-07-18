# NexusCore Production Kubernetes Platform Specification

This document details the architectural specifications, security boundaries, and infrastructure layouts of the **NexusCore Production Kubernetes Platform**. It coordinates the secure orchestration of our highly available (HA), zero-trust microservice and stateful cluster systems.

---

## 1. Multi-Namespace Segregation Blueprint

To enforce security boundaries, simplify access control (RBAC), and isolate workloads, the cluster is divided into four distinct namespaces:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       nexuscore-ingress (Edge Boundary)                     │
│  - Hosts the ingress controllers (Nginx/Envoy) and SSL/TLS terminators.     │
│  - Receives direct public internet traffic from external load balancers.    │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                      nexuscore-prod (Microservices Domain)                  │
│  - Hosts stateless application containers: API Gateway, Auth, Compute.      │
│  - Zero public egress; routing is entirely controlled via mTLS policies.     │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        nexuscore-data (Stateful Domain)                    │
│  - Hosts databases, caches, and message brokers (Postgres, Redis, Kafka).   │
│  - Enforces persistent storage mounts and restricted socket connectivity.   │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Ingress and Traffic Management

At the cluster boundary, traffic routing is governed by an **NGINX Ingress Controller** integrated with **cert-manager** for automated, zero-downtime Let's Encrypt TLS-certificate provisioning.

### Ingress Specifications:
*   **Protocol Hardening**: Terminate all SSL/TLS requests at the Ingress boundary utilizing TLS 1.3 with a hardened modern cipher suite profile.
*   **HTTP Strict Transport Security (HSTS)**: Inject headers instructing browsers to access NexusCore exclusively via HTTPS with preloaded rules.
*   **Edge Rate Limiting**: Limit anonymous traffic at the gateway to prevent buffer overflows, slowloris attempts, and credential brute-forcing.

---

## 3. Stateful Clustering (PostgreSQL, Redis, Kafka)

Databases, brokers, and distributed coordinate structures are managed via Kubernetes **StatefulSets** to guarantee deterministic network identifiers, ordered startup, and dedicated storage mappings.

### Stateful Architecture:
1.  **PostgreSQL Replication**: Deployed with one Read-Write primary and two hot-standby read replicas. Failovers are managed automatically via Patroni or similar orchestrators.
2.  **Redis Cache Sharding**: Organized as a Redis Cluster with 3 master nodes and 3 replica nodes, distributing partition hashes with complete fault tolerance.
3.  **Apache Kafka Event Broker**: A 3-node broker cluster utilizing ZooKeeper (or KRaft) to sustain topic partition offsets with zero data loss.

---

## 4. Workload Resource Profiles & Scheduling Policies

Workloads are scheduled utilizing tailored **Priority Classes** to guarantee that critical cluster components (such as database engines or ingress controllers) are never evicted during node memory starvation or high workload utilization.

### Class Definitions:
*   `nexuscore-critical`: Reserved for Ingress controllers and core stateful database sets (PreemptionPolicy: `PreemptLowerPriority`).
*   `nexuscore-high`: Reserved for front-line API Gateways and microservice handlers.
*   `nexuscore-normal`: Default scheduling class for testing, asynchronous batch compute workers, and reporting tools.

---

## 5. Horizontal Pod Autoscaling (HPA)

Workloads in `nexuscore-prod` scale dynamically based on resources. **Horizontal Pod Autoscalers (HPA v2)** are bound to the API Gateway, Auth, and Compute deployments.

*   **Scaling Metrics**: Configured to scale up if average CPU utilization exceeds **70%** or if average Memory utilization exceeds **80%**.
*   **Scale Down Stabilization**: Custom stabilization windows are configured with a 5-minute cool-down delay to prevent "thrashing" (rapidly scaling up and down during sporadic spikes).

---

## 6. Zero-Trust Network Policies (Segmentation)

By default, a **Deny-All NetworkPolicy** is applied to the namespaces. Sockets can only communicate if explicitly permitted by ingress/egress rules, restricting lateral migration in the event of an application exploit.

```
       [ Public Internet Traffic ]
                   │
                   ▼
       [ Ingress Nginx Controller ]
                   │
                   ▼ (only to port 8080)
         [ API Gateway Pods ]
             │           │
             │           └───────────┐ (only to port 8082)
             ▼ (only to port 8081)    ▼
      [ Auth Service ]       [ Compute Engine ]
             │                       │
      (only to port 5432/6379)       ▼ (only to port 5432/29092)
             └───────────┬───────────┘
                         ▼
        { Stateful Databases, Caches, Kafka }
```

---

## 7. Pod Disruption Budgets (PDB)

To ensure that rolling platform updates or voluntary node drains do not violate our high availability (HA) SLAs, we define strict **Pod Disruption Budgets**.
*   Microservices and Ingresses require `minAvailable: 2` or `maxUnavailable: 1` to guarantee cluster availability during SRE node maintenance.
*   Stateful sets (Kafka, Redis, PostgreSQL) enforce a minimum quorum layout of at least 2 active nodes at all times.
