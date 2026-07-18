resource "google_secret_manager_secret" "secret" {
  for_each  = toset(var.secret_names)
  secret_id = "nexuscore-${each.value}"
  project   = var.project_id

  replication {
    automatic = true
  }

  labels = {
    environment = "production"
    owner       = "nexuscore-ops"
  }
}

# Grant access exclusively to GKE ServiceAccount to fetch database/JWT secrets
resource "google_secret_manager_secret_iam_member" "gke_accessor" {
  for_each  = toset(var.secret_names)
  project   = var.project_id
  secret_id = google_secret_manager_secret.secret[each.value].id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.gke_nodes_sa_email}"
}
