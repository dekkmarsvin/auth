# 2026-11-03 Auth upstream sync through cdc91ef1

## Source identities

- Fork baseline: `75b5cc97042437398448743acd635a245c7ebce4`.
- Upstream range evaluated: `9285460e2f48f4685c7994e9717453002aa0392e..cdc91ef129f4b0a48ee622625c3ac84fdb634893`, 24 commits.
- Auth upstream evaluated through: `cdc91ef129f4b0a48ee622625c3ac84fdb634893`.
- Independent Auth Web Kit pin: `b6277d5be30e40d8911f1c4f5f80e5baf14b45a7` from `auto-novel/web-kit`.
- Paired Auto-Novel upstream: `65f40dc0`; its independent SDK pin is `9c479ae742335830fc4383f1fdb451809a4d56d0`.
- Next Auth sync starts after: `cdc91ef129f4b0a48ee622625c3ac84fdb634893`.

The independently fetched SDK main was `9c479ae7`. Auth deliberately follows its
own selected pin: the newer SDK removes `XAsyncContent`, which this Admin still
uses. `create.ts`, `auth/session.ts`, `api/client.ts`, `api/requests.ts`,
`context.ts`, `types.ts` and `utils/storage.ts` have identical Git blobs at the
old bundled `9285460e`, selected `b6277d5` and Auto-Novel's `9c479ae7`.

## Accepted upstream groups

| Group                  | Representative commits                                                                                                             | Result                                                                                                                                                                         |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Independent SDK        | `00c7cf6`, `92ecc61`, `93f27b7`, `cdc91ef`                                                                                         | Remove the bundled workspace and use Auth's exact independent SDK dependency.                                                                                                  |
| Shared Admin UI        | `e29801c`, `d456f17`, `37269a6`, `53ea96a`                                                                                         | Replace Naive UI with shared/native controls, date and pagination adapters, dialogs, status labels and loading skeletons; retain existing API operations and permission gates. |
| SDK improvements       | `d3b962d`, `1c59152`, `40d140c`, `7a5cf3a`, `81cfb7c`, `bfd11f0`, `f29111e`, `6e1b426`, `1902e78`, `c97324b`, `5d1e8a2`, `847a89e` | Adopt via the immutable selected dependency rather than maintaining local SDK changes.                                                                                         |
| Deployment maintenance | `b6f84de`, `c88c11a`                                                                                                               | Follow Go/Alpine base images and independent-SDK Web installation/build context.                                                                                               |
| Account Web styling    | `81a9d7b`                                                                                                                          | Already represented by the preserved account Web styling; no account-flow contract change.                                                                                     |
| Admin switcher         | `762e9f6`                                                                                                                          | Evaluate but omit the upstream-host switcher; existing navigation remains available.                                                                                           |

No upstream merge commit is merged. The candidate range has no Auth API source,
account Web logic or SQL changes; a whole-tree replacement would erase older
fork safety adaptations and is not the delivery method.

## Fork adaptations and retained boundaries

- Keep Kotoban domains, exact return-origin validation, public runtime Turnstile
  configuration/no-store/fail-closed behavior, action/hostname/replay checks,
  existing CORS and token/cookie contracts, Mailgun domain/region, secrets and
  PostgreSQL mounts. Do not call real OTP/mail/CAPTCHA to prove the sync.
- Keep Admin's one-time storage migration, existing-new-key precedence,
  first-paint ordering and logout/stale-tab protection outside the SDK.
- Remove no data, backup, schema, historical-revocation or transaction guard.
- Omit `AdminSiteSwitcher` because its immutable implementation hardcodes
  Novelia installations and an external forum. Do not patch the SDK.
- Move the existing small SFC compilation/test renderer helpers to host test
  support before removing the SDK workspace; retain Web/Admin test commands.
- Retain the pinned `pnpm@10.33.4`, verified by frozen installation. Keep the
  consolidated CI gate and portable image owner/OCI source labels.
- The SDK package requires Node >=22.18.0; declared host requirements and CI
  must satisfy it. Package pins do not establish production image identity.

## Release and rollback

This range does not repeat the iframe-to-native transition. Existing account,
password, JWT, refresh Cookie and session storage formats remain compatible;
there is no SQL or infrastructure migration. Prepare and verify the complete
release group before root-controlled publication/deployment. Retain previous
Web/API images and exact revision evidence; preserve data-service identities,
mounts and secrets. A source/build result alone does not establish runtime or
live third-party availability.

## Validation

- `corepack pnpm install --frozen-lockfile --config.confirmModulesPurge=false`
  passed using pnpm `10.33.4`; the lockfile remains exactly Auth upstream's blob.
- `corepack pnpm run build` passed both Account Web and Admin, including
  vue-tsc. Admin retains its vendor-chunk size warning.
- `corepack pnpm test` passed all 24 host regressions: 14 Admin migration/
  first-paint/logout tests and 10 Account Web route/return/API/runtime-config
  cases. The App SFC still uses the migrated host helpers.
- In a disposable Go `1.27.3-alpine3.24` container, `go test ./...` and
  `go vet ./...` passed with the source mounted read-only and `-mod=readonly`.
- `go test -tags=integration ./tests -count=1 -p=2` passed against a disposable
  PostgreSQL `17.7-alpine3.23` fixture on an isolated network without a host port.
  Mail and CAPTCHA use the existing local/no-op fixtures.
- The checked-in legacy schema fixture and the assertions from
  `script/test_schema_upgrade.sh` passed in a separate disposable fixture DB:
  two applications retained the legacy user/penalty, and an unmapped revoked
  record caused the expected rejection. Production schema files are unchanged.
- The exact `script/test_turnstile_config.sh` passed in a disposable Alpine
  `3.24.2` container with its checked-in fake configuration.
- Git comparison confirms no changes to Auth API implementation, account Web
  implementation, Mailgun/CORS, token/cookie contracts, SQL/schema/backup/config
  scripts or Admin storage/bootstrap implementation. No bundled SDK tracked
  file remains; the dependency is the exact immutable selected Git revision.
- Focused Prettier checks and `git diff --check` passed. Fixture database and
  isolated test network were removed; module/build caches are local only.
- Sequential `podman build --layers -f deploy/web/Dockerfile` and
  `podman build --layers -f deploy/api/Dockerfile` passed with the final application
  source. Local validation image IDs are Web
  `709e67e44d624844677799a1e513e106f293e115b88ad3fb5e8263d93ce10487` and API
  `7bbd7c8f0714e3b02cc04ca32470b81904764f87e8d60aa74c48cac0a1e860b4`.
  These uncommitted-source builds have no production provenance claim; published
  images must carry their final CI revision and build time.
- Exact-head GitHub Actions and production deployment remain root-controlled
  delivery steps, not results of this local validation. No commit, push, real
  OTP/email or production operation occurred in this implementation subtask.

The curated commit must carry:
`Sync-Manifest: docs/sync/2026-11-03-upstream-cdc91ef1.md`.
