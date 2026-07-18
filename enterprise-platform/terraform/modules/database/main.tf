resource "google_sql_database_instance" "postgres" {
  name             = "nexuscore-prod-postgres"
  database_version = var.db_version
  region           = var.region
  project          = var.project_id

  # Ensures GKE and other internal resources can connect privately
  deletion_protection = false # Set to true in absolute production environments

  settings {
    tier              = var.tier
    availability_type = "REGIONAL" # Regional High Availability (Primary + Standby)

    disk_size             = 100 # 100 GB starting size
    disk_type             = "PD_SSD"
    disk_autoresize       = true
    disk_autoresize_limit = 1000 # Limit auto-resize to 1TB

    ip_configuration {
      ipv4_enabled    = false # Disable public IP address entirely
      private_network = var.vpc_id
    }

    backup_configuration {
      enabled                        = true
      start_time                     = "02:00" # 2 AM daily maintenance/backup
      location                       = "us"
      point_in_time_recovery_enabled = true
      transaction_log_retention_days = 7
    }

    maintenance_window {
      day          = 7 # Sunday
      hour         = 3 # 3 AM
      update_track = "stable"
    }

    database_flags {
      name  = "log_connections"
      value = "on"
    }

    database_flags {
      name  = "log_disconnections"
      value = "on"
    }
  }
}

resource "google_sql_database" "db" {
  name     = "nexuscore_enterprise"
  instance = google_sql_database_instance.postgres.name
  project  = var.project_id
}

resource "google_sql_user" "admin_user" {
  name     = "postgres"
  instance = google_sql_database_instance.postgres.name
  password = var.db_password
  project  = var.project_id
}
