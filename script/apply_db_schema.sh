#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
project_root="$(cd -- "$script_dir/.." && pwd)"

# Take a private, restorable copy before applying the additive schema update.
backup_dir="$project_root/backups"
mkdir -p "$backup_dir"
chmod 700 "$backup_dir"
backup_file="$(mktemp "$backup_dir/auth-before-schema-$(date -u +%Y%m%dT%H%M%SZ)-XXXXXX.dump")"
chmod 600 "$backup_file"
if ! docker compose --file "$project_root/docker-compose.yml" exec -T postgresql \
  pg_dump --username=auth --dbname=auth --format=custom >"$backup_file"; then
  echo "Database backup failed; no schema changes were applied." >&2
  exit 1
fi
docker compose --file "$project_root/docker-compose.yml" exec -T postgresql \
  pg_restore --list <"$backup_file" >/dev/null
echo "Database backup saved to $backup_file"

exec docker compose \
  --file "$project_root/docker-compose.yml" \
  exec -T postgresql \
  psql \
  --username=auth \
  --dbname=auth \
  --set=ON_ERROR_STOP=1 \
  <"$project_root/sql/init.sql"
