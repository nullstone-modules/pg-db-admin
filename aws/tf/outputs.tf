output "function_name" {
  value = aws_lambda_function.db_admin.function_name
}

output "function_url_id" {
  value = aws_lambda_function_url.db_admin.url_id
}

output "function_url" {
  value = aws_lambda_function_url.db_admin.function_url
}
