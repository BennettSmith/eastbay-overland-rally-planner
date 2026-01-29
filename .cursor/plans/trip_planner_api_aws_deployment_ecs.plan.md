---
name: Trip Planner API AWS Deployment (ECS)
overview: Deploy trip-planner-api to AWS using the same container in dev/prod via ECS Fargate + ALB + RDS Postgres, managed by Terraform with GitHub Actions (OIDC) similar to authgenie-backend; keep Cloudflare DNS + ACM TLS, and add a first-class migrations job.
todos:
  - id: constitution-guardrails
    content: Add deployment/infrastructure guardrails to `CONSTITUTION.md` (IaC-only, no long-lived AWS creds, staging-first, prod applies via CI, secrets hygiene, migrations safety, LocalStack rehearsal encouraged).
    status: pending
  - id: bootstrap-terraform-oidc
    content: Add Terraform bootstrap for tfstate (S3+DDB) and GitHub OIDC deploy roles (staging/prod), mirroring authgenie’s pattern.
    status: pending
  - id: terraform-ecs-alb-rds
    content: Create Terraform module for VPC (public/private + NAT), ALB+ACM, ECR, ECS Fargate service (task roles/logs), RDS Postgres, and Secrets Manager wiring.
    status: pending
  - id: ci-cd-build-push-deploy
    content: "Add GitHub Actions deploy workflows: build/push image to ECR, terraform apply, run migrations task, update ECS service."
    status: pending
  - id: migrations-job
    content: Implement a dedicated migrations execution path (ECS one-off task) and document runbook + rollback considerations.
    status: pending
  - id: local-tilt-story
    content: Add a Tilt/K8s local dev layout that runs API + dependencies (Postgres, etc.) using the same container image.
    status: pending
  - id: localstack-rehearsal
    content: Add an optional-but-strongly-encouraged LocalStack rehearsal path (Terraform `infra/local/`, Makefile targets, docs) to validate ECS/ECR/task wiring before AWS.
    status: pending
isProject: false
---

## Reference pattern from authgenie-backend (what we’ll mirror)

- **Terraform-first** repo layout with a reusable module + per-environment roots, and a one-time bootstrap for tfstate + OIDC roles (see [`/Users/bsmith/Developer/authgenie/authgenie-backend/docs/terraform.md`](/Users/bsmith/Developer/authgenie/authgenie-backend/docs/terraform.md) and [`/Users/bsmith/Developer/authgenie/authgenie-backend/.github/workflows/deploy-staging.yml`](/Users/bsmith/Developer/authgenie/authgenie-backend/.github/workflows/deploy-staging.yml)).
- **GitHub Actions deploy** using **AWS OIDC** + Terraform `init/apply`.
- **Cloudflare-managed DNS** with **ACM** certificates (DNS validation records output by Terraform).

## Delivery workflow (branch + PR required)

- All implementation work should be done on a **feature branch** (no direct commits to `main`).
- The change set should land via a **GitHub Pull Request** against `main`, with:
  - CI green (unit/integration tests + terraform lint/validate as applicable)
  - A PR description that references the plan todos and states what is/isn’t included
  - A clear “how to verify” section (local Tilt steps, optional LocalStack rehearsal steps, and staging smoke test expectations)

## Current trip-planner-api state (what we’ll preserve)

- A standard HTTP server entrypoint in [`/Users/bsmith/Developer/Overland-East-Bay/trip-planner-api/cmd/api/main.go`](/Users/bsmith/Developer/Overland-East-Bay/trip-planner-api/cmd/api/main.go) (no Lambda event model).
- A production-grade static binary container build in [`/Users/bsmith/Developer/Overland-East-Bay/trip-planner-api/Dockerfile`](/Users/bsmith/Developer/Overland-East-Bay/trip-planner-api/Dockerfile).
- Env-based config (notably `DATABASE_URL`, `JWT_*`, `PUBLIC_BASE_URL`) in [`/Users/bsmith/Developer/Overland-East-Bay/trip-planner-api/.env.example`](/Users/bsmith/Developer/Overland-East-Bay/trip-planner-api/.env.example).

## Target AWS architecture (same container in prod)

- **ECR**: store the built `trip-planner-api` image.
- **ECS (Fargate)**:
- Cluster + Service running the container.
- Task definition injects env vars and Secrets Manager references.
- CloudWatch Logs for app logs.
- **ALB**:
- HTTPS listener using **ACM** cert.
- Target group health checks (use `/healthz` if present; otherwise add a lightweight endpoint).
- **RDS Postgres**:
- Private subnets in a VPC.
- Security groups allow ECS tasks to connect to Postgres.
- Credentials in **Secrets Manager**; app receives `DATABASE_URL` via secret injection (or assembled from discrete secret fields).
- **Networking**:
- Public subnets for ALB.
- Private subnets for ECS tasks and RDS.
- NAT gateway for ECS egress (e.g., JWKS fetches via `JWT_JWKS_URL`).
- **DNS/TLS** (optional but recommended):
- Custom domain (suggested defaults): `api.staging.overlandeastbay.com` and `api.overlandeastbay.com`.
- Terraform outputs ACM DNS validation records; you add them in Cloudflare.

