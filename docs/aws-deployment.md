# AWS deployment (ECS + ALB + RDS)

This repo uses Terraform and GitHub Actions (OIDC) to deploy to AWS. No long-lived
AWS credentials should be stored in GitHub or the repo.

## Bootstrap (one-time)

Run the Terraform bootstrap in `infra/bootstrap` to create:

- S3 bucket + DynamoDB table for Terraform state
- GitHub OIDC provider
- GitHub deploy roles for `staging` and `prod`

See `infra/bootstrap/README.md` for exact commands and outputs.

## GitHub Environments (names only)

Create GitHub Environments named `staging` and `prod` with these variables:

- `AWS_REGION` (example: `us-west-2`)
- `AWS_ROLE_ARN` (from bootstrap output `deploy_role_arns`)
- `TF_STATE_BUCKET` (from bootstrap output `tfstate_bucket_name`)
- `TF_STATE_LOCK_TABLE` (from bootstrap output `tfstate_lock_table_name`)
- `PROJECT_NAME` (default: `trip-planner-api`)
- `JWT_ISSUER`
- `JWT_AUDIENCE`
- `JWT_JWKS_URL`

Optional environment variables (set only if you need overrides):

- `DOMAIN_NAME` (matches the env root default domain)
- `CERTIFICATE_ARN` (use an existing ACM cert instead of creating one)
- `CREATE_ACM_CERTIFICATE` (`true` to request ACM via Terraform)
- `PUBLIC_BASE_URL` (optional app base URL)

## Terraform variables

Terraform env roots read variables from the GitHub Environment variables above.
If you need to override additional settings, use `TF_VAR_*` variables in the
workflow or set them locally when running Terraform.

## Deploy workflows

- `deploy-staging.yml` runs on pushes to `main`.
- `deploy-prod.yml` is manual (`workflow_dispatch`).

Both workflows:

- Assume the AWS role via GitHub OIDC
- Build and push the Docker image to ECR
- Apply Terraform for the environment

## LocalStack rehearsal (optional)

See `docs/terraform-localstack.md` for a fast ECS/ECR wiring rehearsal path.

## Migrations (ECS task)

Migrations run as a one-off ECS task using the `Dockerfile.migrate` image. The
task is wired to the same database secret as the service and defaults to `up`.

### Runbook

Use the deployment workflow (preferred) or run manually:

```bash
cd infra/aws/envs/<env>
terraform output -raw ecs_cluster_name
terraform output -raw migrations_task_definition_arn
terraform output -raw private_subnet_ids_csv
terraform output -raw ecs_task_security_group_id

aws ecs run-task \
  --cluster "<cluster>" \
  --task-definition "<task-definition-arn>" \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[<subnet-ids>],securityGroups=[<sg-id>],assignPublicIp=DISABLED}"
```

To run a different migration command (e.g., down 1), override the container
command or set `MIGRATE_COMMAND` when invoking the task.

### Rollback considerations

Prefer forward-only migrations. If a rollback is required, restore from an RDS
snapshot and redeploy a compatible application version. Use `migrate down`
only when you have explicitly tested the rollback path.
