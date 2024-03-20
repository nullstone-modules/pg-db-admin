data "google_client_config" "this" {}

locals {
  project_id    = data.google_client_config.this.project
  region        = data.google_client_config.this.region
  region_prefix = lower(substr(local.region, 0, 2))
}

resource "google_project_service" "run" {
  service                    = "run.googleapis.com"
  disable_dependent_services = false
  disable_on_destroy         = false
}
resource "google_project_service" "cloudbuild" {
  service                    = "cloudbuild.googleapis.com"
  disable_dependent_services = false
  disable_on_destroy         = false
}
resource "google_project_service" "function" {
  service                    = "cloudfunctions.googleapis.com"
  disable_dependent_services = false
  disable_on_destroy         = false
}
// Artifact Registry is needed to create a Cloud Function (db-admin) even though we're not using it
resource "google_project_service" "artifact_registry" {
  service                    = "artifactregistry.googleapis.com"
  disable_dependent_services = false
  disable_on_destroy         = false
}