## Migrations (first-class, robust)

- Add a **one-off ECS task** (“migration job”) using either:
- the same image with a `CMD`/flag that runs migrations, or
- a small purpose-built migration image.
- CI/CD runs migrations **before** updating the ECS service (idempotent, safe re-runs).

## Repo layout to add in trip-planner-api (mirrors authgenie structure)

- `infra/bootstrap/`: S3 tfstate bucket, DynamoDB lock table, GitHub OIDC deploy roles.
- `infra/aws/modules/trip-planner-api/`: VPC, ALB, ECS, ECR, RDS, Secrets, CloudWatch.
- `infra/aws/envs/staging/` and `infra/aws/envs/prod/`: environment roots.

## CI/CD to add (GitHub Actions)

- `deploy-staging.yml` (push to `main`) and `deploy-prod.yml` (manual):
- build and push Docker image to ECR (tag with git SHA; optionally also `latest` per env)
- terraform init/apply (OIDC)
- run migrations task
- update ECS service to new task definition/image tag

## Local dev with Tilt + Docker Desktop K8s + LocalStack

- **Inner loop (primary)**: Docker Desktop K8s + Tilt.
- Goal: fast edit → rebuild → redeploy → debug.
- Run the **same container image** locally and in prod (built from [`/Users/bsmith/Developer/Overland-East-Bay/trip-planner-api/Dockerfile`](/Users/bsmith/Developer/Overland-East-Bay/trip-planner-api/Dockerfile)).
- Use local/in-cluster Postgres and do the bulk of development work (debugging, unit tests, integration tests) here.

- **Rehearsal loop (optional, strongly encouraged)**: LocalStack Pro + Terraform `infra/local/`.
- Goal: validate that our Terraform + ECS wiring is correct (and that the container boots in an AWS-like environment) **before** trying AWS.
- This is strongly encouraged via docs + Makefile targets, but it’s **not a required gate**.

Suggested shape:

- `infra/local/` Terraform root that stands up the **subset** worth rehearsing locally:
- ECR repo(s) (or stubs) and image tag conventions
- ECS cluster/service/task definition wiring (port mapping, env vars, health checks)
- CloudWatch Logs wiring (where supported)
- Secrets plumbing (where supported)
- Explicitly **do not** aim for full parity locally for:
- ACM cert DNS validation + Cloudflare DNS
- ALB TLS listener/cert validation edge-cases
- Production-grade networking nuances

- **Encourage mechanisms**:

- Makefile targets like `make tf-apply-local` / `make tf-destroy-local`
- A short `docs/terraform-localstack.md` with a 5-minute happy path
- Optional GitHub Action `workflow_dispatch` job to run the LocalStack apply as a preflight (not a required gate)

## Testing strategy by environment (explicit)

- **Unit tests**: run locally (and in CI) without LocalStack.
- **Integration tests (preferred local path)**: run via Tilt/K8s + local Postgres.
- **Integration rehearsal (LocalStack)**: validate AWS wiring + basic service boot.
- Good tests here: Terraform `apply/destroy`, ECS service steady state, logs appear, minimal HTTP smoke tests.
- Avoid relying on LocalStack for ALB/ACM/DNS/RDS fidelity.
- **E2E tests (AWS staging)**: validate the real stack (ALB + TLS + RDS) before promoting to prod.

## Definition of Done (DoD) + Verification (per todo)

### `bootstrap-terraform-oidc`

- **Deliverables**:
- `infra/bootstrap/` Terraform root that provisions:
- Terraform remote state prerequisites (S3 bucket + DynamoDB lock table)
- GitHub OIDC trust + deploy roles for `staging` and `prod`
- Minimal docs/runbook for the one-time bootstrap.

- **Verification**:
- **Static checks**: `terraform fmt -check`, `terraform validate` for `infra/bootstrap/`.
- **Bootstrap apply**: `terraform apply` succeeds in a clean AWS account.
- **OIDC proof**: a GitHub Actions job can assume the deploy role via OIDC and run at least:
- `aws sts get-caller-identity`
- a no-op `terraform plan` against an env root (after init)

- **Exit criteria**:
- A fresh repo + AWS account can be bootstrapped without manual IAM key distribution, and GitHub Actions can authenticate via OIDC.

### `terraform-ecs-alb-rds`

- **Deliverables**:
- `infra/aws/modules/trip-planner-api/` module that provisions:
- VPC (public + private subnets), NAT, routing, security groups
- ECR repo
- ECS cluster + task definition + service (Fargate)
- ALB + target group + listeners (HTTP→HTTPS redirect optional)
- ACM cert (optional custom domain)
- RDS Postgres + subnet group + parameter group (as needed)
- Secrets Manager secret(s) + ECS secret injection wiring for DB creds / `DATABASE_URL`
- CloudWatch log group(s) and ECS logging config
- `infra/aws/envs/staging/` and `infra/aws/envs/prod/` roots that instantiate the module with env-specific names/variables.

