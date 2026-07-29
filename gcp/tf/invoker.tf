locals {
  truncated_len = min(length(var.name), 28 - length("invoker-"))
  invoker_name  = "invoker-${substr(var.name, 0, local.truncated_len)}"
}

resource "google_service_account" "invoker" {
  account_id   = local.invoker_name
  display_name = "Invoker for pg db admin ${var.name}"
}

// Grant invoke permission on *this* function only.
//
// A Cloud Function (2nd gen) is backed by a Cloud Run service, so invoking it over HTTP requires
// `roles/run.invoker` on that service. This was previously granted at project scope, which let any
// datastore's invoker call every Cloud Run service in the project -- including other datastores'
// db-admin functions (the db-admin API has no per-caller authorization, so that is full DBA on every
// database in the project) and every app deployed on Cloud Run.
//
// `service_config[0].service` is the underlying Cloud Run service name; `basename` tolerates providers
// that return it as a fully-qualified resource path.
resource "google_cloud_run_service_iam_member" "invoker_basic" {
  project  = local.project_id
  location = google_cloudfunctions2_function.function.location
  service  = basename(google_cloudfunctions2_function.function.service_config[0].service)
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.invoker.email}"
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
