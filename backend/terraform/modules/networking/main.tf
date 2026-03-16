# VPC
resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true
  
  tags = merge(var.common_tags, {
    Name = "${var.project_name}-${var.environment}-vpc"
  })
}

# Internet Gateway
resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id
  
  tags = merge(var.common_tags, {
    Name = "${var.project_name}-${var.environment}-igw"
  })
}

# Data source for availability zones
data "aws_availability_zones" "available" {
  state = "available"
}

# Public subnets for NAT gateways
resource "aws_subnet" "public" {
  count = length(var.availability_zones)
  
  vpc_id                  = aws_vpc.main.id
  cidr_block              = cidrsubnet(var.vpc_cidr, 8, count.index)
  availability_zone       = var.availability_zones[count.index]
  map_public_ip_on_launch = true
  
  tags = merge(var.common_tags, {
    Name = "${var.project_name}-${var.environment}-public-${count.index + 1}"
    Type = "Public"
  })
}

# Private subnets for Lambda functions (application layer)
resource "aws_subnet" "app" {
  count = length(var.availability_zones)
  
  vpc_id            = aws_vpc.main.id
  cidr_block        = cidrsubnet(var.vpc_cidr, 8, count.index + 10)
  availability_zone = var.availability_zones[count.index]
  
  tags = merge(var.common_tags, {
    Name = "${var.project_name}-${var.environment}-app-${count.index + 1}"
    Type = "Application"
  })
}

# Private subnets for RDS (database layer)
resource "aws_subnet" "database" {
  count = length(var.availability_zones)
  
  vpc_id            = aws_vpc.main.id
  cidr_block        = cidrsubnet(var.vpc_cidr, 8, count.index + 20)
  availability_zone = var.availability_zones[count.index]
  
  tags = merge(var.common_tags, {
    Name = "${var.project_name}-${var.environment}-db-${count.index + 1}"
    Type = "Database"
  })
}

# Elastic IPs for NAT gateways - COMMENTED OUT TO SAVE COSTS
# Lambdas now use public subnets directly
# resource "aws_eip" "nat" {
#   count = length(var.availability_zones)
#   
#   domain = "vpc"
#   
#   tags = merge(var.common_tags, {
#     Name = "${var.project_name}-${var.environment}-nat-eip-${count.index + 1}"
#   })
#   
#   depends_on = [aws_internet_gateway.main]
# }

# NAT gateways - COMMENTED OUT TO SAVE COSTS
# Lambdas now use public subnets directly
# resource "aws_nat_gateway" "main" {
#   count = length(var.availability_zones)
#   
#   allocation_id = aws_eip.nat[count.index].id
#   subnet_id     = aws_subnet.public[count.index].id
#   
#   tags = merge(var.common_tags, {
#     Name = "${var.project_name}-${var.environment}-nat-${count.index + 1}"
#   })
#   
#   depends_on = [aws_internet_gateway.main]
# }

# Route table for public subnets
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id
  
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }
  
  tags = merge(var.common_tags, {
    Name = "${var.project_name}-${var.environment}-public-rt"
  })
}

# Route table associations for public subnets
resource "aws_route_table_association" "public" {
  count = length(var.availability_zones)
  
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

# Route tables for private subnets (application layer) - UPDATED FOR COST SAVINGS
# Lambdas now use public subnets, but keeping these for other services if needed
resource "aws_route_table" "app" {
  count = length(var.availability_zones)
  
  vpc_id = aws_vpc.main.id
  
  # No internet route needed since Lambdas moved to public subnets
  # route {
  #   cidr_block     = "0.0.0.0/0"
  #   nat_gateway_id = aws_nat_gateway.main[count.index].id
  # }
  
  tags = merge(var.common_tags, {
    Name = "${var.project_name}-${var.environment}-app-rt-${count.index + 1}"
  })
}

# Route table associations for application subnets
resource "aws_route_table_association" "app" {
  count = length(var.availability_zones)
  
  subnet_id      = aws_subnet.app[count.index].id
  route_table_id = aws_route_table.app[count.index].id
}

# Route tables for database subnets (isolated - no internet access)
resource "aws_route_table" "database" {
  count = length(var.availability_zones)
  
  vpc_id = aws_vpc.main.id
  
  tags = merge(var.common_tags, {
    Name = "${var.project_name}-${var.environment}-db-rt-${count.index + 1}"
  })
}

# Route table associations for database subnets
resource "aws_route_table_association" "database" {
  count = length(var.availability_zones)
  
  subnet_id      = aws_subnet.database[count.index].id
  route_table_id = aws_route_table.database[count.index].id
}

# Security group for Lambda functions
resource "aws_security_group" "app" {
  name_prefix = "${var.project_name}-${var.environment}-app-"
  description = "Security group for Lambda functions"
  vpc_id      = aws_vpc.main.id
  
  # Outbound rules
  egress {
    description = "HTTPS outbound"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  egress {
    description = "HTTP outbound"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  egress {
    description = "PostgreSQL outbound"
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = [var.vpc_cidr]
  }
  
  egress {
    description = "CockroachDB outbound"
    from_port   = 26257
    to_port     = 26257
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  tags = merge(var.common_tags, {
    Name = "${var.project_name}-${var.environment}-app-sg"
  })
  
  lifecycle {
    create_before_destroy = true
  }
}

# Security group for RDS
resource "aws_security_group" "database" {
  name_prefix = "${var.project_name}-${var.environment}-db-"
  description = "Security group for RDS instance"
  vpc_id      = aws_vpc.main.id
  
  # Inbound rule for PostgreSQL from application layer
  ingress {
    description     = "PostgreSQL from application layer"
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.app.id]
  }
  
  tags = merge(var.common_tags, {
    Name = "${var.project_name}-${var.environment}-db-sg"
  })
  
  lifecycle {
    create_before_destroy = true
  }
}

# VPC Endpoints for AWS services (to reduce NAT Gateway costs)
resource "aws_vpc_endpoint" "s3" {
  vpc_id       = aws_vpc.main.id
  service_name = "com.amazonaws.${data.aws_region.current.name}.s3"
  
  tags = merge(var.common_tags, {
    Name = "${var.project_name}-${var.environment}-s3-endpoint"
  })
}

resource "aws_vpc_endpoint" "dynamodb" {
  vpc_id       = aws_vpc.main.id
  service_name = "com.amazonaws.${data.aws_region.current.name}.dynamodb"
  
  tags = merge(var.common_tags, {
    Name = "${var.project_name}-${var.environment}-dynamodb-endpoint"
  })
}

# Data source for current region
data "aws_region" "current" {} 