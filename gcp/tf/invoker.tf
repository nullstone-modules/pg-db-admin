locals {
  truncated_len = min(length(var.name), 28 - length("invoker-"))
  invoker_name  = "invoker-${substr(var.name, 0, local.truncated_len)}"
}

resource "google_service_account" "invoker" {
  account_id   = local.invoker_name
  display_name = "Invoker for pg db admin ${var.name}"
}

resource "google_project_iam_member" "invoker_basic" {
  project = local.project_id
  role    = "roles/run.invoker"
  member  = "serviceAccount:${google_service_account.invoker.email}"
}

// Allow agents to impersonate the invoker agent
resource "google_service_account_iam_binding" "invoker_impersonators" {
  service_account_id = google_service_account.invoker.id
  role               = "roles/iam.serviceAccountTokenCreator"
  members            = [for email in var.invoker_impersonators : "serviceAccount:${email}"]
}

// Allow agents to create open id token
resource "google_service_account_iam_binding" "invoker_idtoken" {
  service_account_id = google_service_account.invoker.id
  role               = "roles/iam.serviceAccountOpenIdTokenCreator"
  members            = [for email in var.invoker_impersonators : "serviceAccount:${email}"]
}
