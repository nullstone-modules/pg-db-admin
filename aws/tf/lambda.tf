resource "aws_lambda_function" "db_admin" {
  function_name    = var.name
  tags             = var.tags
  role             = aws_iam_role.db_admin.arn
  runtime          = "provided.al2023"
  handler          = "bootstrap"
  filename         = "${path.module}/files/pg-db-admin.zip"
  source_code_hash = filebase64sha256("${path.module}/files/pg-db-admin.zip")
  // This can take ~5s to create a db sometimes
  timeout = 10

  environment {
    variables = {
      DB_ADMIN_CONN_URL_SECRET_ID = aws_secretsmanager_secret.admin_role_conn_url.id

      // RESET_FUNCTION does 2 things:
      // 1. Waits for lambda invocation of initial setup
      // 2. Any time initial setup is run, forces the lambda to reset to force an update of the connection url
      RESET_FUNCTION = sha1(aws_lambda_invocation.db_admin_setup.result)
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

// NOTE: This resource ensures that the invoker user is properly created
//  IAM is eventually consistent and the aws_lambda_permission fails because the user is not "ready" yet
resource "time_sleep" "wait_for_invoker" {
  create_duration = "5s"

  triggers = {
    invoker_arn = aws_iam_user.invoker.arn
  }
}

// Allow invoker to invoke function url
// See https://docs.aws.amazon.com/lambda/latest/dg/urls-auth.html
resource "aws_lambda_permission" "db_admin_invoke" {
  statement_id_prefix    = "AllowDbAdminInvoke"
  function_name          = aws_lambda_function.db_admin.function_name
  action                 = "lambda:InvokeFunctionUrl"
  principal              = time_sleep.wait_for_invoker.triggers["invoker_arn"]
  function_url_auth_type = "AWS_IAM"
}
