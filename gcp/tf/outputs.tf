output "function_name" {
  value = google_cloudfunctions2_function.function.name
}

output "function_url" {
  value = try(google_cloudfunctions2_function.function.service_config[0].uri, "")
}

output "invoker" {
  value = {
    email       = google_service_account.invoker.email
    private_key = google_service_account_key.invoker.private_key
  }

  description = "object({ email: string, private_key: string }) ||| A GCP service account with explicit privilege invoke db admin cloud function."
  sensitive   = true
}
