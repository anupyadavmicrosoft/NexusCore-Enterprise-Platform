variable "project_id" {
  type        = string
  description = "The GCP Project ID to host resources"
}

variable "notification_email" {
  type        = string
  description = "Primary engineering email for alerting triggers"
  default     = "ops@nexuscore-enterprise.com"
}
