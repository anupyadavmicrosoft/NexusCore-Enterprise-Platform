output "log_archive_bucket_name" {
  value       = google_storage_bucket.log_archive.name
  description = "The name of the GCS bucket archiving cluster logs"
}

output "alert_channel_id" {
  value       = google_monitoring_notification_channel.email.id
  description = "The ID of the notification alert channel"
}
