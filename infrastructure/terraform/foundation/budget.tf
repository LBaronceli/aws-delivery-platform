resource "aws_budgets_budget" "monthly_account_cost" {
  name         = "${var.project_name}-account-monthly-cost"
  budget_type  = "COST"
  limit_amount = tostring(var.monthly_budget_amount)
  limit_unit   = "USD"
  time_unit    = "MONTHLY"

  # Track gross usage so credits and refunds do not hide the underlying spend.
  cost_types {
    include_credit = false
    include_refund = false
  }

  dynamic "notification" {
    for_each = toset([30, 50, 80, 100])

    content {
      comparison_operator        = "GREATER_THAN"
      threshold                  = notification.value
      threshold_type             = "PERCENTAGE"
      notification_type          = "ACTUAL"
      subscriber_email_addresses = [var.billing_alert_email]
    }
  }

  notification {
    comparison_operator        = "GREATER_THAN"
    threshold                  = 80
    threshold_type             = "PERCENTAGE"
    notification_type          = "FORECASTED"
    subscriber_email_addresses = [var.billing_alert_email]
  }

  lifecycle {
    prevent_destroy = true
  }
}
