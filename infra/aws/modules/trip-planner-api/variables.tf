variable "project_name" {
  type        = string
  description = "Short project identifier used in resource names."
}

variable "environment" {
  type        = string
  description = "Deployment environment name (e.g., staging, prod)."
}

variable "vpc_cidr" {
  type        = string
  description = "CIDR block for the VPC."
  default     = "10.0.0.0/16"
}

variable "az_count" {
  type        = number
  description = "Number of availability zones to use."
  default     = 2
}

variable "public_subnet_cidrs" {
  type        = list(string)
  description = "Optional explicit CIDRs for public subnets."
  default     = []
}

variable "private_subnet_cidrs" {
  type        = list(string)
  description = "Optional explicit CIDRs for private subnets."
  default     = []
}

variable "alb_ingress_cidr_blocks" {
  type        = list(string)
  description = "CIDR blocks allowed to reach the ALB."
  default     = ["0.0.0.0/0"]
}

variable "container_port" {
  type        = number
  description = "Container port exposed by the API."
  default     = 8080
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

variable "desired_count" {
  type        = number
  description = "Desired number of ECS tasks."
  default     = 2
}

variable "image_tag" {
  type        = string
  description = "Container image tag to deploy."
  default     = "latest"
}

variable "health_check_path" {
  type        = string
  description = "Health check path for the target group."
  default     = "/healthz"
}

variable "log_retention_days" {
  type        = number
  description = "CloudWatch log group retention in days."
  default     = 30
}

variable "enable_execute_command" {
  type        = bool
  description = "Enable ECS execute command support."
  default     = true
}

variable "enable_container_insights" {
  type        = bool
  description = "Enable ECS container insights."
  default     = true
}

variable "storage_backend" {
  type        = string
  description = "Storage backend for the API."
  default     = "postgres"
}

variable "auth_mode" {
  type        = string
  description = "Authentication mode (jwt or dev)."
  default     = "jwt"
}

variable "jwt_issuer" {
  type        = string
  description = "JWT issuer for auth verification."
  default     = ""

  validation {
    condition     = var.auth_mode != "jwt" || var.jwt_issuer != ""
    error_message = "jwt_issuer is required when auth_mode is jwt."
  }
}

variable "jwt_audience" {
  type        = string
  description = "JWT audience for auth verification."
  default     = ""

  validation {
    condition     = var.auth_mode != "jwt" || var.jwt_audience != ""
    error_message = "jwt_audience is required when auth_mode is jwt."
  }
}

variable "jwt_jwks_url" {
  type        = string
  description = "JWT JWKS URL for auth verification."
  default     = ""

  validation {
    condition     = var.auth_mode != "jwt" || var.jwt_jwks_url != ""
    error_message = "jwt_jwks_url is required when auth_mode is jwt."
  }
}

variable "public_base_url" {
  type        = string
  description = "Public base URL for the API (optional)."
  default     = ""
}

variable "trust_proxy_headers" {
  type        = string
  description = "Whether to trust proxy headers."
  default     = "true"
}

variable "db_name" {
  type        = string
  description = "Database name."
  default     = "tripplanner"
}

variable "db_username" {
  type        = string
  description = "Database master username."
  default     = "ebo"
}

variable "db_port" {
  type        = number
  description = "Database port."
  default     = 5432
}

variable "db_instance_class" {
  type        = string
  description = "RDS instance class."
  default     = "db.t3.micro"
}

variable "db_engine_version" {
  type        = string
  description = "Postgres engine version."
  default     = "15.5"
}

variable "db_allocated_storage" {
  type        = number
  description = "Allocated storage (GiB)."
  default     = 20
}

variable "db_max_allocated_storage" {
  type        = number
  description = "Maximum autoscaled storage (GiB)."
  default     = 100
}

variable "db_backup_retention_days" {
  type        = number
  description = "Backup retention in days."
  default     = 7
}

variable "db_storage_type" {
  type        = string
  description = "RDS storage type."
  default     = "gp3"
}

variable "db_multi_az" {
  type        = bool
  description = "Enable Multi-AZ RDS."
  default     = false
}

variable "db_deletion_protection" {
  type        = bool
  description = "Enable RDS deletion protection."
  default     = false
}

variable "db_skip_final_snapshot" {
  type        = bool
  description = "Skip final snapshot on destroy."
  default     = true
}

variable "db_final_snapshot_identifier" {
  type        = string
  description = "Final snapshot identifier when skip_final_snapshot is false."
  default     = ""

  validation {
    condition     = var.db_skip_final_snapshot || var.db_final_snapshot_identifier != ""
    error_message = "db_final_snapshot_identifier is required when db_skip_final_snapshot is false."
  }
}

variable "db_apply_immediately" {
  type        = bool
  description = "Apply DB changes immediately."
  default     = true
}

variable "db_publicly_accessible" {
  type        = bool
  description = "Whether the RDS instance is publicly accessible."
  default     = false
}

variable "secrets_kms_key_arn" {
  type        = string
  description = "Optional KMS key ARN for Secrets Manager."
  default     = ""
}

variable "domain_name" {
  type        = string
  description = "Optional domain name for the ALB."
  default     = ""
}

variable "subject_alternative_names" {
  type        = list(string)
  description = "Subject alternative names for the ACM certificate."
  default     = []
}

variable "create_acm_certificate" {
  type        = bool
  description = "Whether to create an ACM certificate in this module."
  default     = false

  validation {
    condition     = !var.create_acm_certificate || var.domain_name != ""
    error_message = "domain_name must be set when create_acm_certificate is true."
  }
}

variable "certificate_arn" {
  type        = string
  description = "Existing ACM certificate ARN to use for HTTPS."
  default     = ""
}

variable "enable_http_to_https_redirect" {
  type        = bool
  description = "Redirect HTTP to HTTPS when TLS is enabled."
  default     = true
}

variable "alb_deletion_protection" {
  type        = bool
  description = "Enable deletion protection on the ALB."
  default     = false
}

variable "tags" {
  type        = map(string)
  description = "Additional tags applied to all resources."
  default     = {}
}
