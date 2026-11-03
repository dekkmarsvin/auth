# Follow upstream Auth with installation configuration outside Web Kit

The 2026-10-22 paired release adopts upstream native login and independent
registration/password-reset pages. Auth and its Web Kit are separate release
identities: a client SDK pin does not deploy the Auth service.

Use Auto-Novel's selected SDK unchanged, and follow the compatible Auth frontend
and API contract. Keep Kotoban origins, runtime Turnstile configuration, Mailgun
region, security and data protections, and Admin storage migration as narrow
installation adaptations. Do not maintain a permanent iframe/native-login split.

The initial transition requires coordinated Auth Web, Auth API and Auto-Novel
Web switching and rollback. Old consumers require iframe `login_success`; new
Auth no longer provides it. Old/new Turnstile actions must not be mixed. Use a
short authentication maintenance window with a tested release group instead of
leaving a permanent compatibility layer. No account/password or schema migration
is needed by this upstream range.

On 2026-11-03 the maintainer approved adopting Auth upstream through `cdc91ef1`.
Web Kit now lives in the independent `auto-novel/web-kit` repository. Auth uses
its upstream-selected `b6277d5`, while Auto-Novel uses `9c479ae7`; their core
authentication, API-client and storage implementations are identical. Keep
consumer pins separate: newer UI exports can disappear without changing the
wire contract, and Auth still uses `XAsyncContent` at its selected revision.

Remove the in-repository SDK workspace, migrate only the small test helpers
needed by this host's behavioral regressions, and preserve Admin's storage
adapter outside the SDK. Do not mount the shared Admin switcher, whose upstream
URLs are not installation configuration. This synchronization adds no SQL,
account, password, cookie or login-protocol migration. Its rollback is an image
rollback using the existing data and secrets, not a database recreation.
