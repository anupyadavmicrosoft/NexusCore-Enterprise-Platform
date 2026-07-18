# SRE & On-Call Runbook (NexusCore)

This document contains step-by-step procedures to resolve common production incidents, system failures, and operational alerts.

## 1. Quick Emergency Command Sheet

```
+-----------------------------+-------------------------------------------------------------+
| Incident Type               | Remediation Action / Command                                |
+-----------------------------+-------------------------------------------------------------+
| CrashLoopBackOff Pods       | kubectl rollout restart deployment <name> -n nexus-core     |
| DB Connection Pool Spike    | kubectl scale deployment auth-service --replicas=6          |
| Extreme Traffic Spike       | kubectl scale deployment api-gateway --replicas=8           |
| Secret Rotation / Leak      | kubectl delete secret postgres-db-secret -n nexus-core      |
+-----------------------------+-------------------------------------------------------------+
```

---

## 2. Incident Scenarios & Remediation Playbooks

### Scenario A: `APIHighLatency` Alert Triggered
*   **Alert Criteria**: Ingress p95 response times exceeded 200ms baseline.
*   **Step 1: Identify Victim Service**: Inspect Prometheus queries to pinpoint which downstream microservice is lagging:
    ```promql
    sum(rate(http_request_duration_seconds_sum[5m])) by (service) / sum(rate(http_request_duration_seconds_count[5m])) by (service)
    ```
*   **Step 2: Stream Live Container Trace Logs**: Check if the lagging service has run out of database connection sockets or is throwing memory timeouts:
    ```bash
    kubectl logs -l app=compute-engine -n nexus-core --tail=100 -f
    ```
*   **Step 3: Mitigate with Horizontal Scaling**: Scale up lagging microservice node replicas immediately to spread transaction load constraints:
    ```bash
    kubectl scale deployment compute-engine -n nexus-core --replicas=6
    ```

---

### Scenario B: Database Socket Pool Exhaustion
*   **Alert Criteria**: PostgreSQL active clients count hits 95% of server maximum pool allowance.
*   **Step 1: Check Current PG Active Sockets**: Log into PostgreSQL read-replica to query active client queries:
    ```sql
    SELECT pid, query, state, age(clock_timestamp(), query_start) 
    FROM pg_stat_activity 
    WHERE state != 'idle' 
    ORDER BY age DESC;
    ```
*   **Step 2: Terminate Long-Running / Rogue Queries**: Kill queries that have been active for more than 60 seconds blocking transaction locks:
    ```sql
    SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE age(clock_timestamp(), query_start) > interval '60 seconds';
    ```
*   **Step 3: Temporarily Relieve Load with Replica Read Offloading**: Alter application configs to steer 100% of read-only queries away from primary databases over to read-replicas.

---

### Scenario C: Uncontrolled Pod CrashLoopBackOff Errors
*   **Alert Criteria**: Deployments fail to complete rolling updates, or services crash instantly upon startup.
*   **Step 1: Inspect Pod Crash logs**: Query the previous termination logs to catch critical panics or unhandled exceptions:
    ```bash
    kubectl logs -n nexus-core -l app=auth-service --previous --tail=50
    ```
*   **Step 2: Check Config / Environment Alignment**: Verify if Secret Managers or ConfigMaps failed to mount correctly into the container volume space:
    ```bash
    kubectl describe pod -l app=auth-service -n nexus-core
    ```
*   **Step 3: Rollback to Safe Baseline Build**: If the crash was triggered by a buggy deploy, trigger a GitOps deployment rollback to the previous stable release tag:
    ```bash
    argocd app rollback nexuscore-production <previous-revision-number>
    ```

---

## 3. High-Traffic Workload Prep Scaling

If the enterprise expects a planned high-traffic event (e.g., Black Friday operations), execute the proactive scaling procedures below:

```bash
# Proactively scale Ingress Gateway cluster nodes
kubectl scale deployment api-gateway -n nexus-core --replicas=10

# Scale Transaction processing nodes
kubectl scale deployment compute-engine -n nexus-core --replicas=10

# Scale Identity checking nodes
kubectl scale deployment auth-service -n nexus-core --replicas=8
```

---

## 4. TLS Certificate Renewal Procedure

If Traefik edge certificates are expiring or need manual rotation:

1. Request fresh SSL certificates from Cloud Certificate Manager or Let's Encrypt.
2. Update the Kubernetes TLS Secret:
    ```bash
    kubectl create secret tls nexuscore-tls-secret \
      --cert=path/to/fullchain.pem \
      --key=path/to/privkey.pem \
      -n nexus-core --dry-run=client -o yaml | kubectl apply -f -
    ```
3. Trigger Ingress configuration reloading sequence:
    ```bash
    kubectl rollout restart deployment ingress-controller-traefik -n kube-system
    ```
