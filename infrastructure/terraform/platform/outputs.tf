output "ecr_repository_url" {
  description = "URL used to push the API container image."
  value       = aws_ecr_repository.api.repository_url
}
