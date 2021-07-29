resource "aws_iam_role" "db_admin" {
  name               = var.name
  assume_role_policy = data.aws_iam_policy_document.db_admin-assume.json
  tags               = var.tags
}

data "aws_iam_policy_document" "db_admin-assume" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy_attachment" "db_admin-basic" {
  role       = aws_iam_role.db_admin.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy_attachment" "db_admin-vpc" {
  role       = aws_iam_role.db_admin.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
}

resource "aws_iam_role_policy" "db_admin" {
  role   = aws_iam_role.db_admin.id
  policy = data.aws_iam_policy_document.db_admin.json
}

data "aws_iam_policy_document" "db_admin" {
  statement {
    sid       = "AllowDbAccess"
    effect    = "Allow"
    resources = [aws_secretsmanager_secret.db_admin_pg.arn]
    actions = [
      "secretsmanager:GetSecretValue",
      "kms:Decrypt"
    ]
  }
}
