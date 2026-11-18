<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoomStore } from '../stores/room'
import { useI18n } from '../lib/i18n'
import RoomDialog from './RoomDialog.vue'
defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()
const store = useRoomStore()
const { t, errorText } = useI18n()
const copied = ref(false)
const shareURL = computed(() => {
  const url = new URL(window.location.href)
  url.hash = store.invitation ? `invite=${encodeURIComponent(store.invitation)}` : ''
  return url.toString()
})
async function copy(): Promise<void> {
  try { await navigator.clipboard.writeText(shareURL.value); copied.value = true }
  catch (cause) { store.failTransient(cause) }
}
</script>
<template>
  <RoomDialog :open="open" :title="t('shareRoom')" @close="emit('close')">
    <p>{{ t(store.invitation ? 'shareInviteHint' : 'shareMemberHint') }}</p>
    <div class="share-address">{{ shareURL }}</div>
    <p v-if="store.error" class="media-error" role="alert">{{ errorText(store.errorCode) }}</p><div class="dialog-actions">
      <a v-if="store.canManage && !store.invitation" class="secondary-button" :href="store.adminURL">{{ t('admin') }}</a>
      <button class="secondary-button" @click="copy"><i aria-hidden="true" class="mdi mdi-content-copy" />{{ t(copied ? 'copied' : 'copy') }}</button>
      <a v-if="store.deepLink" class="primary-button" :href="store.deepLink">{{ t('musicMate') }}</a>
    </div>
  </RoomDialog>
</template>
