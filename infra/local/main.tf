provider "aws" {
  region                      = var.aws_region
  access_key                  = "test"
  secret_key                  = "test"
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true

  endpoints {
    cloudwatch     = var.localstack_endpoint
    cloudwatchlogs = var.localstack_endpoint
    ec2            = var.localstack_endpoint
    ecr            = var.localstack_endpoint
    ecs            = var.localstack_endpoint
    iam            = var.localstack_endpoint
    logs           = var.localstack_endpoint
    secretsmanager = var.localstack_endpoint
    sts            = var.localstack_endpoint
  }
}

locals {
  name_prefix      = "${var.project_name}-local"
  repository_name  = var.ecr_repository_name != "" ? var.ecr_repository_name : "${var.project_name}-local"
  app_image        = "${var.image_registry}/${local.repository_name}:${var.image_tag}"
  migrations_image = "${var.image_registry}/${local.repository_name}:${var.migrations_image_tag}"

  private_subnet_cidrs = [
    for idx in range(var.az_count) : cidrsubnet(var.vpc_cidr, 8, idx)
  ]
}

resource "aws_vpc" "local" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = merge(var.tags, {
    Name = "${local.name_prefix}-vpc"
  })
}

resource "aws_subnet" "private" {
  count             = var.az_count
  vpc_id            = aws_vpc.local.id
  cidr_block        = local.private_subnet_cidrs[count.index]
  availability_zone = "${var.aws_region}${count.index == 0 ? "a" : "b"}"

  tags = merge(var.tags, {
    Name = "${local.name_prefix}-subnet-${count.index + 1}"
  })
}

resource "aws_security_group" "ecs" {
  name        = "${local.name_prefix}-ecs"
  description = "LocalStack ECS task ingress"
  vpc_id      = aws_vpc.local.id

  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(var.tags, {
    Name = "${local.name_prefix}-ecs-sg"
  })
}

resource "aws_ecr_repository" "app" {
  name                 = local.repository_name
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }
}

resource "aws_cloudwatch_log_group" "app" {
  name              = "/ecs/${local.name_prefix}"
  retention_in_days = 7
}

resource "aws_cloudwatch_log_group" "migrations" {
  name              = "/ecs/${local.name_prefix}-migrations"
  retention_in_days = 7
}

resource "aws_secretsmanager_secret" "db" {
  name = "/${var.project_name}/local/database"
}

resource "aws_secretsmanager_secret_version" "db" {
  secret_id = aws_secretsmanager_secret.db.id
  secret_string = jsonencode({
    database_url = var.database_url
  })
}

data "aws_iam_policy_document" "ecs_task_assume" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "task_execution" {
  name               = "${local.name_prefix}-task-exec"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume.json
}

resource "aws_iam_role" "task" {
  name               = "${local.name_prefix}-task"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume.json
}

data "aws_iam_policy_document" "task_execution" {
  statement {
    effect    = "Allow"
    actions   = ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"]
    resources = ["*"]
  }

  statement {
    effect    = "Allow"
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [aws_secretsmanager_secret.db.arn]
  }
}

resource "aws_iam_role_policy" "task_execution" {
  role   = aws_iam_role.task_execution.id
  policy = data.aws_iam_policy_document.task_execution.json
}

resource "aws_ecs_cluster" "app" {
  name = local.name_prefix
}

resource "aws_ecs_task_definition" "app" {
  family                   = local.name_prefix
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.task_cpu
  memory                   = var.task_memory
  execution_role_arn       = aws_iam_role.task_execution.arn
  task_role_arn            = aws_iam_role.task.arn

  container_definitions = jsonencode([
    {
      name      = "app"
      image     = local.app_image
      essential = true
      portMappings = [
        {
          containerPort = 8080
          hostPort      = 8080
          protocol      = "tcp"
        }
      ]
      environment = [
        { name = "PORT", value = "8080" },
        { name = "STORAGE_BACKEND", value = "postgres" },
        { name = "DATABASE_URL", value = var.database_url },
        { name = "AUTH_MODE", value = "dev" },
        { name = "DEV_SUBJECT", value = "dev|local" },
        { name = "DEV_ISSUER", value = "dev" },
        { name = "TRUST_PROXY_HEADERS", value = "true" },
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.app.name
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "ecs"
        }
      }
    }
  ])
}

resource "aws_ecs_task_definition" "migrations" {
  family                   = "${local.name_prefix}-migrations"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.migrations_task_cpu
  memory                   = var.migrations_task_memory
  execution_role_arn       = aws_iam_role.task_execution.arn
  task_role_arn            = aws_iam_role.task.arn

  container_definitions = jsonencode([
    {
      name      = "migrations"
      image     = local.migrations_image
      essential = true
      environment = [
        { name = "DATABASE_URL", value = var.database_url },
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.migrations.name
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "ecs"
        }
      }
    }
  ])
}

resource "aws_ecs_service" "app" {
  name            = local.name_prefix
  cluster         = aws_ecs_cluster.app.id
  task_definition = aws_ecs_task_definition.app.arn
  desired_count   = var.desired_count
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = aws_subnet.private[*].id
    security_groups  = [aws_security_group.ecs.id]
    assign_public_ip = true
  }
}
