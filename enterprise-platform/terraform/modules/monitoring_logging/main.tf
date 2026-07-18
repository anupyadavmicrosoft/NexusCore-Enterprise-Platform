# Create Cloud Storage bucket to archive system and audit logs
resource "google_storage_bucket" "log_archive" {
  name                     = "nexuscore-${var.project_id}-log-archive"
  location                 = "US"
  project                  = var.project_id
  force_destroy            = false
  public_access_prevention = "enforced"

  lifecycle_rule {
    condition {
      age = 365 # Keep logs for 1 year for compliance/audit
    }
    action {
      type = "Delete"
    }
  }
}

# Create Log Export Sink for GKE and DB Audit logs
resource "google_logging_project_sink" "audit_sink" {
  name        = "nexuscore-prod-audit-sink"
  destination = "storage.googleapis.com/${google_storage_bucket.log_archive.name}"
  filter      = "resource.type=\"gke_cluster\" OR resource.type=\"cloud_sql_database\""
  project     = var.project_id

  # Export with unique writer identity
  unique_writer_identity = true
}

# Grant Storage Object Creator permissions to the unique sink identity
resource "google_project_iam_binding" "log_writer" {
  project = var.project_id
  role    = "roles/storage.objectCreator"

  members = [
    google_logging_project_sink.audit_sink.writer_identity,
  ]
}

# Alerts Notification Channel
resource "google_monitoring_notification_channel" "email" {
  display_name = "NexusCore Operations SRE Email Channel"
  type         = "email"
  project      = var.project_id

  labels = {
    email_address = var.notification_email
  }
}

# Cloud SQL High CPU Alarm Policy
resource "google_monitoring_alert_policy" "sql_cpu_high" {
  display_name = "Cloud SQL Instance CPU Utilization Alert"
  combiner     = "OR"
  project      = var.project_id

  conditions {
    display_name = "CPU utilization over 85% for 5 mins"
    condition_threshold {
      filter          = "resource.type = \"cloud_sql_database\" AND metric.type = \"cloudsql.googleapis.com/database/cpu/utilization\""
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 0.85

      trigger {
        count = 1
      }

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MEAN"
      }
    }
  }

  notification_channels = [
    google_monitoring_notification_channel.email.name
  ]

  user_labels = {
    severity = "warning"
  }
}
