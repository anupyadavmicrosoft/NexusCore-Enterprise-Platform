resource "google_dns_managed_zone" "dns_zone" {
  name        = "nexuscore-public-zone"
  dns_name    = "${var.domain_name}."
  description = "NexusCore Public Domain Authority DNS Zone"
  project     = var.project_id

  labels = {
    environment = "production"
  }
}

# Reserve public static IP address for ingress load balancer
resource "google_compute_global_address" "ingress_ip" {
  name        = "nexuscore-prod-ingress-ip"
  description = "Reserved global IPv4 for external Load Balancer routing"
  project     = var.project_id
}

# Route the subdomain traffic directly to the reserved public IP address
resource "google_dns_record_set" "api_record" {
  name         = "${var.subdomain}."
  managed_zone = google_dns_managed_zone.dns_zone.name
  type         = "A"
  ttl          = 300
  project      = var.project_id

  rrdatas = [
    google_compute_global_address.ingress_ip.address
  ]
}
