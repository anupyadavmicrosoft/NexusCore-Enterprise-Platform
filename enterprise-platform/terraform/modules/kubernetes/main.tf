resource "google_container_cluster" "gke" {
  name     = var.cluster_name
  location = var.region
  project  = var.project_id

  network    = var.vpc_id
  subnetwork = var.subnet_id

  # Deleting standard pool to use custom optimized node pools
  remove_default_node_pool = true
  initial_node_count       = 1

  # IP range mapping
  ip_allocation_policy {
    cluster_secondary_range_name  = var.pod_range_name
    services_secondary_range_name = var.service_range_name
  }

  # Establish GKE Private Cluster posture
  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = false # Private nodes, but public master endpoint with restriction
    master_ipv4_cidr_block  = "172.16.0.0/28"
  }

  master_authorized_networks_config {
    cidr_blocks {
      cidr_block   = "0.0.0.0/0" # In real prod, this is limited to bastion hosts or corporate VPN CIDRs
      display_name = "Whitelisted Admin Network"
    }
  }

  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  # Security configurations
  enable_shielded_nodes = true

  release_channel {
    channel = "STABLE"
  }
}

resource "google_container_node_pool" "primary_nodes" {
  name       = "nexuscore-prod-primary-node-pool"
  location   = var.region
  cluster    = google_container_cluster.gke.name
  project    = var.project_id
  node_count = var.min_node_count

  autoscaling {
    min_node_count = var.min_node_count
    max_node_count = var.max_node_count
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }

  node_config {
    preemptible  = false
    machine_type = var.machine_type

    # Workload identity configuration
    service_account = google_service_account.gke_nodes_sa.email
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]

    # Hardening via Shielded VMs
    shielded_instance_config {
      enable_secure_boot          = true
      enable_integrity_monitoring = true
    }

    labels = {
      environment = "production"
      role        = "primary-compute"
    }

    metadata = {
      disable-legacy-endpoints = "true"
    }
  }
}

# Dedicated service account for GKE worker VMs to restrict privilege scope
resource "google_service_account" "gke_nodes_sa" {
  account_id   = "nexuscore-gke-nodes-sa"
  display_name = "NexusCore GKE Nodes VM Identity"
  project      = var.project_id
}
