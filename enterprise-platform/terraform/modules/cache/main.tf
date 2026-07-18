resource "google_redis_instance" "redis" {
  name               = "nexuscore-prod-redis"
  tier               = "STANDARD_HA" # Highly Available primary-replica failover cluster
  memory_size_gb     = var.memory_size_gb
  region             = var.region
  project            = var.project_id
  redis_version      = var.redis_version
  auth_enabled       = true # Enforce connection passwords

  # Connect privately
  authorized_network = var.vpc_id
  connect_mode       = "PRIVATE_SERVICE_ACCESS"

  maintenance_policy {
    weekly_maintenance_window {
      day = "SUNDAY"
      start_time {
        hours   = 4
        minutes = 0
        seconds = 0
        nanos   = 0
      }
    }
  }

  labels = {
    environment = "production"
    service     = "caching"
  }
}
