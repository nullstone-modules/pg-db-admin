resource "aws_iam_user" "invoker" {
  name = "${var.name}-invoker"
  tags = var.tags
}

resource "aws_iam_access_key" "invoker" {
  user = aws_iam_user.invoker.name
}

resource "aws_iam_user_policy" "invoker" {
  user = aws_iam_user.invoker.name
  policy = data.aws_iam_policy_document.invoker.json
}

data "aws_iam_policy_document" "invoker" {
  statement {
    sid       = "AllowFunctionUrlInvoke"
    effect    = "Allow"
    actions   = ["lambda:InvokeFunctionUrl"]
    resources = [aws_lambda_function.db_admin.arn]

    condition {
      variable = "lambda:FunctionUrlAuthType"
      test     = "StringEquals"
      values   = ["AWS_IAM"]
    }
  }
}
