data "google_client_config" "this" {}

locals {
  project_id    = data.google_client_config.this.project
  region        = data.google_client_config.this.region
  region_prefix = lower(substr(local.region, 0, 2))
}
