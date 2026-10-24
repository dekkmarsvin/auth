# Kotoban Auth fork

This installation follows the upstream Auth/Web Kit architecture. Preserve only
deployment identity, required security/data protections, and browser-storage
compatibility outside the SDK.

- `apps/web` serves independent `/register` and `/reset-password` flows.
  Consumers use the upstream Web Kit native login dialog; do not restore iframe
  login or a permanent old/new authentication adapter.
- Keep `packages/web-kit` exactly at the revision selected by the paired
  Auto-Novel release (currently upstream `9285460e2f48f4685c7994e9717453002aa0392e`).
  Put installation adaptations outside that package.
- Preserve `auth.kotoban.top`, the exact `https://books.kotoban.top` return origin,
  existing CORS origins, fork repository/image ownership, OCI revisions, pinned
  package manager, PostgreSQL/data mounts, and ignored environment/backups.
- Keep the public runtime Turnstile site-key endpoint, no-store responses,
  fail-closed widgets, exact action/hostname validation, replay protection, and
  production hostname preflight. Never expose secrets or call real OTP/mail for
  validation. Register/reset actions are `register`/`reset_password` on both sides.
- Preserve Mailgun domain/region configuration and backend account/role/token
  protections. Keep legacy-schema backup, transaction and revoked-record guards.
- Retain Auth Admin's one-time storage migration, existing-new-key precedence,
  and logout protection. Do not copy fork behavior into Web Kit.
- Sync on a separate branch with a dated `docs/sync` manifest and a
  `Sync-Manifest:` commit trailer. Record Auth source, SDK, paired Auto-Novel
  source, accepted groups, adaptations, validation and coordinated rollback.
- Before delivery, run frozen install, full workspace build/tests, Web Kit
  playground build, Go tests/vet/integration, schema/config fixtures, and touched
  container builds. After push, verify exact-head GitHub Actions.

The native-login transition is a paired Web/API/Auto-Novel release. Prepare and
verify the complete group before a short authentication maintenance window;
retain the previous group for rollback. This transition does not authorize
production database changes, secret changes, real email, or deployment by itself.
