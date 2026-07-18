terraform {
  required_version = ">= 1.3.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 4.80.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.22.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.10.0"
    }
  }

  # Production remote state backend in GCS (parameters passed dynamically or locally in tfvars)
  backend "gcs" {
    bucket = "nexuscore-terraform-state-prod"
    prefix = "terraform/state"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# Fetch token credentials for our GKE cluster to bootstrap configurations
data "google_client_config" "default" {}

provider "kubernetes" {
  host                   = "https://${module.kubernetes.endpoint}"
  token                  = data.google_client_config.default.access_token
  cluster_ca_certificate = base64decode(module.kubernetes.ca_certificate)
}

provider "helm" {
  kubernetes {
    host                   = "https://${module.kubernetes.endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(module.kubernetes.ca_certificate)
  }
}
