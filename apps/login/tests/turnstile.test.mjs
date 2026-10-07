import assert from 'node:assert/strict';
import test from 'node:test';

let moduleId = 0;
async function freshLoader() {
  return (await import(`../src/data/turnstile.ts?test=${++moduleId}`))
    .loadTurnstileSiteKey;
}

test('Turnstile widgets share one public configuration request', async (t) => {
  let requests = 0;
  t.mock.method(globalThis, 'fetch', async (url, options) => {
    requests++;
    assert.equal(url, '/api/v1/auth/config');
    assert.equal(options.cache, 'no-store');
    return Response.json({ turnstileSiteKey: ' public-site-key ' });
  });
  const load = await freshLoader();
  const keys = await Promise.all([load(), load(), load()]);
  assert.deepEqual(keys, Array(3).fill('public-site-key'));
  assert.equal(await load(), 'public-site-key');
  assert.equal(requests, 1);
});

test('failed or malformed public configuration leaves no usable key and can retry', async (t) => {
  const replies = [
    new Response('', { status: 503 }),
    Response.json({ turnstileSiteKey: '' }),
    Response.json({ turnstileSiteKey: 123 }),
    Response.json({ secret: 'unexpected-private-field' }),
    Response.json({ turnstileSiteKey: 'configured-public-key' }),
  ];
  t.mock.method(globalThis, 'fetch', async () => replies.shift());
  const load = await freshLoader();
  for (let i = 0; i < 4; i++) await assert.rejects(load());
  assert.equal(await load(), 'configured-public-key');
});
