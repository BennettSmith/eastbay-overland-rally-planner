variable "project_name" {
  type        = string
  description = "Short project identifier used in resource names."
  default     = "trip-planner-api"
}

variable "aws_region" {
  type        = string
  description = "AWS region for all bootstrap resources."
  default     = "us-west-2"
}

variable "tfstate_bucket_name" {
  type        = string
  description = "S3 bucket name for Terraform state."
  default     = "trip-planner-api-tfstate"
}

variable "tfstate_bucket_force_destroy" {
  type        = bool
  description = "Allow force-destroying the tfstate bucket (use only for teardown)."
  default     = false
}

variable "tfstate_lock_table_name" {
  type        = string
  description = "DynamoDB table name for Terraform state locks."
  default     = "trip-planner-api-tfstate-lock"
}

variable "github_org" {
  type        = string
  description = "GitHub organization or user that owns the repository."
  default     = "Overland-East-Bay"
}

variable "github_repo" {
  type        = string
  description = "GitHub repository name."
  default     = "trip-planner-api"
}

variable "github_subjects" {
  type        = map(list(string))
  description = "Map of environment to allowed OIDC subject patterns."
  default = {
    staging = ["repo:Overland-East-Bay/trip-planner-api:environment:staging"]
    prod    = ["repo:Overland-East-Bay/trip-planner-api:environment:prod"]
  }
}

variable "github_oidc_thumbprints" {
  type        = list(string)
  description = "Thumbprints for the GitHub OIDC provider."
  default     = ["6938fd4d98bab03faadb97b34396831e3780aea1"]
}

variable "deploy_role_policy_arns" {
  type        = list(string)
  description = "Managed policy ARNs attached to the GitHub deploy roles."
  default     = ["arn:aws:iam::aws:policy/AdministratorAccess"]
}

variable "tags" {
  type        = map(string)
  description = "Additional tags applied to all resources."
  default     = {}
}
