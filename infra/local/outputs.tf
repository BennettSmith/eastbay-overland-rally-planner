output "ecs_cluster_name" {
  description = "ECS cluster name."
  value       = aws_ecs_cluster.app.name
}

output "ecs_service_name" {
  description = "ECS service name."
  value       = aws_ecs_service.app.name
}

output "migrations_task_definition_arn" {
  description = "Migrations task definition ARN."
  value       = aws_ecs_task_definition.migrations.arn
}

output "private_subnet_ids_csv" {
  description = "CSV list of private subnet IDs."
  value       = join(",", aws_subnet.private[*].id)
}

output "ecs_task_security_group_id" {
  description = "Security group ID for ECS tasks."
  value       = aws_security_group.ecs.id
}
