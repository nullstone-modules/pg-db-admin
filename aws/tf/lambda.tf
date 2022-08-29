resource "aws_lambda_function" "db_admin" {
  function_name    = var.name
  tags             = var.tags
  role             = aws_iam_role.db_admin.arn
  runtime          = "go1.x"
  handler          = "pg-db-admin"
  filename         = "${path.module}/files/pg-db-admin.zip"
  source_code_hash = filebase64sha256("${path.module}/files/pg-db-admin.zip")
  // This can take ~5s to create a db sometimes
  timeout = 10

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

resource "aws_lambda_function_url" "db_admin" {
  function_name      = aws_lambda_function.db_admin.function_name
  authorization_type = "AWS_IAM"
}

// Allow invoker to invoke function url
// See https://docs.aws.amazon.com/lambda/latest/dg/urls-auth.html
// TODO: This fails if the invoker isn't fully created yet; how can we make it work?
resource "aws_lambda_permission" "db_admin_invoke" {
  statement_id_prefix    = "AllowDbAdminInvoke"
  function_name          = aws_lambda_function.db_admin.function_name
  action                 = "lambda:InvokeFunctionUrl"
  principal              = aws_iam_user.invoker.arn
  function_url_auth_type = "AWS_IAM"
}
