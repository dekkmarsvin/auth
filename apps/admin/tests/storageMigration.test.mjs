import assert from 'node:assert/strict';
import test from 'node:test';
import vm from 'node:vm';

import { migrateLegacyAdminStorage } from '../storageMigration.js';
import { adminThemeBootstrapPlugin } from '../storageMigrationPlugin.js';

const marker = 'auth-admin:web-kit-storage-migration:v1';
const sessionKey = 'web-kit:session:v1';
const themeKey = 'web-kit:theme:v1';

function session(overrides = {}, adminMode = true) {
  const now = Math.floor(Date.now() / 1000);
  const payload = {
    uid: 1,
    sub: '管理員',
    role: 'admin',
    crat: now - 1000,
    iat: now,
    exp: now + 3600,
    ...overrides,
  };
  const token = `header.${Buffer.from(JSON.stringify(payload)).toString('base64url')}.signature`;
  return JSON.stringify({ token, adminMode });
}

function fakeStorage(entries = []) {
  const values = new Map(entries);
  return {
    values,
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, value),
    removeItem: (key) => values.delete(key),
  };
}

test('migrates a valid administrator and bare-string theme once', () => {
  const original = session();
  const storage = fakeStorage([
    ['auth-admin-session', original],
    ['auth-admin-theme', 'dark'],
  ]);
  migrateLegacyAdminStorage(storage);
  assert.equal(storage.getItem(sessionKey), original);
  assert.equal(storage.getItem(themeKey), '"dark"');
  assert.equal(storage.getItem(marker), 'attempted');
  assert.equal(storage.getItem('auth-admin-session'), null);
  assert.equal(storage.getItem('auth-admin-theme'), null);
});

test('current keys take precedence even when malformed', () => {
  const storage = fakeStorage([
    [sessionKey, 'current-invalid-session'],
    [themeKey, 'current-invalid-theme'],
    ['auth-admin-session', session()],
    ['auth-admin-theme', 'dark'],
  ]);
  migrateLegacyAdminStorage(storage);
  assert.equal(storage.getItem(sessionKey), 'current-invalid-session');
  assert.equal(storage.getItem(themeKey), 'current-invalid-theme');
  assert.equal(storage.getItem('auth-admin-session'), null);
});

test('logout and later stale legacy writes never resurrect the old token', () => {
  const storage = fakeStorage([['auth-admin-session', session()]]);
  migrateLegacyAdminStorage(storage);
  storage.removeItem(sessionKey);
  storage.setItem('auth-admin-session', session());
  migrateLegacyAdminStorage(storage);
  assert.equal(storage.getItem(sessionKey), null);
});

for (const [label, raw] of [
  ['malformed JSON', 'not JSON'],
  ['malformed token', JSON.stringify({ token: 'invalid' })],
  ['expired token', session({ exp: 1 })],
  ['missing uid', session({ uid: undefined })],
  ['unknown role', session({ role: 'superadmin' })],
]) {
  test(`${label} is retired without becoming the current session`, () => {
    const storage = fakeStorage([['auth-admin-session', raw]]);
    migrateLegacyAdminStorage(storage);
    assert.equal(storage.getItem(sessionKey), null);
    assert.equal(storage.getItem('auth-admin-session'), null);
    assert.equal(storage.getItem(marker), 'attempted');
  });
}

test('non-admin legacy users cannot retain admin mode', () => {
  const storage = fakeStorage([
    ['auth-admin-session', session({ role: 'member' })],
  ]);
  migrateLegacyAdminStorage(storage);
  assert.equal(JSON.parse(storage.getItem(sessionKey)).adminMode, false);
});

test('marker write failure prevents copying; read failures never throw', () => {
  const storage = fakeStorage([['auth-admin-session', session()]]);
  storage.setItem = () => {
    throw new Error('quota');
  };
  assert.doesNotThrow(() => migrateLegacyAdminStorage(storage));
  assert.equal(storage.getItem(sessionKey), null);
  assert.equal(storage.getItem(marker), null);
  assert.doesNotThrow(() =>
    migrateLegacyAdminStorage({
      getItem() {
        throw new Error('blocked');
      },
    }),
  );
  assert.doesNotThrow(() => migrateLegacyAdminStorage(undefined));
});

test('partial copy failure keeps a marker and permits independent theme migration', () => {
  const storage = fakeStorage([
    ['auth-admin-session', session()],
    ['auth-admin-theme', 'light'],
  ]);
  const write = storage.setItem;
  storage.setItem = (key, value) => {
    if (key === sessionKey) throw new Error('quota');
    write(key, value);
  };
  migrateLegacyAdminStorage(storage);
  assert.equal(storage.getItem(sessionKey), null);
  assert.equal(storage.getItem(themeKey), '"light"');
  assert.equal(storage.getItem(marker), 'attempted');
});

test('failure to remove legacy keys still cannot revive a logged-out session', () => {
  const storage = fakeStorage([['auth-admin-session', session()]]);
  storage.removeItem = () => {
    throw new Error('blocked');
  };
  migrateLegacyAdminStorage(storage);
  storage.values.delete(sessionKey);
  migrateLegacyAdminStorage(storage);
  assert.equal(storage.getItem(sessionKey), null);
});

test('the actual bootstrap tag migrates before selecting the first-paint theme', () => {
  const storage = fakeStorage([['auth-admin-theme', 'dark']]);
  const plugin = adminThemeBootstrapPlugin();
  const tags = plugin.transformIndexHtml.handler();
  assert.equal(tags.length, 1);
  assert.equal(tags[0].injectTo, 'head-prepend');
  const root = { dataset: {} };
  vm.runInNewContext(tags[0].children, {
    window: { localStorage: storage },
    localStorage: storage,
    document: { documentElement: root },
    matchMedia: () => ({ matches: false }),
    atob,
    TextDecoder,
  });
  assert.equal(root.dataset.theme, 'dark');
  assert.equal(storage.getItem(themeKey), '"dark"');
});

test('blocked localStorage safely falls back to the upstream system theme', () => {
  const plugin = adminThemeBootstrapPlugin();
  const root = { dataset: {} };
  const browser = {};
  Object.defineProperty(browser, 'localStorage', {
    get() {
      throw new Error('blocked');
    },
  });
  vm.runInNewContext(plugin.transformIndexHtml.handler()[0].children, {
    window: browser,
    document: { documentElement: root },
    matchMedia: () => ({ matches: true }),
  });
  assert.equal(root.dataset.theme, 'dark');
});
