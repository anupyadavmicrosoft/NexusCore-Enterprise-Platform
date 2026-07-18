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
  description = "The VPC network ID to peer Redis to"
}

variable "memory_size_gb" {
  type        = number
  description = "Memory size of Redis instance in GB"
  default     = 5
}

variable "redis_version" {
  type        = string
  description = "The engine version of Redis"
  default     = "REDIS_7_0"
}
