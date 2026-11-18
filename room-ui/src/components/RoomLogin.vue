<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoomStore } from '../stores/room'
import { useI18n } from '../lib/i18n'
import LanguageSelector from './LanguageSelector.vue'
const store = useRoomStore()
const { t, errorText } = useI18n()
const username = ref('')
const password = ref('')
const showPassword = ref(false)
const submitting = computed(() => store.busy || !username.value.trim() || !password.value)
async function submit(): Promise<void> {
  if (submitting.value) return
  await store.login(username.value, password.value)
  password.value = ''
}
</script>
<template>
  <main class="login-shell">
    <section class="login-art" aria-hidden="true"><div class="record-orbit"><i aria-hidden="true" class="mdi mdi-access-point" /></div></section>
    <section class="login-panel" aria-labelledby="login-title">
      <LanguageSelector /><div class="room-mark"><i aria-hidden="true" class="mdi mdi-headphones" /></div>
      <h1 id="login-title">{{ t('loginTitle') }}</h1><p>{{ t('loginHint') }}</p>
      <form @submit.prevent="submit">
        <label><span>{{ t('username') }}</span><span class="field-wrap"><i aria-hidden="true" class="mdi mdi-account-outline" /><input v-model="username" autocomplete="username" required /></span></label>
        <label><span>{{ t('password') }}</span><span class="field-wrap"><i aria-hidden="true" class="mdi mdi-lock-outline" /><input v-model="password" :type="showPassword ? 'text' : 'password'" autocomplete="current-password" required /><button type="button" class="icon-button" :aria-label="t(showPassword ? 'hidePassword' : 'showPassword')" @click="showPassword = !showPassword"><i aria-hidden="true" :class="showPassword ? 'mdi mdi-eye-off-outline' : 'mdi mdi-eye-outline'" /></button></span></label>
        <p v-if="store.error" class="inline-error" role="alert">{{ errorText(store.errorCode) }}</p>
        <button class="primary-button login-button" type="submit" :disabled="submitting"><span v-if="store.busy" class="spinner" />{{ t(store.busy ? 'loading' : 'login') }}</button>
      </form>
    </section>
  </main>
</template>
