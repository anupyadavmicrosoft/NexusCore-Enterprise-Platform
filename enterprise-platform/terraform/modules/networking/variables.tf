variable "project_id" {
  type        = string
  description = "The GCP Project ID to host resources"
}

variable "region" {
  type        = string
  description = "The target GCP region"
  default     = "us-central1"
}

variable "vpc_name" {
  type        = string
  description = "The name of the VPC network"
  default     = "nexuscore-prod-vpc"
}

variable "subnet_cidr" {
  type        = string
  description = "CIDR block for the primary private node subnet"
  default     = "10.0.0.0/20"
}

variable "pod_cidr_range" {
  type        = string
  description = "CIDR block for Kubernetes pods"
  default     = "10.128.0.0/14"
}

variable "service_cidr_range" {
  type        = string
  description = "CIDR block for Kubernetes services"
  default     = "10.132.0.0/18"
}
