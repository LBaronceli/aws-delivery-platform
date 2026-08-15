output "ecr_repository_url" {
  description = "URL used to push the API container image"
  value       = aws_ecr_repository.api.repository_url
}

output "monthly_budget_name" {
  description = "Name of the account-wide monthly AWS cost budget"
  value       = aws_budgets_budget.monthly_account_cost.name
}
