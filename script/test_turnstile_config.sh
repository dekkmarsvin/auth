#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
checker="$script_dir/check_turnstile_config.sh"
export TURNSTILE_SITE_KEY=0xfixture_site_key_123456789
export TURNSTILE_SECRET=0xfixture_secret_key_12345678901234567890
export TURNSTILE_HOSTNAMES=auth.kotoban.top
sh "$checker" --environment >/dev/null

must_reject() {
  if env "$@" sh "$checker" --environment >/dev/null 2>&1; then
    printf '%s\n' 'An invalid Turnstile production fixture was accepted.' >&2
    exit 1
  fi
}

must_reject TURNSTILE_SITE_KEY=
must_reject TURNSTILE_SECRET=
must_reject TURNSTILE_HOSTNAMES=
must_reject TURNSTILE_SITE_KEY=1x00000000000000000000AA
must_reject TURNSTILE_SECRET=1x0000000000000000000000000000000AA
must_reject TURNSTILE_SITE_KEY='<site-key-placeholder>'
must_reject TURNSTILE_SECRET=short
must_reject TURNSTILE_HOSTNAMES='*.kotoban.top'
must_reject TURNSTILE_HOSTNAMES=localhost
must_reject TURNSTILE_HOSTNAMES=auth.kotoban.top,localhost
must_reject TURNSTILE_HOSTNAMES=auth.kotoban.top.evil.example
must_reject TURNSTILE_HOSTNAMES='auth. kotoban.top'
printf '%s\n' 'Turnstile deployment configuration regression checks passed.'
