resource "aws_secretsmanager_secret" "db_admin_pg" {
  name                    = "${var.name}/conn_url"
  tags                    = var.tags
  recovery_window_in_days = var.is_prod_env ? 7 : 0
}

resource "aws_secretsmanager_secret_version" "db_admin_pg" {
  secret_id     = aws_secretsmanager_secret.db_admin_pg.id
  secret_string = "postgres://${urlencode(var.username)}:${urlencode(var.password)}@${var.host}:${var.port}/${urlencode(var.database)}"
}

// admin_role_conn_url secret value will be configured inside the db_admin_setup lambda invocation
resource "aws_secretsmanager_secret" "admin_role_conn_url" {
  name = "${var.name}-setup"
  tags = var.tags
}
