import assert from 'node:assert/strict';
import test from 'node:test';
import * as Vue from 'vue';
import { compileComponent } from '../../../packages/web-kit/tests/helpers/compile-component.mjs';
import {
  createTestRenderer,
  dispatch,
} from '../../../packages/web-kit/tests/helpers/test-renderer.mjs';
import { parseAccountFlow } from '../src/accountFlow.ts';

const App = compileComponent(new URL('../src/App.vue', import.meta.url), {
  vue: Vue,
  'vue-sonner': { Toaster: { render: () => null } },
  'vue-sonner/style.css': {},
  './accountFlow': { parseAccountFlow },
});

function descendants(node, predicate) {
  return [
    node,
    ...(node.children ?? []).flatMap((child) => descendants(child, predicate)),
  ].filter(predicate);
}

function mockGlobal(t, key, value) {
  const previous = Object.getOwnPropertyDescriptor(globalThis, key);
  Object.defineProperty(globalThis, key, { configurable: true, value });
  t.after(() => {
    if (previous) Object.defineProperty(globalThis, key, previous);
    else delete globalThis[key];
  });
}

for (const [path, expectedForm] of [
  ['register', 'register'],
  ['reset-password', 'reset'],
]) {
  test(`${path} mounts its own form and returns to the current books page only after success`, async (t) => {
    const returned = [];
    const returnTo =
      'https://books.kotoban.top/workspace/gpt-pipeline?job=fixture#queue';
    const href = new URL(`https://auth.kotoban.top/${path}`);
    href.searchParams.set('app', 'n');
    href.searchParams.set('returnTo', returnTo);
    mockGlobal(t, 'window', {
      location: { href: href.href, replace: (url) => returned.push(url) },
      matchMedia: () => ({ matches: false }),
    });
    mockGlobal(t, 'document', {
      documentElement: { classList: { toggle() {} } },
    });
    const host = createTestRenderer(t);
    const app = host.createApp(App);
    for (const [component, flow] of [
      ['FormRegister', 'register'],
      ['FormResetPassword', 'reset'],
    ]) {
      app.component(component, {
        props: ['app'],
        emits: ['success'],
        setup(props, { emit }) {
          return () =>
            Vue.h(
              'button',
              {
                'data-flow': flow,
                'data-app': props.app,
                onClick: () => emit('success'),
              },
              flow,
            );
        },
      });
    }
    const { container } = host.mount(app);
    const button = descendants(container, (node) => node.props['data-flow'])[0];
    assert.equal(button.props['data-flow'], expectedForm);
    if (path === 'register') assert.equal(button.props['data-app'], 'n');
    assert.deepEqual(returned, []);
    dispatch(button, 'click');
    await Vue.nextTick();
    assert.deepEqual(returned, [returnTo]);
  });
}

test('an untrusted return destination stays on the success page', async (t) => {
  const returned = [];
  mockGlobal(t, 'window', {
    location: {
      href: 'https://auth.kotoban.top/register?returnTo=https%3A%2F%2Fbooks.kotoban.top.evil.example%2F',
      replace: (url) => returned.push(url),
    },
    matchMedia: () => ({ matches: false }),
  });
  mockGlobal(t, 'document', {
    documentElement: { classList: { toggle() {} } },
  });
  const host = createTestRenderer(t);
  const app = host.createApp(App);
  app.component('FormRegister', {
    emits: ['success'],
    setup(_props, { emit }) {
      return () =>
        Vue.h('button', { onClick: () => emit('success') }, 'register');
    },
  });
  app.component('FormResetPassword', { render: () => null });
  const { container } = host.mount(app);
  dispatch(descendants(container, (node) => node.tag === 'button')[0], 'click');
  await Vue.nextTick();
  assert.deepEqual(returned, []);
  assert.ok(
    descendants(container, (node) => node.props.role === 'status').length,
  );
});
