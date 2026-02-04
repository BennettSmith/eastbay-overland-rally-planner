output "vpc_id" {
  description = "VPC ID."
  value       = aws_vpc.main.id
}

output "public_subnet_ids" {
  description = "Public subnet IDs."
  value       = aws_subnet.public[*].id
}

output "private_subnet_ids" {
  description = "Private subnet IDs."
  value       = aws_subnet.private[*].id
}

output "ecs_security_group_id" {
  description = "Security group ID for ECS tasks."
  value       = aws_security_group.ecs.id
}

output "alb_dns_name" {
  description = "ALB DNS name."
  value       = aws_lb.app.dns_name
}

output "alb_zone_id" {
  description = "ALB hosted zone ID."
  value       = aws_lb.app.zone_id
}

output "ecs_cluster_name" {
  description = "ECS cluster name."
  value       = aws_ecs_cluster.app.name
}

output "ecs_service_name" {
  description = "ECS service name."
  value       = aws_ecs_service.app.name
}

output "migrations_task_definition_arn" {
  description = "ECS task definition ARN for migrations."
  value       = aws_ecs_task_definition.migrations.arn
}

output "ecr_repository_url" {
  description = "ECR repository URL."
  value       = aws_ecr_repository.app.repository_url
}

output "db_instance_endpoint" {
  description = "RDS instance endpoint."
  value       = aws_db_instance.postgres.address
}

output "db_secret_arn" {
  description = "Secrets Manager ARN with database credentials."
  value       = aws_secretsmanager_secret.db_credentials.arn
}

output "acm_dns_validation_records" {
  description = "DNS validation records for ACM certificate."
  value       = var.create_acm_certificate ? aws_acm_certificate.app[0].domain_validation_options : []
}

output "certificate_arn" {
  description = "ACM certificate ARN in use."
  value       = local.tls_certificate_arn
}
