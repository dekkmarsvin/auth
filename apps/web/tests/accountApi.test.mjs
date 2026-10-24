import assert from 'node:assert/strict';
import test from 'node:test';
import { Api } from '../src/data/api.ts';

test('registration, OTP and reset preserve the existing API wire contract', async (t) => {
  const requests = [];
  t.mock.method(globalThis, 'fetch', async (url, options) => {
    requests.push({
      url,
      method: options.method,
      body: JSON.parse(options.body),
    });
    assert.equal(options.headers['Content-Type'], 'application/json');
    assert.ok(options.signal instanceof AbortSignal);
    return new Response('');
  });
  const registration = {
    app: 'n',
    username: 'fixture',
    password: 'fixture-password',
    email: 'fixture@example.test',
    otp: '123456',
  };
  await Api.register(registration);
  const otp = {
    email: registration.email,
    type: 'reset_password',
    turnstileToken: 'fixture-token',
  };
  await Api.requestOtp(otp);
  const reset = {
    email: registration.email,
    password: 'new-fixture-password',
    otp: '654321',
  };
  await Api.resetPassword(reset);
  assert.deepEqual(requests, [
    { url: '/api/v1/auth/register', method: 'POST', body: registration },
    { url: '/api/v1/auth/otp/request', method: 'POST', body: otp },
    { url: '/api/v1/auth/password/reset', method: 'POST', body: reset },
  ]);
});

test('a rejected OTP cannot be reported as a successful send', async (t) => {
  t.mock.method(
    globalThis,
    'fetch',
    async () => new Response('fixture verification rejected', { status: 400 }),
  );
  await assert.rejects(
    Api.requestOtp({
      email: 'fixture@example.test',
      type: 'verify',
      turnstileToken: 'invalid-fixture',
    }),
    (error) => error === '[400] fixture verification rejected',
  );
});
