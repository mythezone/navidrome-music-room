<script setup lang="ts">
import { ref, watch } from 'vue'
import RoomDialog from './RoomDialog.vue'
import { useRoomStore } from '../stores/room'
import { useI18n } from '../lib/i18n'
import type { RoomSettings } from '../types'
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()
const store = useRoomStore()
const { t, errorText } = useI18n()
const saving = ref(false)
const form = ref<RoomSettings>({ name: '', allowMemberSkip: false, playbackMode: 'fifo', queueLimit: 5, preloadNextTrack: false })
watch(() => props.open, open => {
  if (open && store.room) {
    store.error = ''
    const { name, allowMemberSkip, playbackMode, queueLimit, preloadNextTrack } = store.room
    form.value = { name, allowMemberSkip: Boolean(allowMemberSkip), playbackMode, queueLimit, preloadNextTrack }
  }
})
async function save(): Promise<void> {
  saving.value = true
  try { if (await store.updateRoomSettings({ ...form.value, name: form.value.name.trim() })) emit('close') }
  finally { saving.value = false }
}
</script>
<template>
  <RoomDialog :open="open" :title="t('settings')" @close="emit('close')">
    <form class="dialog-form" @submit.prevent="save">
      <label>{{ t('roomName') }}<input v-model="form.name" required maxlength="80" /></label>
      <label class="toggle-label"><input v-model="form.allowMemberSkip" type="checkbox" />{{ t('allowSkip') }}</label>
      <label>{{ t('playbackMode') }}<select v-model="form.playbackMode"><option value="fifo">{{ t('fifo') }}</option><option value="fair_random">{{ t('fair_random') }}</option></select></label>
      <label>{{ t('queueLimit') }}<input v-model.number="form.queueLimit" type="number" min="1" max="100" required /></label>
      <label class="toggle-label"><input v-model="form.preloadNextTrack" type="checkbox" />{{ t('preload') }}</label>
      <p v-if="store.error" class="media-error" role="alert">{{ errorText(store.errorCode) }}</p><div class="dialog-actions"><a class="secondary-button" :href="store.adminURL">{{ t('admin') }}</a><button class="primary-button" :disabled="saving || !store.canManage || !form.name.trim()">{{ t(saving ? 'loading' : 'save') }}</button></div>
    </form>
  </RoomDialog>
</template>
