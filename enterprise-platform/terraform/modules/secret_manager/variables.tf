variable "project_id" {
  type        = string
  description = "The GCP Project ID to host resources"
}

variable "gke_nodes_sa_email" {
  type        = string
  description = "Service account email of the GKE nodes to grant read access"
}

variable "secret_names" {
  type        = list(string)
  description = "List of secrets to register in Cloud Secret Manager"
  default     = ["database-dsn", "redis-password", "jwt-secret"]
}
