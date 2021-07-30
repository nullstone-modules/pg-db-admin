resource "aws_security_group" "db_admin" {
  name   = "db-admin/${var.name}"
  tags   = var.tags
  vpc_id = var.network.vpc_id
}

resource "aws_security_group_rule" "this-to-world-https" {
  security_group_id = aws_security_group.db_admin.id
  protocol          = "tcp"
  type              = "egress"
  from_port         = 443
  to_port           = 443
  cidr_blocks       = ["0.0.0.0/0"]
}
