output "cluster_name" {
  value       = google_container_cluster.gke.name
  description = "The name of the GKE cluster"
}

output "endpoint" {
  value       = google_container_cluster.gke.endpoint
  description = "The master endpoint of the GKE cluster"
}

output "ca_certificate" {
  value       = google_container_cluster.gke.master_auth[0].cluster_ca_certificate
  description = "The public certificate of the GKE cluster"
}

output "node_service_account" {
  value       = google_service_account.gke_nodes_sa.email
  description = "The ServiceAccount used by the GKE workers"
}

output "workload_pool" {
  value       = "${var.project_id}.svc.id.goog"
  description = "Workload Identity resource path pool"
}
