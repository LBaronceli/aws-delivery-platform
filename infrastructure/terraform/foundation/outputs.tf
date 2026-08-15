output "monthly_budget_name" {
  description = "Name of the persistent account-wide monthly AWS cost budget."
  value       = aws_budgets_budget.monthly_account_cost.name
}
