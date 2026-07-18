output "vpc_id" {
  value       = google_compute_network.vpc.id
  description = "The ID of the VPC created"
}

output "vpc_name" {
  value       = google_compute_network.vpc.name
  description = "The name of the VPC created"
}

output "subnet_id" {
  value       = google_compute_subnetwork.private_subnet.id
  description = "The ID of the private subnet"
}

output "subnet_name" {
  value       = google_compute_subnetwork.private_subnet.name
  description = "The name of the private subnet"
}

output "pod_range_name" {
  value       = "gke-pods"
  description = "The name of the pods IP secondary range"
}

output "service_range_name" {
  value       = "gke-services"
  description = "The name of the services IP secondary range"
}
