<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoomStore } from '../stores/room'

defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()
const store = useRoomStore()
const copied = ref(false)
const shareURL = computed(() => {
  const url = new URL(window.location.href)
  url.hash = store.invitation ? `invite=${encodeURIComponent(store.invitation)}` : ''
  return url.toString()
})
const adminURL = computed(() => {
  const match = window.location.pathname.match(/^(.*)\/join\/[a-f\d]{32}(?:\/.*)?$/i)
  return `${match?.[1] || ''}/admin/`
})

async function copy(): Promise<void> {
  await navigator.clipboard.writeText(shareURL.value)
  copied.value = true
  window.setTimeout(() => { copied.value = false }, 1800)
}
</script>

<template>
  <div v-if="open" class="dialog-backdrop" @click.self="emit('close')">
    <section class="dialog" role="dialog" aria-modal="true" aria-labelledby="share-title">
      <header><h2 id="share-title">分享房间</h2><button class="icon-button" aria-label="关闭" @click="emit('close')"><i class="mdi mdi-close" /></button></header>
      <p v-if="store.invitation">这个页面仍在内存中保留本次邀请密钥，复制后可继续分享；刷新页面后密钥会消失。</p>
      <p v-else>当前地址只适用于已经加入的成员。管理员可在插件管理页生成新的邀请链接和二维码。</p>
      <div class="share-address">{{ shareURL }}</div>
      <div class="dialog-actions">
        <a v-if="store.canManage && !store.invitation" class="secondary-button" :href="adminURL"><i class="mdi mdi-cog-outline" />打开房间管理</a>
        <button class="secondary-button" @click="copy"><i class="mdi mdi-content-copy" />{{ copied ? '已复制' : store.invitation ? '复制邀请链接' : '复制房间地址' }}</button>
        <a v-if="store.deepLink" class="primary-button" :href="store.deepLink"><i class="mdi mdi-cellphone-music" />在 MusicMate 中打开</a>
      </div>
    </section>
  </div>
</template>
