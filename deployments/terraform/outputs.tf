# ==============================================================================
# Terraform Outputs Definitions
# Standard output endpoints for CI/CD pipeline integrations and configuration
# ==============================================================================

output "vpc_network_id" {
  value       = google_compute_network.vpc_network.id
  description = "The fully qualified unique resource identifier of the VPC network"
}

output "private_subnet_id" {
  value       = google_compute_subnetwork.private_subnet.id
  description = "The fully qualified unique resource identifier of the GKE compute subnet"
}

output "gke_cluster_endpoint" {
  value       = google_container_cluster.primary.endpoint
  description = "IP address endpoint of the primary Kubernetes cluster API server"
}

output "gke_cluster_name" {
  value       = google_container_cluster.primary.name
  description = "The cluster name identifier utilized for cloud provider logins"
}

output "gke_cluster_ca_certificate" {
  value       = google_container_cluster.primary.master_auth[0].cluster_ca_certificate
  sensitive   = true
  description = "Base64 encoded certificate public key authority for secure client handshake"
}

output "cloud_sql_instance_connection_name" {
  value       = google_sql_database_instance.postgres.connection_name
  description = "GCP Cloud SQL Proxy connection name string formatted as project:region:instance"
}

output "cloud_sql_private_ip" {
  value       = google_sql_database_instance.postgres.private_ip_address
  description = "The private internal IP address allocated for direct database connections"
}

output "database_user" {
  value       = google_sql_user.app_user.name
  description = "PostgreSQL authenticated database database username"
}

output "database_password" {
  value       = random_password.db_password.result
  sensitive   = true
  description = "High entropy database password generated during system provisioning"
}
