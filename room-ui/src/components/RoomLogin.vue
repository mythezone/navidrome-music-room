<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoomStore } from '../stores/room'

const store = useRoomStore()
const username = ref('')
const password = ref('')
const showPassword = ref(false)
const submitting = computed(() => store.busy || !username.value.trim() || !password.value)

function submit(): void {
  if (submitting.value) return
  void store.login(username.value, password.value)
}
</script>

<template>
  <main class="login-shell">
    <section class="login-art" aria-hidden="true">
      <div class="record-orbit"><i class="mdi mdi-access-point" /></div>
    </section>
    <section class="login-panel" aria-labelledby="login-title">
      <div class="room-mark"><i class="mdi mdi-headphones" /></div>
      <h1 id="login-title">使用 Navidrome 加入房间</h1>
      <p>每位听众使用自己的账号和音乐库权限，密码只提交给当前 Navidrome。</p>
      <form @submit.prevent="submit">
        <label>
          <span>用户名</span>
          <span class="field-wrap"><i class="mdi mdi-account-outline" /><input v-model="username" autocomplete="username" autofocus required /></span>
        </label>
        <label>
          <span>密码</span>
          <span class="field-wrap"><i class="mdi mdi-lock-outline" /><input v-model="password" :type="showPassword ? 'text' : 'password'" autocomplete="current-password" required /><button type="button" class="icon-button" :aria-label="showPassword ? '隐藏密码' : '显示密码'" @click="showPassword = !showPassword"><i :class="showPassword ? 'mdi mdi-eye-off-outline' : 'mdi mdi-eye-outline'" /></button></span>
        </label>
        <p v-if="store.error" class="inline-error" role="alert">{{ store.error }}</p>
        <button class="primary-button login-button" type="submit" :disabled="submitting">
          <span v-if="store.busy" class="spinner" />
          {{ store.busy ? '正在验证' : '登录并加入' }}
        </button>
      </form>
      <small>邀请码不会进入服务器访问日志；兑换成功后会从地址栏移除。</small>
    </section>
  </main>
</template>
