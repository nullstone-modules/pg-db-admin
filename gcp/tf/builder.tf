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

// The builder needs to read the source zip out of two places:
//   1. our `binaries` bucket, where this module uploads it
//   2. the GCP-managed gen2 staging buckets, because Cloud Functions gen2 does not build straight from `binaries`.
//      It copies the zip into `gcf-v2-sources-<project_number>-<region>` and Cloud Build reads it from there.
//      Without this, the build fails with "Access to bucket gcf-v2-... denied".
//
// This is a project-level binding narrowed by an IAM condition rather than three bucket-scoped bindings.
// A bucket-scoped binding requires the bucket to already exist, and GCP does not create the gen2 staging
// buckets until the first gen2 deploy in the region, so on a new project those grants 404 on the first apply.
// The condition is only a CEL string, so it applies cleanly before the buckets exist while keeping the grant scoped to the same set of buckets.
//
// NOTE: `roles/cloudbuild.builds.builder` above does *not* cover this. It carries object create/write for pushing build artifacts, not object read across arbitrary buckets.
locals {
  builder_source_buckets = [
    google_storage_bucket.binaries.name,
    "gcf-v2-sources-${local.project_number}-${local.region}",
    "gcf-v2-uploads-${local.project_number}-${local.region}",
  ]

  // Matches each bucket itself (needed for storage.objects.list) and every object inside it.
  builder_source_bucket_condition = join(" || ", [
    for bucket in local.builder_source_buckets :
    "resource.name.startsWith(\"projects/_/buckets/${bucket}\")"
  ])
}

resource "google_project_iam_member" "builder_code_access" {
  project = local.project_id
  role    = "roles/storage.objectViewer"
  member  = "serviceAccount:${google_service_account.builder.email}"

  condition {
    title       = "source-buckets-only"
    description = "Read the function source zip from the binaries bucket and the gen2 staging buckets"
    expression  = local.builder_source_bucket_condition
  }
}
