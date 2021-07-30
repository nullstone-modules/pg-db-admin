data "archive_file" "db_admin" {
  type             = "zip"
  source_file      = "${path.module}/files/pg-db-admin"
  output_file_mode = "0755"
  output_path      = "${path.module}/files/pg-db-admin.zip"
}

resource "aws_lambda_function" "db_admin" {
  function_name    = var.name
  tags             = var.tags
  role             = aws_iam_role.db_admin.arn
  package_type     = "Zip"
  runtime          = "go1.x"
  filename         = data.archive_file.db_admin.output_path
  source_code_hash = data.archive_file.db_admin.output_base64sha256

  environment {
    variables = {
      DB_CONN_URL_SECRET_ID = aws_secretsmanager_secret.db_admin_pg.id
    }
  }

  vpc_config {
    security_group_ids = concat([aws_security_group.db_admin.id], var.network.security_group_ids)
    subnet_ids         = var.network.subnet_ids
  }
}
