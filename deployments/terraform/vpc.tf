# ==============================================================================
# Terraform Google Cloud Virtual Private Cloud (VPC) Subnets & Gateway Setup
# Implements complete network isolation for internal node pools
# ==============================================================================

# Custom VPC Network
resource "google_compute_network" "vpc_network" {
  name                    = "nexuscore-${var.environment}-vpc"
  auto_create_subnetworks = false
  routing_mode            = "REGIONAL"
  mtu                     = 1460
}

# Private Subnet for GKE Nodes and Pods Egress
resource "google_compute_subnetwork" "private_subnet" {
  name                     = "nexuscore-${var.environment}-gke-subnet"
  ip_cidr_range            = var.gke_nodes_cidr
  region                   = var.region
  network                  = google_compute_network.vpc_network.id
  private_ip_google_access = true

  # IP range aliasing for native Kubernetes routing (VPC-Native cluster configuration)
  secondary_ip_range {
    range_name    = "gke-pods-range"
    ip_cidr_range = var.gke_pods_cidr
  }

  secondary_ip_range {
    range_name    = "gke-services-range"
    ip_cidr_range = var.gke_services_cidr
  }
}

# Cloud Router required for NAT egress setup
resource "google_compute_router" "router" {
  name    = "nexuscore-${var.environment}-nat-router"
  region  = var.region
  network = google_compute_network.vpc_network.id

  bgp {
    asn = 64514
  }
}

# Cloud NAT for private subnets to reach out for external dependencies (e.g. download Docker base images)
resource "google_compute_router_nat" "nat_gateway" {
  name                               = "nexuscore-${var.environment}-nat-gateway"
  router                             = google_compute_router.router.name
  region                             = var.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "LIST_OF_SUBNETWORKS"

  subnetwork {
    name                    = google_compute_subnetwork.private_subnet.id
    source_ip_ranges_to_nat = ["PRIMARY_IP_RANGE", "LIST_OF_SECONDARY_IP_RANGES"]
    secondary_ip_range_names = [
      "gke-pods-range",
      "gke-services-range"
    ]
  }

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}
