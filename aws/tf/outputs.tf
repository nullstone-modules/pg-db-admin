output "function_name" {
  value = aws_lambda_function.db_admin.function_name
}

output "function_url_id" {
  value = aws_lambda_function_url.db_admin.url_id
}

output "function_url" {
  value = aws_lambda_function_url.db_admin.function_url
}

output "invoker" {
  value = {
    name       = aws_iam_user.invoker.name
    access_key = aws_iam_access_key.invoker.id
    secret_key = aws_iam_access_key.invoker.secret
  }

  description = "object({ name: string, access_key: string, secret_key: string }) ||| An AWS User with explicit privilege to invoke db admin lambda function."

  sensitive = true
}
