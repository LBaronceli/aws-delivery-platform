output "ecr_repository_url" {
  description = "URL used to push the API container image"
  value       = aws_ecr_repository.api.repository_url
}

output "load_balancer_dns_name" {
  description = "Public DNS name of the development Application Load Balancer"
  value       = aws_lb.api.dns_name
}

output "api_url" {
  description = "Base URL of the IP-restricted development API"
  value       = "http://${aws_lb.api.dns_name}"
}
