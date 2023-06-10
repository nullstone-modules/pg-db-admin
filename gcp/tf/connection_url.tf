resource "google_secret_manager_secret" "db_admin_pg" {
  secret_id = "${var.name}_conn_url"
  labels    = var.labels

  replication {
    automatic = true
  }
}

resource "google_secret_manager_secret_version" "db_admin_pg" {
  secret      = google_secret_manager_secret.db_admin_pg.id
  secret_data = "postgres://${urlencode(var.username)}:${urlencode(var.password)}@${var.host}:${var.port}/${urlencode(var.database)}"
  enabled     = true
}
