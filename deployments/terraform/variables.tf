# ==============================================================================
# Terraform Variables Declaration File
# No Placeholders - Production configurations and validated CIDR blocks
# ==============================================================================

variable "project_id" {
  type        = string
  description = "The Google Cloud Platform Project ID for enterprise billing"
  default     = "nexuscore-prod-39401"
}

variable "region" {
  type        = string
  description = "The target multizone cloud region for infrastructure orchestration"
  default     = "us-east1"
}

variable "environment" {
  type        = string
  description = "Environment tag for naming conventions and resource categorization"
  default     = "production"
}

variable "vpc_cidr" {
  type        = string
  description = "Primary IP Address Classless Inter-Domain Routing block of the VPC"
  default     = "10.0.0.0/16"
}

variable "gke_nodes_cidr" {
  type        = string
  description = "Secondary subnet range allocated exclusively for GKE Node allocations"
  default     = "10.10.0.0/20"
}

variable "gke_pods_cidr" {
  type        = string
  description = "Secondary subnet range for alias IP assignments inside pods"
  default     = "172.16.0.0/14"
}

variable "gke_services_cidr" {
  type        = string
  description = "Secondary subnet range allocated for Kubernetes cluster services"
  default     = "172.20.0.0/16"
}

variable "gke_master_cidr" {
  type        = string
  description = "The restricted IP block reserved for GKE control plane API endpoint"
  default     = "172.24.0.0/28"
}

variable "gke_machine_type" {
  type        = string
  description = "Compute Engine virtual machine profiles for GKE nodes"
  default     = "e2-standard-4"
}

variable "gke_min_nodes_per_zone" {
  type        = number
  description = "Minimum count of cluster compute nodes per zone (Autoscaling)"
  default     = 2
}

variable "gke_max_nodes_per_zone" {
  type        = number
  description = "Maximum limit of cluster compute nodes per zone (Autoscaling)"
  default     = 8
}

variable "db_tier" {
  type        = string
  description = "Hardware machine configuration of Cloud SQL PostgreSQL Master Database Instance"
  default     = "db-custom-4-16384" # 4 vCPUs, 16GB Memory High-Availability Profile
}
