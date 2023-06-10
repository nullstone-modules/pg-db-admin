locals {
  truncated_len = min(length(var.name), 28 - length("invoker-"))
  invoker_name  = "invoker-${substr(var.name, 0, local.truncated_len)}"
}

resource "google_service_account" "invoker" {
  account_id   = local.invoker_name
  display_name = "Invoker for pg db admin ${var.name}"
}

resource "google_service_account_key" "invoker" {
  service_account_id = google_service_account.invoker.account_id
}

resource "google_project_iam_member" "invoker_basic" {
  project = local.project_id
  role    = "roles/run.invoker"
  member  = "serviceAccount:${google_service_account.invoker.email}"
}
