variable "aws_region" {
  description = "AWS region used by the provider for account controls."
  type        = string
  default     = "ap-southeast-2"
}

variable "project_name" {
  description = "Name used for persistent account-control resources."
  type        = string
  default     = "delivery-platform"
}

variable "monthly_budget_amount" {
  description = "Monthly account-wide AWS cost budget in USD."
  type        = number
  default     = 10

  validation {
    condition     = var.monthly_budget_amount > 0
    error_message = "monthly_budget_amount must be greater than zero."
  }
}

variable "billing_alert_email" {
  description = "Email address that receives AWS Budget notifications."
  type        = string

  validation {
    condition     = can(regex("^[^@]+@[^@]+\\.[^@]+$", var.billing_alert_email))
    error_message = "billing_alert_email must be a valid email address."
  }
}
