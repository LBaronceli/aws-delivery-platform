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