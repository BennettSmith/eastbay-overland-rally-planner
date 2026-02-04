variable "project_name" {
  type        = string
  description = "Project identifier."
  default     = "trip-planner-api"
}

variable "aws_region" {
  type        = string
  description = "AWS region."
  default     = "us-west-2"
}

variable "vpc_cidr" {
  type        = string
  description = "CIDR block for the VPC."
  default     = "10.1.0.0/16"
}

variable "az_count" {
  type        = number
  description = "Number of availability zones to use."
  default     = 2
}

variable "image_tag" {
  type        = string
  description = "Container image tag to deploy."
  default     = "latest"
}

variable "migrations_image_tag" {
  type        = string
  description = "Migrations image tag (defaults to image_tag)."
  default     = ""
}

variable "desired_count" {
  type        = number
  description = "Desired ECS task count."
  default     = 2
}

variable "jwt_issuer" {
  type        = string
  description = "JWT issuer."
}

variable "jwt_audience" {
  type        = string
  description = "JWT audience."
}

variable "jwt_jwks_url" {
  type        = string
  description = "JWKS URL."
}

variable "public_base_url" {
  type        = string
  description = "Public base URL (optional)."
  default     = ""
}

variable "trust_proxy_headers" {
  type        = string
  description = "Whether to trust proxy headers."
  default     = "true"
}

variable "domain_name" {
  type        = string
  description = "Domain name for the ALB (optional)."
  default     = "api.overlandeastbay.com"
}

variable "subject_alternative_names" {
  type        = list(string)
  description = "Subject alternative names for ACM."
  default     = []
}

variable "create_acm_certificate" {
  type        = bool
  description = "Create ACM certificate in this root."
  default     = false
}

variable "certificate_arn" {
  type        = string
  description = "Existing ACM certificate ARN to use (optional)."
  default     = ""
}

variable "alb_ingress_cidr_blocks" {
  type        = list(string)
  description = "CIDR blocks allowed to reach the ALB."
  default     = ["0.0.0.0/0"]
}

variable "log_retention_days" {
  type        = number
  description = "Log retention in days."
  default     = 60
}

variable "db_instance_class" {
  type        = string
  description = "RDS instance class."
  default     = "db.t3.small"
}

variable "db_multi_az" {
  type        = bool
  description = "Enable Multi-AZ for RDS."
  default     = true
}

variable "db_backup_retention_days" {
  type        = number
  description = "RDS backup retention days."
  default     = 14
}

variable "db_deletion_protection" {
  type        = bool
  description = "Enable deletion protection."
  default     = true
}

variable "db_skip_final_snapshot" {
  type        = bool
  description = "Skip final snapshot on destroy."
  default     = false
}

variable "db_final_snapshot_identifier" {
  type        = string
  description = "Final snapshot identifier when skip_final_snapshot is false."
  default     = "trip-planner-api-prod-final"
}

variable "secrets_kms_key_arn" {
  type        = string
  description = "Optional KMS key ARN for Secrets Manager."
  default     = ""
}

variable "tags" {
  type        = map(string)
  description = "Extra tags to apply."
  default     = {}
}
