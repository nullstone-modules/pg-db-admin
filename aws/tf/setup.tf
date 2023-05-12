resource "aws_lambda_function" "db_admin_setup" {
  function_name    = var.name
  tags             = var.tags
  role             = aws_iam_role.db_admin.arn
  runtime          = "go1.x"
  handler          = "pg-db-admin"
  filename         = "${path.module}/files/pg-db-admin.zip"
  source_code_hash = filebase64sha256("${path.module}/files/pg-db-admin.zip")

  environment {
    variables = {
      DB_SETUP_CONN_URL_SECRET_ID = aws_secretsmanager_secret.db_admin_pg.id
      DB_ADMIN_CONN_URL_SECRET_ID = aws_secretsmanager_secret.admin_role_conn_url.id
    }
  }

  vpc_config {
    security_group_ids = concat([aws_security_group.db_admin.id], var.network.security_group_ids)
    subnet_ids         = var.network.subnet_ids
  }
}

resource "aws_lambda_invocation" "db_admin_setup" {
  function_name = aws_lambda_function.db_admin_setup.function_name
  input         = jsonencode({ setup : true })
}
