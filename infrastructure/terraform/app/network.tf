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

resource "aws_security_group" "alb" {
  name        = "${local.name_prefix}-alb"
  description = "Controls public access to the delivery API load balancer"
  vpc_id      = data.aws_vpc.default.id
}

resource "aws_vpc_security_group_ingress_rule" "alb_http_from_my_ip" {
  security_group_id = aws_security_group.alb.id
  description       = "HTTP access from my public IP"

  cidr_ipv4   = var.allowed_cidr
  from_port   = 80
  to_port     = 80
  ip_protocol = "tcp"
}

resource "aws_vpc_security_group_egress_rule" "alb_to_api" {
  security_group_id            = aws_security_group.alb.id
  referenced_security_group_id = aws_security_group.api.id
  description                  = "Forward requests and health checks to the API"

  from_port   = 8080
  to_port     = 8080
  ip_protocol = "tcp"
}

resource "aws_vpc_security_group_ingress_rule" "api_from_alb" {
  security_group_id = aws_security_group.api.id
  description       = "API traffic and health checks from the load balancer"

  referenced_security_group_id = aws_security_group.alb.id
  from_port                    = 8080
  to_port                      = 8080
  ip_protocol                  = "tcp"
}

resource "aws_vpc_security_group_egress_rule" "api_outbound" {
  security_group_id = aws_security_group.api.id
  description       = "Allow the task to reach ECR, CloudWatch, and the internet"

  cidr_ipv4   = "0.0.0.0/0"
  ip_protocol = "-1"
}
