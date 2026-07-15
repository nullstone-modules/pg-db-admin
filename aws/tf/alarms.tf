locals {
  alarm_functions = var.alerts.enabled ? {
    db_admin       = aws_lambda_function.db_admin.function_name
    db_admin_setup = aws_lambda_function.db_admin_setup.function_name
  } : {}

  alarm_actions = var.alerts.notification_arn != "" ? [var.alerts.notification_arn] : []
}

resource "aws_cloudwatch_metric_alarm" "error_rate" {
  for_each = local.alarm_functions

  alarm_name          = "${each.value}/error-rate"
  alarm_description   = "Lambda error rate >= ${var.alerts.error_rate}% on ${each.value}"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = 1
  threshold           = var.alerts.error_rate
  treat_missing_data  = "notBreaching"
  tags                = var.tags

  metric_query {
    id          = "error_rate"
    expression  = "100 * errors / invocations"
    label       = "Error Rate (%)"
    return_data = true
  }

  metric_query {
    id = "errors"

    metric {
      metric_name = "Errors"
      namespace   = "AWS/Lambda"
      period      = 300
      stat        = "Sum"
      dimensions  = { FunctionName = each.value }
    }
  }

  metric_query {
    id = "invocations"

    metric {
      metric_name = "Invocations"
      namespace   = "AWS/Lambda"
      period      = 300
      stat        = "Sum"
      dimensions  = { FunctionName = each.value }
    }
  }

  alarm_actions = local.alarm_actions
  ok_actions    = local.alarm_actions
}
