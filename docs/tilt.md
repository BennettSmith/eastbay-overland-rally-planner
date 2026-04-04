# Local dev with Tilt

This inner loop runs the API and Postgres in Kubernetes using the same container
image as production.

## Prereqs

- Docker Desktop with Kubernetes enabled (or another local Kubernetes cluster)
- Tilt installed (`https://tilt.dev`)
- `kubectl` configured for the local cluster

## Quick start

```bash
tilt up
```

In the Tilt UI:

- Wait for `postgres` to be healthy.
- `trip-planner-api` will build and deploy automatically.
- Trigger `trip-planner-migrations` once to apply schema changes.

The API is exposed on `http://localhost:8080` via port-forward.

```bash
curl http://localhost:8080/healthz
```

For authenticated endpoints, use the dev header:

```bash
curl -H "X-Debug-Subject: dev|local" http://localhost:8080/members/me
```

## Notes

- The Postgres data volume uses `emptyDir`, so data resets on pod restart.
- To re-run migrations, trigger `trip-planner-migrations` again in Tilt.
