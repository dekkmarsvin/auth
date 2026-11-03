# Kotoban Auth fork

This installation follows the upstream Auth/Web Kit architecture. Preserve only
deployment identity, required security/data protections, and browser-storage
compatibility outside the SDK.

- `apps/web` serves independent `/register` and `/reset-password` flows.
  Consumers use the upstream Web Kit native login dialog; do not restore iframe
  login or a permanent old/new authentication adapter.
- Web Kit is an immutable dependency from `auto-novel/web-kit`, rather than a
  local `packages/web-kit` workspace. Use the revision selected by Auth upstream
  (currently `b6277d5be30e40d8911f1c4f5f80e5baf14b45a7`). Auto-Novel selects its
  own compatible revision; do not force both consumers to the same pin.
  Put installation adaptations outside the SDK and do not restore its sources.
- Do not mount the upstream `AdminSiteSwitcher`: its URLs point to upstream
  installations. Keep this installation's ordinary navigation and same-origin
  administrator entry; never expose an unconfigured external forum.
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
- Preserve local behavioral tests and their host-owned helpers when removing
  the SDK workspace. Before delivery, run frozen install, full workspace
  build/tests, Go tests/vet/integration, schema/config fixtures, and touched
  container builds. The independently pinned SDK is not patched or rebuilt here.
  After push, verify exact-head GitHub Actions.

The original native-login transition required a paired Web/API/Auto-Novel
release. Subsequent updates retain the same JWT/session/login/refresh contract;
record source identities and verify compatibility instead of inferring a new
protocol transition from SDK commit counts. Keep previous images for rollback.
Source synchronization does not by itself authorize production database changes,
secret changes, real email, or deployment.
