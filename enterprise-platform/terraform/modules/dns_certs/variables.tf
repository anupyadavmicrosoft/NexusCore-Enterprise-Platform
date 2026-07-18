variable "project_id" {
  type        = string
  description = "The GCP Project ID to host resources"
}

variable "domain_name" {
  type        = string
  description = "The root domain name"
  default     = "nexuscore-enterprise.com"
}

variable "subdomain" {
  type        = string
  description = "The API subdomain pointing to our Edge Load Balancer"
  default     = "api.nexuscore-enterprise.com"
}
