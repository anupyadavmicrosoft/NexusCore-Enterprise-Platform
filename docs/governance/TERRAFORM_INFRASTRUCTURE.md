# NexusCore Terraform & IaC Governance Specification

## 1. Executive Mandate & Multi-Region Resiliency

This document establishes the official **NexusCore Infrastructure as Code (IaC) Standards**. 

All cloud-native virtual environments, networking layers, database instances, and container orchestration engines must be declared, provisioned, and audited exclusively via **Terraform**. Manual modifications via the console ("ClickOps") are strictly prohibited to prevent configuration drift and guarantee reproducible multi-region configurations.

---

## 2. Reusable Modular Directory Layout

The Terraform layout is structured into highly cohesive, loosely coupled **reusable modules** paired with **environment environments** to isolate production state from testing and staging branches.

```
/enterprise-platform/terraform/
├── environments/
│   └── production/
│       ├── main.tf             # Core composition of modular resources
│       ├── variables.tf        # Environment-specific configuration switches
│       ├── outputs.tf          # Core infrastructure outputs (kubeconfigs, connection strings)
│       ├── providers.tf        # Cloud providers, helm setups, and backend state tracking
│       └── terraform.tfvars    # Environment variables overrides (No secrets!)
└── modules/
    ├── networking/             # Subnets, VPC, Cloud NAT, Cloud Router, Private Service Connect
    ├── kubernetes/             # GKE private clusters, autoscaling node pools, system services
    ├── database/               # Cloud SQL (PostgreSQ) high-availability clusters, backup windows
    ├── cache/                  # Cloud Memorystore Redis highly available cache clusters
    ├── secret_manager/         # Cloud Secret Manager / Vault orchestration systems
    ├── dns_certs/              # Cloud DNS zones, Google-managed certificates, and Load Balancers
    └── monitoring_logging/     # Log exports, Stackdriver monitoring dashboard configurations
```

---

## 3. Remote State Locking & Backend Reliability

*   **Remote Backend**: Development workstation state storage is strictly prohibited. State files must be persisted in **Google Cloud Storage (GCS) Buckets** or **AWS S3 Buckets** featuring versioning enabled.
*   **State Locking**: Simultaneous executions are prevented via database/table locking (GCS natively locks, DynamoDB handles locks on AWS S3).
*   **State Encryption**: Backend states are encrypted at-rest using **Customer-Managed Encryption Keys (CMEK)** inside Key Management Services (KMS).

---

## 4. Security Hardening & Zero-Trust Infrastructure

All modular templates must strictly enforce security boundaries at the infrastructure layer:

### 4.1 Private-Only Workloads
*   Absolutely no compute node, database engine, or cache cluster is allowed to bind to a public IPv4 address.
*   **Private Service Access (PSA)** or **VPC Network Peering** must be utilized for database and cache connections.
*   Workloads must run behind a **Cloud NAT** gateway with dedicated egress IPs to receive external patches without public ingress.

### 4.2 GKE Cluster Hardening
*   **Private Nodes**: GKE worker nodes must only reside on private subnets. The Kubernetes Control Plane is only accessible via authorized networks (CIDR whitelists).
*   **Shielded VMs**: Worker nodes utilize Shielded GKE VMs with Secure Boot enabled to prevent rootkits and system alterations.
*   **Workload Identity**: Map Kubernetes ServiceAccounts directly to Google Cloud IAM ServiceAccounts to prevent hardcoded JSON keys on the file system.

---

## 5. Drift Detection & Automation Gates

To preserve environment integrity, NexusCore enforces automated GitOps execution pipelines:
1.  **Format Compliance**: Pre-commit hooks run `terraform fmt -check` to preserve indentation standards.
2.  **Lint Verification**: Terraform manifests must pass structural inspections via `tflint`.
3.  **Static Vulnerability Scans**: Infrastructure blocks are parsed using `tfsec` or `checkov` to identify open ingress ports, unencrypted volumes, or default admin roles.
4.  **Dry-Run Approvals (`terraform plan`)**: Continuous integration generates a detailed execution plan before any merge. No resource modification can occur without visual authorization and a signed review from the SRE Lead.
