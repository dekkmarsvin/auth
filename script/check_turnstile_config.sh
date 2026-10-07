#!/bin/sh
set -eu

fail() {
  printf '%s\n' "$1" >&2
  exit 1
}

# Internal mode is also used by fixture tests. Never print environment values.
if [ "${1:-}" = '--environment' ]; then
  [ -n "${TURNSTILE_SITE_KEY:-}" ] || fail 'TURNSTILE_SITE_KEY is missing.'
  [ -n "${TURNSTILE_SECRET:-}" ] || fail 'TURNSTILE_SECRET is missing.'
  [ -n "${TURNSTILE_HOSTNAMES:-}" ] || fail 'TURNSTILE_HOSTNAMES is missing.'
  for value in "$TURNSTILE_SITE_KEY" "$TURNSTILE_SECRET"; do
    case "$value" in
      *[!A-Za-z0-9_-]*) fail 'Turnstile keys contain invalid characters.' ;;
      1x000000*|2x000000*|3x000000*|4x000000*|5x000000*)
        fail 'Turnstile test keys cannot be used for production.' ;;
    esac
  done
  [ "${#TURNSTILE_SITE_KEY}" -ge 20 ] || fail 'Turnstile site key is invalid.'
  [ "${#TURNSTILE_SECRET}" -ge 32 ] || fail 'Turnstile secret is invalid.'
  hostnames=$(printf '%s' "$TURNSTILE_HOSTNAMES" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
  [ "$hostnames" = 'auth.kotoban.top' ] ||
    fail 'Production Turnstile hostnames must allow only auth.kotoban.top.'
  printf '%s\n' 'Production Turnstile configuration checks passed.'
  exit 0
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
project_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
cd "$project_root"
# Compose resolves the same .env and service environment used during rollout.
# This disposable container runs only the checker; it never starts the API or DB.
docker compose run --rm --no-deps -T --entrypoint /bin/sh api \
  -s -- --environment < "$script_dir/check_turnstile_config.sh"
