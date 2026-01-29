# Terraform bootstrap (state + GitHub OIDC)

This root provisions the shared AWS prerequisites used by all environments:

- S3 bucket for Terraform state
- DynamoDB table for state locking
- GitHub OIDC provider
- GitHub deploy roles for `staging` and `prod`

## Prereqs

- Terraform >= 1.6
- AWS credentials with permissions to create S3, DynamoDB, IAM, and OIDC providers

## One-time apply

```bash
cd infra/bootstrap
terraform init
terraform apply \
  -var "aws_region=us-west-2" \
  -var "tfstate_bucket_name=trip-planner-api-tfstate" \
  -var "tfstate_lock_table_name=trip-planner-api-tfstate-lock" \
  -var "github_org=Overland-East-Bay" \
  -var "github_repo=trip-planner-api"
```

If you use GitHub Environments for OIDC scoping, keep the default `github_subjects`
map; otherwise override it to match your branch-based policy.

## Next steps (environment roots)

Use the outputs to configure remote state in environment roots:

```hcl
terraform {
  backend "s3" {
    bucket         = "<tfstate_bucket_name>"
    key            = "aws/<environment>/terraform.tfstate"
    region         = "<aws_region>"
    dynamodb_table = "<tfstate_lock_table_name>"
    encrypt        = true
  }
}
```

## GitHub Actions configuration (names only)

Create GitHub Environments named `staging` and `prod`, then add variables/secrets:

- `AWS_REGION` (matches `aws_region`)
- `AWS_ROLE_ARN` (from `deploy_role_arns` output per environment)

No long-lived AWS keys should be stored in the repo or GitHub secrets.
