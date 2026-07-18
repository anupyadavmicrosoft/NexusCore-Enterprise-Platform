output "instance_name" {
  value       = google_sql_database_instance.postgres.name
  description = "The database instance name"
}

output "private_ip" {
  value       = google_sql_database_instance.postgres.private_ip_address
  description = "The private IP address of the database"
}

output "connection_name" {
  value       = google_sql_database_instance.postgres.connection_name
  description = "The database connection name used by Cloud SQL Proxy"
}

output "db_name" {
  value       = google_sql_database.db.name
  description = "The name of the database created"
}
