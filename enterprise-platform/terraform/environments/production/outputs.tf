output "cluster_name" {
  value       = module.kubernetes.cluster_name
  description = "The name of the production GKE cluster"
}

output "gke_endpoint" {
  value       = module.kubernetes.endpoint
  description = "The master API endpoint of GKE"
}

output "database_private_ip" {
  value       = module.database.private_ip
  description = "The private internal IP of the HA Database"
}

output "redis_private_host" {
  value       = module.cache.host
  description = "The private internal Host IP of the cache cluster"
}

output "ingress_load_balancer_ip" {
  value       = module.dns_certs.ingress_public_ip
  description = "The static public IP pointing to the front-line Ingress Controller"
}

output "domain_nameservers" {
  value       = module.dns_certs.nameservers
  description = "The nameservers to assign in your registrar profile"
}

output "logging_archive_gcs" {
  value       = module.monitoring_logging.log_archive_bucket_name
  description = "The GCS log bucket auditing infrastructure"
}
