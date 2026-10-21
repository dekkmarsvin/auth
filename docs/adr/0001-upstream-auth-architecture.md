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
