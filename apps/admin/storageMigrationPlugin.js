import { readFileSync } from 'node:fs';
import { themeBootstrapPlugin } from '@novelia/web-kit/vite';

/** Keep migration and upstream first-paint theme selection in one ordered tag. */
export function adminThemeBootstrapPlugin() {
  const plugin = themeBootstrapPlugin();
  const transform = plugin.transformIndexHtml;
  // Read the self-contained function rather than using Function.toString():
  // config bundling can insert helper references that do not exist in the page.
  const source = readFileSync(
    new URL('./storageMigration.js', import.meta.url),
    'utf8',
  ).replace(/^export (?=function migrateLegacyAdminStorage\()/m, '');
  const migration = `try { (${source})(window.localStorage); } catch {}`;
  return {
    ...plugin,
    name: 'auth-admin-storage-theme-bootstrap',
    transformIndexHtml: {
      ...transform,
      handler(...args) {
        return transform.handler(...args).map((tag) => ({
          ...tag,
          children: `${migration}\n${tag.children}`,
        }));
      },
    },
  };
}
