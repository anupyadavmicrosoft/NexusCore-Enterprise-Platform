# Production Deployment Architecture (NexusCore)

This document specifies GKE deployment architecture, ArgoCD GitOps pipelines, Helm configurations, and rolling upgrade protocols.

## 1. Multi-Region GKE Target Topology

The platform targets multi-region active-active clusters managed with Google Anthos Multi-Cluster Ingress.

```
                    [ Cloud DNS / Traffic Manager ]
                                  |
            +---------------------+---------------------+
            | (Geo-Routing)                             | (Geo-Routing)
            v                                           v
     [ GKE Cluster - us-central1 ]               [ GKE Cluster - us-east1 ]
     +---------------------------+               +------------------------+
     | Ingress (Traefik)         |               | Ingress (Traefik)      |
     |                           |               |                        |
     | api-gateway (3 Replicas)  |               | api-gateway (3 Replicas|
     |                           |               |                        |
     | auth-service (3 Replicas) |               | auth-service (3 Replica|
     |                           |               |                        |
     | compute-engine (3 Replic) |               | compute-engine (3 Repl)|
     +---------------------------+               +------------------------+
```

---

## 2. Kubernetes Deployment Manifest Example

The following is a production-grade Kubernetes deployment manifest (`gateway-deployment.yaml`) highlighting resource limits, liveness probes, rolling update strategies, and Pod anti-affinity rules.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-gateway
  namespace: nexus-core
  labels:
    app: api-gateway
    tier: edge
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: api-gateway
  template:
    metadata:
      labels:
        app: api-gateway
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - api-gateway
              topologyKey: kubernetes.io/hostname
      containers:
      - name: gateway
        image: gcr.io/nexuscore-prod/api-gateway:v1.3.0
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8080
          name: http
        resources:
          limits:
            cpu: "1"
            memory: "512Mi"
          requests:
            cpu: "200m"
            memory: "128Mi"
        securityContext:
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 10001
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 15
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

---

## 3. ArgoCD GitOps Continuous Delivery

NexusCore uses **ArgoCD** to achieve declarative continuous delivery. The source of truth for all Kubernetes resources is the git repository structure under `/deployments/kubernetes/`.

### 3.1 ArgoCD Application Spec
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: nexuscore-production
  namespace: argocd
spec:
  project: default
  source:
    repoURL: 'https://github.com/enterprise/nexus-core.git'
    targetRevision: HEAD
    path: deployments/kubernetes/helm/nexuscore
    helm:
      valueFiles:
        - values-prod.yaml
  destination:
    server: 'https://kubernetes.default.svc'
    namespace: nexus-core
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

---

## 4. Helm Values Production Baseline (`values-prod.yaml`)

```yaml
global:
  environment: production
  domain: nexuscore.enterprise.com

apiGateway:
  replicaCount: 3
  image:
    repository: gcr.io/nexuscore-prod/api-gateway
    tag: v1.3.0
  resources:
    limits:
      cpu: 1000m
      memory: 512Mi
    requests:
      cpu: 200m
      memory: 128Mi

authService:
  replicaCount: 3
  database:
    host: pg-primary.nexus-core.svc.cluster.local
    name: nexus_auth_prod

computeEngine:
  replicaCount: 3
  kafka:
    brokers:
      - kafka-0.kafka-headless.nexus-core.svc.cluster.local:9092
```
