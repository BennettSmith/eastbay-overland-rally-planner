#!/bin/sh
set -eu

if [ -z "${DATABASE_URL:-}" ]; then
  echo "DATABASE_URL is required" >&2
  exit 1
fi

if [ "$#" -gt 0 ]; then
  COMMAND="$*"
elif [ -n "${MIGRATE_COMMAND:-}" ]; then
  COMMAND="${MIGRATE_COMMAND}"
else
  COMMAND="up"
fi

exec migrate -path /migrations -database "${DATABASE_URL}" ${COMMAND}
