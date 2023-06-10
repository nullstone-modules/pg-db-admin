locals {
  truncated_executor_len = min(length(var.name), 28 - length("executor-"))
  executor_name  = "executor-${substr(var.name, 0, local.truncated_executor_len)}"
}

resource "google_service_account" "executor" {
  account_id   = local.executor_name
  display_name = "Executor for pg db admin ${var.name}"
}

resource "google_project_iam_member" "executor_artifacts" {
  project = local.project_id
  role    = "roles/artifactregistry.reader"
  member  = "serviceAccount:${google_service_account.executor.email}"
}

resource "google_secret_manager_secret_iam_member" "executor_secrets" {
  project   = local.project_id
  secret_id = google_secret_manager_secret.db_admin_pg.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.executor.email}"
}
