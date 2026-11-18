<script setup lang="ts">
import { ref } from 'vue'
import { useRoomStore } from '../stores/room'
import { useI18n } from '../lib/i18n'
import LanguageSelector from './LanguageSelector.vue'
import RoomSettingsDialog from './RoomSettingsDialog.vue'
const emit = defineEmits<{ share: [] }>()
const store = useRoomStore()
const { t } = useI18n()
const settingsOpen = ref(false)
</script>
<template>
  <header class="room-header">
    <div class="room-identity"><div class="room-avatar"><i aria-hidden="true" class="mdi mdi-radio" /></div><div><h1>{{ store.room?.name || t('appTitle') }}</h1><p><span :class="['connection-dot', { offline: !store.connected }]" />{{ t(store.connected ? 'connected' : store.reconnecting ? 'reconnecting' : 'loading') }}</p></div></div>
    <div class="header-actions">
      <LanguageSelector />
      <a v-if="store.session?.user.adminRole && store.updateNotice" class="header-action update-badge" :href="store.adminURL" :title="t('update', { version: store.updateNotice.latestVersion })" :aria-label="t('update', { version: store.updateNotice.latestVersion })"><i aria-hidden="true" class="mdi mdi-update" /><span>{{ t('update', { version: store.updateNotice.latestVersion }) }}</span></a>
      <button class="header-action presence-button" :aria-label="t('members')" @click="store.infoOpen = true"><i aria-hidden="true" class="mdi mdi-account-multiple-outline" /><span>{{ t('onlineCount', { count: store.onlineCount }) }}</span></button>
      <button class="header-action" :aria-label="t('share')" @click="emit('share')"><i aria-hidden="true" class="mdi mdi-share-variant-outline" /><span>{{ t('share') }}</span></button>
      <button v-if="store.canManage" class="header-action" :aria-label="t('settings')" @click="settingsOpen = true"><i aria-hidden="true" class="mdi mdi-cog-outline" /></button>
      <a v-if="store.deepLink" class="header-action app-link" :href="store.deepLink" :aria-label="t('musicMate')"><i aria-hidden="true" class="mdi mdi-cellphone-play" /><span>MusicMate</span></a>
      <button class="user-avatar" :aria-label="t('profile')" @click="store.profileUsername = store.session?.user.username || ''">{{ (store.session?.user.displayName || store.session?.user.username || '?').slice(0, 1).toUpperCase() }}</button>
    </div>
    <RoomSettingsDialog :open="settingsOpen" @close="settingsOpen = false" />
  </header>
</template>
