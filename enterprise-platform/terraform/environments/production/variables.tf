variable "project_id" {
  type        = string
  description = "The GCP Project ID to host resources"
}

variable "region" {
  type        = string
  description = "The target GCP region"
  default     = "us-central1"
}

variable "domain_name" {
  type        = string
  description = "Root domain mapping"
  default     = "nexuscore-enterprise.com"
}

variable "subdomain" {
  type        = string
  description = "Subdomain routing endpoint for the gateway"
  default     = "api.nexuscore-enterprise.com"
}

variable "db_password" {
  type        = string
  description = "The administrator database password"
  sensitive   = true
}

variable "notification_email" {
  type        = string
  description = "Target email for alert routing logs"
  default     = "ops@nexuscore-enterprise.com"
}
