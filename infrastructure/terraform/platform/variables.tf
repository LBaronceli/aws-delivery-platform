variable "aws_region" {
  description = "AWS region used for persistent platform resources."
  type        = string
  default     = "ap-southeast-2"
}

variable "project_name" {
  description = "Name of the project."
  type        = string
  default     = "delivery-platform"
}

variable "environment" {
  description = "Deployment environment served by these platform resources."
  type        = string
  default     = "dev"
}
