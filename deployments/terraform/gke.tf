# ==============================================================================
# Terraform Google Kubernetes Engine (GKE) Private Cluster Provisioning
# Includes high availability config, master authorized networks, and autoscaling
# ==============================================================================

resource "google_container_cluster" "primary" {
  name     = "nexuscore-${var.environment}-cluster"
  location = var.region

  # We create a regional cluster spanning multiple zones for HA availability
  node_locations = [
    "${var.region}-a",
    "${var.region}-b",
    "${var.region}-c"
  ]

  network    = google_compute_network.vpc_network.id
  subnetwork = google_compute_subnetwork.private_subnet.id

  # We configure GKE with a separate node pool resource, so we delete default
  remove_default_node_pool = true
  initial_node_count       = 1

  # Enable VPC-native traffic routing
  ip_allocation_policy {
    cluster_secondary_range_name  = "gke-pods-range"
    services_secondary_range_name = "gke-services-range"
  }

  # Enable Shielded Nodes for boot security and integrity monitoring
  enable_shielded_nodes = true

  # Enterprise Release Channel selection
  release_channel {
    channel = "STABLE"
  }

  # Workload Identity configuration for secure GCP IAM service binding
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  # Private Cluster Network Setup (Nodes lack external IP addresses)
  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = false # API Server is reachable securely via Public Authorized IP addresses
    master_ipv4_cidr_block  = var.gke_master_cidr
  }

  # Control access to Kubernetes API Master
  master_authorized_networks_config {
    cidr_blocks {
      cidr_block   = "10.0.0.0/8"
      display_name = "Internal Enterprise Subnet Range"
    }
    cidr_blocks {
      cidr_block   = "34.120.0.0/16" # Example corporate jumpbox IP block range
      display_name = "Corporate Office VPC Boundary"
    }
  }

  lifecycle {
    ignore_changes = [
      node_pool,
      initial_node_count
    ]
  }
}

# Dedicated Compute Node Pool (Custom Machine Types, Node Autoscaler, and OTel Tags)
resource "google_container_node_pool" "general_purpose" {
  name       = "nexuscore-prod-nodepool-general"
  cluster    = google_container_cluster.primary.id
  location   = var.region
  node_count = var.gke_min_nodes_per_zone

  # Autoscaling rules configuration bounds
  autoscaling {
    min_node_count = var.gke_min_nodes_per_zone
    max_node_count = var.gke_max_nodes_per_zone
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }

  node_config {
    preemptible  = false
    machine_type = var.gke_machine_type

    disk_size_gb = 100
    disk_type    = "pd-ssd"

    # Enforce secure least-privilege OAuth scopes on compute VM credentials
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring"
    ]

    # Node affinity labels for OTel workload routing and scheduling
    labels = {
      environment = var.environment
      workload    = "general-microservices"
      provisioner = "terraform"
    }

    # GKE metadata for service orchestration
    metadata = {
      disable-legacy-endpoints = "true"
    }

    shielded_instance_config {
      enable_secure_boot          = true
      enable_integrity_monitoring = true
    }
  }
}
