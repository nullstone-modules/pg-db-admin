locals {
  storage_location = local.region_prefix == "us" ? "US" : (local.region_prefix == "eu" ? "EU" : "ASIA")
  package_filename = "${path.module}/files/pg-db-admin.zip"
}

resource "google_storage_bucket" "binaries" {
  name          = "${var.name}-binaries"
  location      = local.storage_location
  labels        = var.labels
  force_destroy = true
}

resource "google_storage_bucket_object" "binary" {
  bucket = google_storage_bucket.binaries.name
  name   = "pg-db-admin.zip"
  source = local.package_filename
}
