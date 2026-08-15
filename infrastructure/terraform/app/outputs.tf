output "load_balancer_dns_name" {
  description = "Public DNS name of the development Application Load Balancer"
  value       = aws_lb.api.dns_name
}

output "api_url" {
  description = "Base URL of the IP-restricted development API"
  value       = "http://${aws_lb.api.dns_name}"
}

output "vpc_id" {
  description = "ID of the application VPC"
  value       = aws_vpc.main.id
}

output "public_subnet_ids" {
  description = "Public subnet IDs used by the Application Load Balancer"
  value       = [for subnet in aws_subnet.public : subnet.id]
}

output "private_subnet_ids" {
  description = "Private subnet IDs used by ECS tasks"
  value       = [for subnet in aws_subnet.private : subnet.id]
}

output "nat_gateway_public_ip" {
  description = "Public egress IP used by ECS tasks through the NAT Gateway"
  value       = aws_eip.nat.public_ip
}
