output "tfstate_bucket_name" {
  description = "S3 bucket used for Terraform state."
  value       = aws_s3_bucket.tf_state.bucket
}

output "tfstate_lock_table_name" {
  description = "DynamoDB table used for Terraform state locking."
  value       = aws_dynamodb_table.tf_lock.name
}

output "github_oidc_provider_arn" {
  description = "OIDC provider ARN for GitHub Actions."
  value       = aws_iam_openid_connect_provider.github.arn
}

output "deploy_role_arns" {
  description = "Map of environment to GitHub deploy role ARN."
  value = {
    for env, role in aws_iam_role.github_deploy :
    env => role.arn
  }
}
