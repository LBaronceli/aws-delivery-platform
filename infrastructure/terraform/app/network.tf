data "aws_availability_zones" "available" {
  state = "available"
}

locals {
  availability_zones = slice(data.aws_availability_zones.available.names, 0, 2)

  public_subnet_cidrs = {
    for index, availability_zone in local.availability_zones :
    availability_zone => cidrsubnet(var.vpc_cidr, 8, index)
  }

  private_subnet_cidrs = {
    for index, availability_zone in local.availability_zones :
    availability_zone => cidrsubnet(var.vpc_cidr, 8, index + 10)
  }
}

resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = local.name_prefix
  }
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = local.name_prefix
  }
}

resource "aws_subnet" "public" {
  for_each = local.public_subnet_cidrs

  vpc_id                  = aws_vpc.main.id
  availability_zone       = each.key
  cidr_block              = each.value
  map_public_ip_on_launch = true

  tags = {
    Name = "${local.name_prefix}-public-${each.key}"
    Tier = "public"
  }
}

resource "aws_subnet" "private" {
  for_each = local.private_subnet_cidrs

  vpc_id                  = aws_vpc.main.id
  availability_zone       = each.key
  cidr_block              = each.value
  map_public_ip_on_launch = false

  tags = {
    Name = "${local.name_prefix}-private-${each.key}"
    Tier = "private"
  }
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "${local.name_prefix}-public"
  }
}

resource "aws_route" "public_internet" {
  route_table_id         = aws_route_table.public.id
  destination_cidr_block = "0.0.0.0/0"
  gateway_id             = aws_internet_gateway.main.id
}

resource "aws_route_table_association" "public" {
  for_each = aws_subnet.public

  subnet_id      = each.value.id
  route_table_id = aws_route_table.public.id
}

resource "aws_eip" "nat" {
  domain = "vpc"

  tags = {
    Name = "${local.name_prefix}-nat"
  }

  depends_on = [aws_internet_gateway.main]
}

resource "aws_nat_gateway" "main" {
  allocation_id = aws_eip.nat.id
  subnet_id     = aws_subnet.public[local.availability_zones[0]].id

  tags = {
    Name = local.name_prefix
  }

  depends_on = [aws_internet_gateway.main]
}

resource "aws_route_table" "private" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "${local.name_prefix}-private"
  }
}

resource "aws_route" "private_internet" {
  route_table_id         = aws_route_table.private.id
  destination_cidr_block = "0.0.0.0/0"
  nat_gateway_id         = aws_nat_gateway.main.id
}

resource "aws_route_table_association" "private" {
  for_each = aws_subnet.private

  subnet_id      = each.value.id
  route_table_id = aws_route_table.private.id
}

resource "aws_security_group" "api" {
  name        = "${local.name_prefix}-api"
  description = "Controls access to the delivery API"
  vpc_id      = aws_vpc.main.id
}

resource "aws_security_group" "alb" {
  name        = "${local.name_prefix}-alb"
  description = "Controls public access to the delivery API load balancer"
  vpc_id      = aws_vpc.main.id
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
  security_group_id            = aws_security_group.api.id
  referenced_security_group_id = aws_security_group.alb.id
  description                  = "API traffic and health checks from the load balancer"

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
