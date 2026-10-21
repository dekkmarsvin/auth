import assert from 'node:assert/strict';
import test from 'node:test';
import { parseAccountFlow, validateReturnTo } from '../src/accountFlow.ts';

const origin = 'https://auth.kotoban.top';

test('registration and reset preserve the caller, theme and exact return page', () => {
  const returnTo = 'https://books.kotoban.top/novel/p/n?lang=zh#volume-2';
  for (const [path, flow] of [
    ['register', 'register'],
    ['reset-password', 'reset_password'],
  ]) {
    const url = new URL(`/auth-proxy/${path}/`, origin);
    url.search = new URLSearchParams({
      app: 'n',
      theme: 'dark',
      returnTo,
    }).toString();
    const parsed = parseAccountFlow(url.href);
    assert.equal(parsed.app, 'n');
    assert.equal(parsed.flow, flow);
    assert.equal(parsed.theme, 'dark');
    assert.equal(parsed.returnTo.href, returnTo);
    assert.equal(parsed.redirectTo, undefined);
  }
});

test('the standalone entry canonicalizes to registration and preserves query values', () => {
  const result = parseAccountFlow(`${origin}/?app=auth&theme=light`);
  assert.equal(result.flow, 'register');
  assert.equal(result.redirectTo, `${origin}/register?app=auth&theme=light`);
  assert.equal(result.returnTo, null);
  assert.equal(parseAccountFlow(`${origin}/register`).app, 'auth');
});

test('only same-origin HTTP(S) or the exact production books origin can receive return navigation', () => {
  for (const allowed of [
    `${origin}/admin/?tab=users`,
    'https://books.kotoban.top/workspace/gpt-pipeline',
  ])
    assert.equal(validateReturnTo(allowed, origin)?.href, allowed);
  assert.equal(
    validateReturnTo('http://localhost:5173/admin/', 'http://localhost:5173')
      ?.href,
    'http://localhost:5173/admin/',
  );
  for (const rejected of [
    null,
    '',
    '/workspace',
    '//books.kotoban.top/',
    'javascript:alert(1)',
    'data:text/html,test',
    'http://books.kotoban.top/',
    'https://books.kotoban.top:444/',
    'https://books.kotoban.top.evil.example/',
    'https://sub.books.kotoban.top/',
    'https://novelia.cc/',
    'https://n.novelia.cc/',
    'https://books.kotoban.top@evil.example/',
    'https://user:password@books.kotoban.top/',
  ])
    assert.equal(validateReturnTo(rejected, origin), null, String(rejected));
});
