#!/bin/bash -e

# Usage: ./wait-for-db.sh <docker-compose-project>

until [ "$(docker inspect --format='{{.State.Health.Status}}' "${1}-db-1")" = "healthy" ]; do
  echo "Waiting for postgres to be healthy..."
  sleep 2
done
echo "Postgres is healthy!"
