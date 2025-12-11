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
  for_each = var.invoker_impersonators

  service_account_id = google_service_account.invoker.id
  role               = "roles/iam.serviceAccountTokenCreator"
  members            = ["serviceAccount:${each.value}"]
}
