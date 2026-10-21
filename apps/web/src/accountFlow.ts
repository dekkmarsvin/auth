/** Deployment destinations, matched by exact origin rather than suffix. */
const returnOrigins = new Set(['https://books.kotoban.top']);

export function validateReturnTo(
  raw: string | null,
  currentOrigin: string,
): URL | null {
  if (!raw) return null;
  try {
    const target = new URL(raw);
    if (target.username || target.password) return null;
    if (target.origin === currentOrigin) {
      return target.protocol === 'https:' || target.protocol === 'http:'
        ? target
        : null;
    }
    return returnOrigins.has(target.origin) ? target : null;
  } catch {
    return null;
  }
}

export function parseAccountFlow(href: string) {
  const url = new URL(href);
  const requestedFlow = url.pathname.replace(/\/+$/, '').split('/').pop();
  const flow =
    requestedFlow === 'reset-password' ? 'reset_password' : 'register';
  let redirectTo: string | undefined;
  if (requestedFlow !== 'register' && requestedFlow !== 'reset-password') {
    url.pathname = '/register';
    redirectTo = url.href;
  }
  return {
    app: url.searchParams.get('app') || 'auth',
    flow,
    returnTo: validateReturnTo(url.searchParams.get('returnTo'), url.origin),
    theme: url.searchParams.get('theme'),
    redirectTo,
  };
}
