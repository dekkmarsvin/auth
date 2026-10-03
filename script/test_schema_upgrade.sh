#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
project_root="$(cd -- "$script_dir/.." && pwd)"
container_name="auth-schema-test-$$"
docker run --rm --detach --name "$container_name" \
  --env POSTGRES_USER=auth --env POSTGRES_PASSWORD=auth-schema-test-password \
  --env POSTGRES_DB=auth_test --tmpfs /var/lib/postgresql/data \
  postgres:17-alpine >/dev/null
trap 'docker stop "$container_name" >/dev/null' EXIT INT TERM

for _ in {1..30}; do
  if docker exec "$container_name" pg_isready -h 127.0.0.1 -U auth -d auth_test >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
docker exec "$container_name" pg_isready -h 127.0.0.1 -U auth -d auth_test >/dev/null

psql_args=(--username=auth --dbname=auth_test --set=ON_ERROR_STOP=1)
docker exec -i "$container_name" psql "${psql_args[@]}" <"$project_root/sql/tests/legacy-schema.sql" >/dev/null
for _ in 1 2; do
  docker exec -i "$container_name" psql "${psql_args[@]}" <"$project_root/sql/init.sql" >/dev/null
done
docker exec -i "$container_name" psql "${psql_args[@]}" <<'SQL'
do $$
begin
    if not exists (select 1 from auth_user where id=1 and username='legacy-user'
        and password='preserved-password-hash' and attr='{"legacy":true}'::jsonb
        and last_seen_strike_id=0) then
        raise exception 'The upgrade changed the legacy user';
    end if;
    if not exists (select 1 from auth_strike_record where id=1 and user_id=1
        and reason='existing penalty' and evidence='retained evidence'
        and status=0 and revoked_at is null and revoked_by is null) then
        raise exception 'The upgrade changed the legacy penalty';
    end if;
end $$;
insert into auth_strike_record (user_id, reason, evidence, status)
values (1, 'old revoked penalty', 'unknown revocation date', 1);
SQL
if docker exec -i "$container_name" psql "${psql_args[@]}" <"$project_root/sql/init.sql" >"/dev/null" 2>&1; then
  echo 'The upgrade must refuse unmapped historical revocations.' >&2
  exit 1
fi
echo 'Legacy schema upgrade and historical revocation guard passed.'
