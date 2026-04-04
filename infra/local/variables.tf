variable "project_name" {
  type        = string
  description = "Project identifier."
  default     = "trip-planner-api"
}

variable "aws_region" {
  type        = string
  description = "AWS region used by LocalStack."
  default     = "us-east-1"
}

variable "localstack_endpoint" {
  type        = string
  description = "LocalStack endpoint URL."
  default     = "http://localhost:4566"
}

variable "vpc_cidr" {
  type        = string
  description = "CIDR block for the VPC."
  default     = "10.2.0.0/16"
}

variable "az_count" {
  type        = number
  description = "Number of availability zones to use."
  default     = 2
}

variable "image_tag" {
  type        = string
  description = "Container image tag to deploy."
  default     = "local"
}

variable "migrations_image_tag" {
  type        = string
  description = "Migrations image tag (defaults to image_tag)."
  default     = "local-migrations"
}

variable "image_registry" {
  type        = string
  description = "Image registry host used by LocalStack."
  default     = "localhost:4566"
}

variable "ecr_repository_name" {
  type        = string
  description = "ECR repository name (defaults to <project>-local)."
  default     = ""
}

variable "database_url" {
  type        = string
  description = "Database URL for local ECS tasks."
  default     = "postgres://eb:eb@host.docker.internal:5432/eastbay?sslmode=disable"
}

variable "task_cpu" {
  type        = string
  description = "Fargate task CPU units."
  default     = "256"
}

variable "task_memory" {
  type        = string
  description = "Fargate task memory (MiB)."
  default     = "512"
}

variable "migrations_task_cpu" {
  type        = string
  description = "Fargate task CPU units for migrations."
  default     = "256"
}

variable "migrations_task_memory" {
  type        = string
  description = "Fargate task memory (MiB) for migrations."
  default     = "512"
}

variable "desired_count" {
  type        = number
  description = "Desired ECS task count."
  default     = 1
}

variable "tags" {
  type        = map(string)
  description = "Extra tags to apply."
  default     = {}
}
