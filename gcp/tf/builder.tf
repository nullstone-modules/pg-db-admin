// The builder service account has enough permissions to build the code for the cloud function

locals {
  truncated_builder_len = min(length(var.name), 28 - length("builder-"))
  builder_name          = "builder-${substr(var.name, 0, local.truncated_builder_len)}"
}

resource "google_service_account" "builder" {
  account_id   = local.builder_name
  display_name = "Builder for pg db admin ${var.name}"
}

// Allow cloud builder to impersonate the builder service account
resource "google_service_account_iam_member" "cloudbuild_impersonate_cf_build" {
  service_account_id = google_service_account.builder.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${local.project_number}@cloudbuild.gserviceaccount.com"
}

resource "google_project_iam_member" "builder_build" {
  project = local.project_id
  role    = "roles/cloudbuild.builds.builder"
  member  = "serviceAccount:${google_service_account.builder.email}"
}

resource "google_project_iam_member" "builder_publish_artifacts" {
  project = local.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${google_service_account.builder.email}"
}

// The builder only needs to read the source zip out of the binaries bucket, so scope the grant to that
// bucket rather than every bucket in the project.
//
// NOTE: this is hygiene, not a meaningful reduction in effective access. `roles/cloudbuild.builds.builder`
// above already carries project-wide storage object read/write. Actually tightening the builder means
// replacing that role with a minimal custom one, which risks breaking function builds.
resource "google_storage_bucket_iam_member" "builder_code_access" {
  bucket = google_storage_bucket.binaries.name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${google_service_account.builder.email}"
}
