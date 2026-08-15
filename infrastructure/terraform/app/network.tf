data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

resource "aws_security_group" "api" {
  name        = "${local.name_prefix}-api"
  description = "Controls access to the delivery API"
  vpc_id      = data.aws_vpc.default.id
}

resource "aws_vpc_security_group_ingress_rule" "api_from_my_ip" {
  security_group_id = aws_security_group.api.id
  description       = "API access from my public IP"

  cidr_ipv4   = var.allowed_cidr
  from_port   = 8080
  to_port     = 8080
  ip_protocol = "tcp"
}

resource "aws_vpc_security_group_egress_rule" "api_outbound" {
  security_group_id = aws_security_group.api.id
  description       = "Allow the task to reach ECR, CloudWatch, and the internet"

  cidr_ipv4   = "0.0.0.0/0"
  ip_protocol = "-1"
}
