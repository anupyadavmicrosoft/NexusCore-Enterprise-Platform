# 1. Private VPC Networking Subsystem
module "networking" {
  source     = "../../modules/networking"
  project_id = var.project_id
  region     = var.region
}

# 2. Hardened Private Google Kubernetes Engine
module "kubernetes" {
  source             = "../../modules/kubernetes"
  project_id         = var.project_id
  region             = var.region
  vpc_id             = module.networking.vpc_id
  subnet_id          = module.networking.subnet_id
  pod_range_name     = module.networking.pod_range_name
  service_range_name = module.networking.service_range_name
}

# 3. High-Availability Private Cloud SQL Cluster
module "database" {
  source      = "../../modules/database"
  project_id  = var.project_id
  region      = var.region
  vpc_id      = module.networking.vpc_id
  db_password = var.db_password
}

# 4. Multi-Zone Private Cache Cluster (Redis)
module "cache" {
  source     = "../../modules/cache"
  project_id = var.project_id
  region     = var.region
  vpc_id     = module.networking.vpc_id
}

# 5. Cloud Secret Manager for Key Vault Integrations
module "secret_manager" {
  source             = "../../modules/secret_manager"
  project_id         = var.project_id
  gke_nodes_sa_email = module.kubernetes.node_service_account
}

# 6. Global Load Balancing, DNS, and Cert Reserves
module "dns_certs" {
  source      = "../../modules/dns_certs"
  project_id  = var.project_id
  domain_name = var.domain_name
  subdomain   = var.subdomain
}

# 7. Stackdriver Dashboards, Audit Sinks, Alert Pools
module "monitoring_logging" {
  source             = "../../modules/monitoring_logging"
  project_id         = var.project_id
  notification_email = var.notification_email
}
