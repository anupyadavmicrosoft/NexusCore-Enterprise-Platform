variable "project_id" {
  type        = string
  description = "The GCP Project ID to host resources"
}

variable "region" {
  type        = string
  description = "The target GCP region"
}

variable "cluster_name" {
  type        = string
  description = "The name of the GKE cluster"
  default     = "nexuscore-prod-cluster"
}

variable "vpc_id" {
  type        = string
  description = "The network VPC ID"
}

variable "subnet_id" {
  type        = string
  description = "The private subnet ID"
}

variable "pod_range_name" {
  type        = string
  description = "Secondary range name for GKE pods"
}

variable "service_range_name" {
  type        = string
  description = "Secondary range name for GKE services"
}

variable "min_node_count" {
  type        = number
  description = "Minimum node count per zone for autoscaling"
  default     = 1
}

variable "max_node_count" {
  type        = number
  description = "Maximum node count per zone for autoscaling"
  default     = 5
}

variable "machine_type" {
  type        = string
  description = "GKE node compute type"
  default     = "e2-standard-4"
}
