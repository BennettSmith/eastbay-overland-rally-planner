docker_build("trip-planner-api-local", ".", dockerfile="Dockerfile")
docker_build("trip-planner-api-migrate-local", ".", dockerfile="Dockerfile.migrate")

k8s_yaml([
    "deploy/tilt/postgres.yaml",
    "deploy/tilt/api.yaml",
    "deploy/tilt/migrations-job.yaml",
])

k8s_resource("postgres")
k8s_resource("trip-planner-api", port_forwards=8080, resource_deps=["postgres"])
k8s_resource("trip-planner-migrations", resource_deps=["postgres"], trigger_mode=TRIGGER_MODE_MANUAL)
