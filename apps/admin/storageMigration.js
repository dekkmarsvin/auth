/** One attempt per origin; a persisted marker prevents logout reviving legacy data. */
export function migrateLegacyAdminStorage(storage) {
  if (!storage) return;
  const marker = 'auth-admin:web-kit-storage-migration:v1';
  const oldSession = 'auth-admin-session';
  const oldTheme = 'auth-admin-theme';

  function validSession(raw) {
    try {
      const value = JSON.parse(raw);
      if (typeof value?.token !== 'string') return;
      const payload = value.token.split('.')[1];
      if (!payload) return;
      const base64 = payload.replace(/-/g, '+').replace(/_/g, '/');
      const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4);
      const claims = JSON.parse(
        new TextDecoder().decode(
          Uint8Array.from(atob(padded), (character) => character.charCodeAt(0)),
        ),
      );
      if (
        !Number.isSafeInteger(claims.uid) ||
        claims.uid <= 0 ||
        typeof claims.sub !== 'string' ||
        !claims.sub ||
        !['admin', 'trusted', 'member', 'restricted', 'banned'].includes(
          claims.role,
        ) ||
        !Number.isFinite(claims.crat) ||
        !Number.isFinite(claims.iat) ||
        !Number.isFinite(claims.exp) ||
        Date.now() >= claims.exp * 1000
      ) {
        return;
      }
      return JSON.stringify({
        token: value.token,
        adminMode: claims.role === 'admin' && value.adminMode === true,
      });
    } catch {
      return;
    }
  }

  try {
    if (storage.getItem(marker) !== null) return;
    // Claim before any copy. If persistence is unavailable, use cookie refresh
    // and the upstream theme fallback rather than importing a replayable token.
    storage.setItem(marker, 'attempted');
    if (storage.getItem(marker) !== 'attempted') return;

    for (const [legacyKey, currentKey, convert] of [
      [oldSession, 'web-kit:session:v1', validSession],
      [
        oldTheme,
        'web-kit:theme:v1',
        (raw) =>
          raw === 'light' || raw === 'dark' ? JSON.stringify(raw) : undefined,
      ],
    ]) {
      try {
        // Any current value takes precedence, including one the SDK rejects.
        if (storage.getItem(currentKey) === null) {
          const raw = storage.getItem(legacyKey);
          const converted = raw === null ? undefined : convert(raw);
          if (converted !== undefined) storage.setItem(currentKey, converted);
        }
      } catch {
        // A blocked/quota-limited store must not prevent the app from starting.
      } finally {
        try {
          storage.removeItem(legacyKey);
        } catch {
          // The persisted marker still prevents a later import of this key.
        }
      }
    }
  } catch {
    // Includes storage access and marker writes. No token copy has occurred.
  }
}
