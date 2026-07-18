# Disaster Recovery (DR) Protocol & Business Continuity (NexusCore)

This document establishes the Disaster Recovery guidelines, RTO/RPO metrics, regional failover steps, and database backup routines.

## 1. Key Metrics & Service Level Agreements (SLAs)

NexusCore categorizes system disruptions into three severity tiers, mapping to explicit Recovery Time Objectives (RTO) and Recovery Point Objectives (RPO):

| System Tier | Severity / Impact | Target RTO | Target RPO | Backup Recovery Method |
| :--- | :--- | :--- | :--- | :--- |
| **Tier 1 (Identity & Auth)** | Authentication / Login blocked. | **< 5 Minutes** | **< 30 Seconds** | Read-Replica Multi-Region promotion |
| **Tier 2 (Transactions Core)** | Compute Engine transactions blocked. | **< 15 Minutes** | **< 10 Seconds** | Kafka multi-region broker mirroring |
| **Tier 3 (Analytics / Reporting)** | Non-critical reporting lagging. | **< 4 Hours** | **< 1 Hour** | Nightly Point-In-Time backups restoration |

---

## 2. Multi-Region Replication Strategy

To maintain SLAs under a complete regional outage, data replication is configured across Google Cloud's `us-central1` and `us-east1` regions.

```
+-----------------------------------+         +-----------------------------------+
|     Region A (us-central1)        |         |        Region B (us-east1)        |
|                                   |         |                                   |
|   PostgreSQL Primary Database     |=======> |   PostgreSQL Sync Read-Replica    |
|                                   | (Sync)  |                                   |
|   Kafka Broker (Primary Cluster)  |=======> |   Kafka MirrorMaker Cluster       |
|                                   | (Async) |                                   |
+-----------------------------------+         +-----------------------------------+
```

### 2.1 Database Replication (PostgreSQL Cloud SQL)
Primary PostgreSQL acts as the write target in Region A. Synchronous replication is active to a replica instance in Region B, ensuring that transactions completed in Region A are atomically mirrored prior to execution response return.

### 2.2 Kafka Event Mesh Mirroring
Kafka MirrorMaker 2 replication processes real-time event logs asynchronously between the cluster brokers, maintaining identical transactional histories on both endpoints.

---

## 3. Automated Failover Orchestration Runbook

In the event of a catastrophic Region A failure:

### Step 1: Detect Outage & Trigger SRE Incident Panel
The Global Traffic Manager / Cloud DNS records more than 3 missed cluster heartbeat events on Region A. SREs are paged via AlertManager on critical status.

### Step 2: Reroute External Traffic via Global DNS
Reroute Edge DNS configurations immediately to forward 100% of ingress queries directly to Traefik endpoints in Region B:
```bash
gcloud dns record-sets transaction-changes update nexuscore.enterprise.com \
  --type=A --ttl=30 --rrdatas="[IP_REGION_B_LOAD_BALANCER]" \
  --zone="enterprise-dns-zone"
```

### Step 3: Promote PostgreSQL Read-Replica to Primary
Execute replication termination and promote Region B read-replica to accept transactional write workflows:
```bash
gcloud sql instances promote pg-replica-region-b --project=nexuscore-prod
```

### Step 4: Scale Region B Replicas & Assess Integrity
Perform Helm override commands to scale deployment node configurations on Region B cluster, preparing it for double capacity handling:
```bash
kubectl scale deployment api-gateway compute-engine auth-service -n nexus-core --replicas=6
```

---

## 4. Backup & Point-In-Time-Recovery (PITR) Schedule

*   **Primary DB Backup**: Auto-scheduled snapshots every 24 hours at 02:00 UTC with 30-day retention policies.
*   **Point-In-Time-Recovery (PITR)**: Write-Ahead Logs (WAL) are mirrored to Cloud Storage (GCS) cold classes every 5 minutes. This enables system database rollback down to the exact millisecond boundary in the event of logical corruption.
*   **Verification Routine**: Automated backup restoration trials execute inside an isolated sandbox cluster every Tuesday morning, validating snapshot integrity.