- **Verification**:
- **Static checks**: `terraform fmt -check`, `terraform validate` for each env root and module.
- **AWS staging deploy**:
- `terraform apply` succeeds.
- ECS service reaches steady state (desired tasks == running tasks).
- Target group reports healthy targets.
- HTTPS endpoint returns `200` on health check.
- App can connect to RDS with injected secret (e.g., startup logs indicate successful connection, or a DB-backed endpoint succeeds).

- **Exit criteria**:
- Staging environment is reachable over HTTPS and serves requests via ALB, with persistence backed by RDS.

### `ci-cd-build-push-deploy`

- **Deliverables**:
- GitHub Actions workflows:
- `deploy-staging.yml` (push to `main`)
- `deploy-prod.yml` (manual / environment-protected)
- GitHub Environments (`staging`, `prod`) configured with required variables/secrets (AWS role ARN, region, etc.).
- ECR image tagging strategy documented (e.g., git SHA + optional env tag).

- **Verification**:
- On staging deploy:
- Workflow builds image and pushes to ECR successfully.
- Workflow applies Terraform successfully.
- ECS service updates to a new task definition revision that references the new image tag.
- Post-deploy smoke test hits the service endpoint and passes.

- **Exit criteria**:
- A merge to `main` results in a new version running in staging with no manual AWS console steps.

### `migrations-job`

- **Deliverables**:
- A repeatable migrations mechanism as a one-off ECS task (invokable from CI), using either:
- the same container image with a dedicated command/flag, or
- a dedicated migration image.
- Runbook: how to run migrations, how to recover from a failed migration, and how to re-run safely.

- **Verification**:
- Migrations are **idempotent** (running the job twice results in no changes / no errors).
- CI deploy runs migrations **before** service update.
- Failure behavior: if migrations fail, the deploy workflow stops and the ECS service is not advanced.

- **Exit criteria**:
- Schema updates are consistently applied as part of deploy and are safe to retry.

### `local-tilt-story`

- **Deliverables**:
- A `Tiltfile` + supporting Kubernetes manifests (or Helm/Kustomize) that run:
- `trip-planner-api` container
- local/in-cluster Postgres (and any required sidecars)
- Short docs: “clone → `tilt up` → run requests/tests”.

- **Verification**:
- `tilt up` results in a reachable local API endpoint.
- Code change triggers rebuild/redeploy (inner-loop experience).
- Local integration tests can be run against the Tilt environment (at minimum: health + a representative flow).

- **Exit criteria**:
- Developers can iterate quickly on API code with auto-reload/redeploy and realistic dependencies, without AWS/LocalStack.

### `localstack-rehearsal`

- **Deliverables**:
- `infra/local/` Terraform root targeting LocalStack (subset of resources that are valuable to emulate):
- ECR and ECS wiring (cluster/service/task definition)
- logging/secrets wiring where supported
- Make targets such as `tf-init-local`, `tf-apply-local`, `tf-destroy-local`.
- `docs/terraform-localstack.md` describing the 5-minute happy path and known limitations.

- **Verification**:
- `terraform apply` and `terraform destroy` succeed against LocalStack.
- ECS service starts and container runs.
- Minimal HTTP smoke test passes (health + one representative endpoint), if endpoint exposure is available.

- **Exit criteria**:
- Developers can quickly rehearse the AWS-shaped “container runs under ECS” assumption locally, and catch wiring errors before AWS.

## Release checklist (staging → prod)

- **Preflight (local/CI)**:
- Unit tests pass.
- Integration tests pass in Tilt/K8s (preferred).
- `terraform fmt -check` + `terraform validate` pass for module + env roots.
- PR is open against `main` (feature branch), with CI green.

- **Optional preflight (strongly encouraged)**:
- LocalStack rehearsal: `infra/local` apply succeeds, ECS service runs, smoke tests pass.

- **Staging deploy**:
- Deploy workflow succeeds end-to-end (build → push → terraform apply → migrations → ECS update).
- Target group healthy; smoke tests pass.
- Basic operational checks: logs present, error rates acceptable.

- **Promotion to prod**:
- Prod deploy workflow (manual) succeeds end-to-end.
- Post-deploy smoke tests pass against prod endpoint.
- Rollback plan verified: ability to redeploy prior task definition/image tag; migration rollback strategy understood/documented.

## Key implementation notes

- Prefer **Secrets Manager** and ECS secret injection over plain TF vars for DB creds.
- Add/standardize a **health endpoint** suitable for ALB target group checks if not already present.
- Keep config contract stable: the container still reads `PORT`, `DATABASE_URL`, `JWT_*`, etc.