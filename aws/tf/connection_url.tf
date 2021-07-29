resource "aws_secretsmanager_secret" "db_admin_pg" {
  name = "${var.name}/conn_url"
  tags = var.tags
}

resource "aws_secretsmanager_secret_version" "db_admin_pg" {
  secret_id     = aws_secretsmanager_secret.db_admin_pg.id
  secret_string = "postgres://${urlencode(var.username)}:${urlencode(var.password)}@${var.host}:${var.port}/${urlencode(var.database)}"
}
