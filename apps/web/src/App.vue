<script setup lang="ts">
import 'vue-sonner/style.css';
import { Toaster } from 'vue-sonner';
import { ref } from 'vue';
import { parseAccountFlow } from './accountFlow';

const { app, flow, returnTo, theme, redirectTo } = parseAccountFlow(
  window.location.href,
);
if (redirectTo) window.location.replace(redirectTo);
const succeeded = ref(false);
const dark =
  theme === 'dark' ||
  (theme !== 'light' &&
    window.matchMedia('(prefers-color-scheme: dark)').matches);
document.documentElement.classList.toggle('dark', dark);

function onSuccess() {
  if (returnTo) {
    window.location.replace(returnTo.href);
    return;
  }
  succeeded.value = true;
}
</script>

<template>
  <Toaster position="top-center" />
  <!-- Keep an explicit scrollport so every field remains reachable with a keyboard. -->
  <div
    class="bg-surface text-on-surface flex h-dvh w-full flex-col overflow-x-hidden overflow-y-auto overscroll-y-contain px-8 pt-[10vh] pb-8"
  >
    <div class="mx-auto flex w-full max-w-md shrink-0 flex-col gap-4">
      <img
        class="m-auto mt-0 mb-0 aspect-square w-1/2 max-w-[200px] select-none"
        src="https://books.kotoban.top/files-extra/girl.6e4fe22c238737fd028247f8f0cfd4ee.webp"
        alt=""
      />

      <div
        v-if="succeeded"
        role="status"
        class="flex flex-col gap-2 text-center"
      >
        <h1 class="text-xl font-bold">
          {{ flow === 'register' ? '注册成功' : '密码重置成功' }}
        </h1>
        <p>请返回应用登录。</p>
      </div>
      <template v-else>
        <h1 class="text-center text-xl font-bold">
          {{ flow === 'register' ? '注册' : '重置密码' }}
        </h1>
        <FormRegister
          v-if="flow === 'register'"
          :app="app"
          @success="onSuccess"
        />
        <FormResetPassword v-else @success="onSuccess" />
      </template>
    </div>
  </div>
</template>
