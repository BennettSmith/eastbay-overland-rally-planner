output "alb_dns_name" {
  description = "ALB DNS name."
  value       = module.trip_planner_api.alb_dns_name
}

output "alb_zone_id" {
  description = "ALB zone ID."
  value       = module.trip_planner_api.alb_zone_id
}

output "ecs_cluster_name" {
  description = "ECS cluster name."
  value       = module.trip_planner_api.ecs_cluster_name
}

output "migrations_task_definition_arn" {
  description = "Migrations task definition ARN."
  value       = module.trip_planner_api.migrations_task_definition_arn
}

output "ecs_service_name" {
  description = "ECS service name."
  value       = module.trip_planner_api.ecs_service_name
}

output "ecs_task_security_group_id" {
  description = "Security group ID for ECS tasks."
  value       = module.trip_planner_api.ecs_security_group_id
}

output "private_subnet_ids_csv" {
  description = "CSV list of private subnet IDs."
  value       = join(",", module.trip_planner_api.private_subnet_ids)
}

output "ecr_repository_url" {
  description = "ECR repository URL."
  value       = module.trip_planner_api.ecr_repository_url
}

output "db_secret_arn" {
  description = "Database secret ARN."
  value       = module.trip_planner_api.db_secret_arn
}

output "acm_dns_validation_records" {
  description = "ACM DNS validation records."
  value       = module.trip_planner_api.acm_dns_validation_records
}
