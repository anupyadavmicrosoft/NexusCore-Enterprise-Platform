variable "project_id" {
  type        = string
  description = "The GCP Project ID to host resources"
}

variable "region" {
  type        = string
  description = "The target GCP region"
}

variable "vpc_id" {
  type        = string
  description = "The VPC network ID to peer the private database to"
}

variable "db_version" {
  type        = string
  description = "The PostgreSQL database version"
  default     = "POSTGRES_15"
}

variable "tier" {
  type        = string
  description = "The machine tier for Cloud SQL"
  default     = "db-custom-4-16384" # 4 vCPUs, 16GB RAM for production workloads
}

variable "db_password" {
  type        = string
  description = "The primary database password"
  sensitive   = true
}
