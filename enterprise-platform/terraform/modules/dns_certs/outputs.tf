output "nameservers" {
  value       = google_dns_managed_zone.dns_zone.name_servers
  description = "The list of authority nameservers assigned to our public DNS zone"
}

output "ingress_public_ip" {
  value       = google_compute_global_address.ingress_ip.address
  description = "The reserved external IP address pointing to the ingress gateway"
}
