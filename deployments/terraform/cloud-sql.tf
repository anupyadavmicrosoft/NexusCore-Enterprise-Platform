# ==============================================================================
# Terraform Google Cloud SQL (PostgreSQL) Provisioning
# Implements private service peering connection to custom VPC network
# ==============================================================================

# Private IP Allocation Range for private DB connection
resource "google_compute_global_address" "private_ip_alloc" {
  name          = "nexuscore-${var.environment}-db-private-ip"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = google_compute_network.vpc_network.id
}

# Establishing private peering bridge between VPC and Google Services Network
resource "google_service_networking_connection" "private_vpc_connection" {
  network                 = google_compute_network.vpc_network.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_alloc.name]
}

# Random high-entropy generator for PostgreSQL Master User credentials
resource "random_password" "db_password" {
  length           = 24
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

# High-Availability PostgreSQL Cluster
resource "google_sql_database_instance" "postgres" {
  name             = "nexuscore-${var.environment}-postgresql-instance"
  database_version = "POSTGRES_15"
  region           = var.region

  # Enforce completion of the private service networking bridge before instance boot
  depends_on = [google_service_networking_connection.private_vpc_connection]

  settings {
    tier              = var.db_tier
    availability_type = "REGIONAL" # High Availability with secondary standby failover node
    disk_size         = 50
    disk_type         = "PD_SSD"
    disk_autoresize   = true

    ip_configuration {
      ipv4_enabled    = false # Wholly private instance (No public IP address)
      private_network = google_compute_network.vpc_network.id
      ssl_mode        = "ENCRYPTED_ONLY"
    }

    # Backup and point-in-time recovery configurations
    backup_configuration {
      enabled                        = true
      start_time                     = "03:00" # Run daily snapshots outside of active business hours
      point_in_time_recovery_enabled = true
      transaction_log_retention_days = 7
    }

    # Custom PostgreSQL Performance Flags for Enterprise Workloads
    database_flags {
      name  = "max_connections"
      value = "250"
    }

    database_flags {
      name  = "shared_buffers"
      value = "4194304" # 4GB shared buffers for high-speed indexing
    }

    database_flags {
      name  = "log_connections"
      value = "on"
    }

    database_flags {
      name  = "log_disconnections"
      value = "on"
    }

    database_flags {
      name  = "log_min_messages"
      value = "warning"
    }
  }

  lifecycle {
    prevent_destroy = false # Set to true in real environment to prevent accidental loss
  }
}

# Primary system database catalog creation
resource "google_sql_database" "nexus_db" {
  name     = "nexuscore"
  instance = google_sql_database_instance.postgres.name
}

# Primary unprivileged application user
resource "google_sql_user" "app_user" {
  name     = "nexus_app_svc"
  instance = google_sql_database_instance.postgres.name
  password = random_password.db_password.result
}
