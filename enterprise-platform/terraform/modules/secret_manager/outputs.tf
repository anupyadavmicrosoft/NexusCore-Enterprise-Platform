output "secret_ids" {
  value       = { for k, v in google_secret_manager_secret.secret : k => v.id }
  description = "A map of secret keys to Secret Manager IDs"
}
