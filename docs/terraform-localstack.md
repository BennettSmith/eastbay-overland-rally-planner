# LocalStack rehearsal (ECS + ECR wiring)

This is an optional rehearsal path to validate Terraform + ECS wiring using
LocalStack. It does not aim to be production-faithful for ALB/ACM/RDS.

## Prereqs

- LocalStack Pro (ECS support requires Pro)
- Docker + `docker compose`
- `aws` CLI and `terraform`

## Quick start

```bash
docker compose --profile localstack up -d localstack
docker compose up -d db

export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
export LOCALSTACK_ENDPOINT=http://localhost:4566

make tf-init-local
make tf-apply-local
```

## Build/push local images

LocalStack emulates ECR; point the AWS CLI to LocalStack for login:

```bash
aws --endpoint-url="${LOCALSTACK_ENDPOINT}" ecr create-repository --repository-name trip-planner-api-local
aws --endpoint-url="${LOCALSTACK_ENDPOINT}" ecr get-login-password | docker login --username AWS --password-stdin localhost:4566

docker build -t localhost:4566/trip-planner-api-local:local .
docker push localhost:4566/trip-planner-api-local:local

docker build -f Dockerfile.migrate -t localhost:4566/trip-planner-api-local:local-migrations .
docker push localhost:4566/trip-planner-api-local:local-migrations
```

## Run migrations

```bash
cd infra/local
terraform output -raw ecs_cluster_name
terraform output -raw migrations_task_definition_arn
terraform output -raw private_subnet_ids_csv
terraform output -raw ecs_task_security_group_id

aws --endpoint-url="${LOCALSTACK_ENDPOINT}" ecs run-task \
  --cluster "<cluster>" \
  --task-definition "<task-definition-arn>" \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[<subnet-ids>],securityGroups=[<sg-id>],assignPublicIp=ENABLED}"
```

## Notes and limitations

- ALB/ACM/DNS are not validated here; this rehearsal focuses on ECS + ECR wiring.
- LocalStack RDS support is limited; this path uses a host database URL (defaults to `host.docker.internal`).
- If your Docker host does not support `host.docker.internal`, override `database_url` in `infra/local/variables.tf` or via `TF_VAR_database_url`.
