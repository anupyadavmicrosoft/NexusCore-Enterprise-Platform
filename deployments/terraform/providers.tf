# ==============================================================================
# Terraform Cloud Providers & State Declarations
# Standardized GKE and Cloud SQL provisioning on Google Cloud Platform (GCP)
# ==============================================================================

terraform {
  required_version = ">= 1.5.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.30.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.30.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.13.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6.0"
    }
  }

  # Configured for standard Google Cloud Storage backend
  backend "gcs" {
    bucket = "nexuscore-terraform-state-prod"
    prefix = "gke-cluster/state"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
  zone    = "${var.region}-a"
}

# Fetch client authentication credentials for cluster initialization
data "google_client_config" "default" {}

data "google_container_cluster" "primary" {
  name     = google_container_cluster.primary.name
  location = var.region
}

provider "kubernetes" {
  host                   = "https://${data.google_container_cluster.primary.endpoint}"
  token                  = data.google_client_config.default.access_token
  cluster_ca_certificate = base64decode(data.google_container_cluster.primary.master_auth[0].cluster_ca_certificate)
}

provider "helm" {
  kubernetes {
    host                   = "https://${data.google_container_cluster.primary.endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(data.google_container_cluster.primary.master_auth[0].cluster_ca_certificate)
  }
}
