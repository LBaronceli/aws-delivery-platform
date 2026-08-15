variable "aws_region" {
  description = "AWS region used for application resources"
  type        = string
  default     = "ap-southeast-2"
}

variable "project_name" {
  description = "Name of the project"
  type        = string
  default     = "delivery-platform"
}

variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "dev"
}

variable "allowed_cidr" {
  description = "Public IP allowed to access the API, expressed as a /32 CIDR"
  type        = string
}

variable "image_uri" {
  description = "Complete ECR image URI, including its tag"
  type        = string
}

variable "ecs_desired_count" {
  description = "Number of API tasks to run. Use 0 when the development service is not needed."
  type        = number
  default     = 0

  validation {
    condition     = contains([0, 1], var.ecs_desired_count)
    error_message = "ecs_desired_count must be 0 or 1 for this cost-controlled development environment."
  }
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
