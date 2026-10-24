/** Options accepted by `turnstile.render`. */
interface TurnstileRenderOptions {
  sitekey: string;
  /** `flexible` stretches the widget to the container width. */
  size?: 'normal' | 'compact' | 'flexible';
  theme?: 'light' | 'dark' | 'auto';
  language?: string;
  action?: string;
  callback?: (token: string) => void;
  'error-callback'?: (errorCode?: string) => void;
  'expired-callback'?: () => void;
}

interface TurnstileApi {
  render(container: HTMLElement, options: TurnstileRenderOptions): string;
  remove(widgetId: string): void;
  reset(widgetId?: string): void;
}

declare global {
  interface Window {
    turnstile?: TurnstileApi;
  }
}

/**
 * Shared loader for the Cloudflare Turnstile script.
 *
 * The script is loaded once per page, no matter how many widgets mount. A
 * failed load is retried on the next mount instead of being cached forever.
 */

const SCRIPT_ID = 'cf-turnstile-script';
const SCRIPT_SRC =
  'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';

/** In-flight load, shared by every widget on the page. */
let pending: Promise<void> | undefined;

let siteKeyPromise: Promise<string> | undefined;

/** Fetch the deployment's public key once; failures may be retried. */
export function loadTurnstileSiteKey(): Promise<string> {
  if (siteKeyPromise) return siteKeyPromise;
  siteKeyPromise = (async () => {
    const response = await fetch('/api/v1/auth/config', {
      cache: 'no-store',
      signal: AbortSignal.timeout(10000),
    });
    if (!response.ok) throw new Error('Turnstile configuration is unavailable');
    const config: unknown = await response.json();
    if (
      !config ||
      typeof config !== 'object' ||
      !('turnstileSiteKey' in config) ||
      typeof config.turnstileSiteKey !== 'string' ||
      config.turnstileSiteKey.trim() === ''
    ) {
      throw new Error('Turnstile site key is unavailable');
    }
    return config.turnstileSiteKey.trim();
  })().catch((error) => {
    siteKeyPromise = undefined;
    throw error;
  });
  return siteKeyPromise;
}

export function loadTurnstileScript(): Promise<void> {
  if (window.turnstile) return Promise.resolve();
  if (pending) return pending;

  pending = new Promise<void>((resolve, reject) => {
    const script = document.createElement('script');
    script.id = SCRIPT_ID;
    script.src = SCRIPT_SRC;
    script.async = true;
    script.defer = true;
    const timeout = setTimeout(onError, 10000);

    function cleanup() {
      clearTimeout(timeout);
      script.removeEventListener('load', onLoad);
      script.removeEventListener('error', onError);
      pending = undefined;
    }

    function onLoad() {
      if (!window.turnstile) {
        onError();
        return;
      }
      cleanup();
      resolve();
    }

    function onError() {
      cleanup();
      script.remove();
      reject(new Error('Failed to load the Turnstile script'));
    }

    script.addEventListener('load', onLoad);
    script.addEventListener('error', onError);
    document.head.appendChild(script);
  });

  return pending;
}
