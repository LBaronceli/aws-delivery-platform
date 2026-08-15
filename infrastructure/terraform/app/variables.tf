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

variable "vpc_cidr" {
  description = "IPv4 CIDR block for the application VPC."
  type        = string
  default     = "10.0.0.0/16"

  validation {
    condition     = can(cidrnetmask(var.vpc_cidr))
    error_message = "vpc_cidr must be a valid IPv4 CIDR block."
  }
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
  description = "Number of API tasks to run when the disposable application stack is deployed."
  type        = number
  default     = 1

  validation {
    condition     = contains([0, 1], var.ecs_desired_count)
    error_message = "ecs_desired_count must be 0 or 1 for this development environment."
  }
}
