<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch, watchEffect } from 'vue'
import CatalogPanel from './components/CatalogPanel.vue'
import FavoritesPanel from './components/FavoritesPanel.vue'
import HistoryPanel from './components/HistoryPanel.vue'
import ChatPanel from './components/ChatPanel.vue'
import NowPlaying from './components/NowPlaying.vue'
import QueuePanel from './components/QueuePanel.vue'
import RoomHeader from './components/RoomHeader.vue'
import RoomLogin from './components/RoomLogin.vue'
import ShareDialog from './components/ShareDialog.vue'
import LanguageSelector from './components/LanguageSelector.vue'
import RoomInfoDialog from './components/RoomInfoDialog.vue'
import MemberProfileDialog from './components/MemberProfileDialog.vue'
import { routeContext } from './lib/location'
import { clamp } from './lib/sync'
import { useI18n } from './lib/i18n'
import { useRoomStore } from './stores/room'

const store = useRoomStore()
const { t, locale, errorText } = useI18n()
const shareOpen = ref(false)
const content = ref<HTMLElement>()
const split = ref(40)
try { const saved = Number(localStorage.getItem('nmr:split:v1')); if (saved) split.value = clamp(saved, 28, 60) } catch { /* Use the default layout. */ }
const dragging = ref(false)
const tabs = computed(() => [
  { value: 'playing' as const, icon: 'mdi-disc-player' }, { value: 'queue' as const, icon: 'mdi-playlist-music' },
  { value: 'history' as const, icon: 'mdi-history' }, { value: 'catalog' as const, icon: 'mdi-magnify' },
  { value: 'favorites' as const, icon: 'mdi-heart-outline' }, { value: 'chat' as const, icon: 'mdi-message-text-outline' },
])
watchEffect(() => {
  document.title = `${store.room?.name ? `${store.room.name} · ` : ''}${t('appTitle')} · Navidrome`
  document.documentElement.lang = locale.value
})
function updateChatVisibility(): void { store.chatVisible = store.activeTab === 'chat' && !store.immersive && document.visibilityState !== 'hidden' }
watch(() => [store.activeTab, store.immersive], updateChatVisibility, { immediate: true })
function persistSplit(): void { try { localStorage.setItem('nmr:split:v1', String(split.value)) } catch { /* Keep the layout in memory. */ } }
function moveSplit(event: PointerEvent): void {
  if (!dragging.value || !content.value) return
  const rect = content.value.getBoundingClientRect()
  split.value = clamp((event.clientX - rect.left) / rect.width * 100, 28, 60)
}
function startSplit(event: PointerEvent): void {
  if (event.button !== 0) return
  dragging.value = true
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}
function endSplit(): void { if (dragging.value) { dragging.value = false; persistSplit() } }
function keySplit(event: KeyboardEvent): void {
  const values: Record<string, number> = { ArrowLeft: split.value - 2, ArrowRight: split.value + 2, Home: 28, End: 60 }
  if (event.key in values) { event.preventDefault(); split.value = clamp(values[event.key], 28, 60); persistSplit() }
}
watch(() => store.immersive, enabled => {
  if (enabled) void document.documentElement.requestFullscreen?.().catch(() => {})
  else if (document.fullscreenElement) void document.exitFullscreen?.().catch(() => {})
})
function fullscreenChanged(): void { if (!document.fullscreenElement) store.immersive = false }
function escape(event: KeyboardEvent): void { if (event.key === 'Escape' && !document.querySelector('dialog[open]')) store.immersive = false }
onMounted(() => {
  if (!window.matchMedia('(max-width: 820px)').matches) store.activeTab = 'queue'
  store.configure(routeContext())
  void store.start()
  document.addEventListener('visibilitychange', updateChatVisibility)
  document.addEventListener('fullscreenchange', fullscreenChanged)
  document.addEventListener('keydown', escape)
})
onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', updateChatVisibility)
  document.removeEventListener('fullscreenchange', fullscreenChanged)
  document.removeEventListener('keydown', escape)
  store.disconnect()
})
</script>

<template>
  <RoomLogin v-if="store.phase === 'login'" />
  <main v-else-if="store.phase === 'error'" class="state-shell">
    <div class="state-card error-state">
      <LanguageSelector />
      <span class="state-icon"><i aria-hidden="true" class="mdi mdi-link-variant-off" /></span>
      <h1>{{ t('unavailable') }}</h1><p>{{ errorText(store.errorCode) }}</p>
      <div class="toolbar"><button class="secondary-button" @click="store.start">{{ t('retry') }}</button><a class="primary-button" href="/app/">{{ t('backNavidrome') }}</a></div>
    </div>
  </main>
  <main v-else-if="store.phase !== 'ready'" class="state-shell">
    <div class="state-card loading-state"><LanguageSelector /><span class="room-mark pulse"><i aria-hidden="true" class="mdi mdi-headphones" /></span><h1>{{ t(store.phase === 'joining' ? 'joining' : 'loading') }}</h1><span class="spinner large" /></div>
  </main>
  <div v-else :class="['room-shell', { immersive: store.immersive }]" :data-connected="store.connected" :data-reconnecting="store.reconnecting" :data-revision="store.playback?.revision || 0">
    <RoomHeader v-if="!store.immersive" @share="shareOpen = true" />
    <div v-if="store.reconnecting" class="connection-banner" role="status"><span class="spinner" />{{ t('reconnecting') }}</div>
    <div v-if="store.error" class="toast error-toast" role="alert">{{ errorText(store.errorCode) }}</div>
    <div v-if="store.notice" class="toast notice-toast" role="status">{{ store.notice }}</div>
    <button v-if="store.immersive" class="secondary-button exit-immersive" @click="store.immersive = false"><i aria-hidden="true" class="mdi mdi-fullscreen-exit" />{{ t('exitImmersive') }}</button>
    <main ref="content" class="room-content" :style="{ '--player-width': `${split}%` }">
      <div :class="['player-pane', { 'mobile-active': store.activeTab === 'playing' }]"><NowPlaying /></div>
      <div v-if="!store.immersive" class="split-divider" role="separator" tabindex="0" aria-orientation="vertical" :aria-label="t('resize')" :aria-valuenow="Math.round(split)" :aria-valuemin="28" :aria-valuemax="60" @pointerdown="startSplit" @pointermove="moveSplit" @pointerup="endSplit" @pointercancel="endSplit" @lostpointercapture="endSplit" @keydown="keySplit" />
      <section v-if="!store.immersive" :class="['workspace', { 'mobile-hidden': store.activeTab === 'playing' }]">
        <nav class="desktop-tabs" :aria-label="t('appTitle')">
          <button v-for="tab in tabs.filter(item => item.value !== 'playing')" :key="tab.value" :class="{ active: store.activeTab === tab.value }" :aria-pressed="store.activeTab === tab.value" @click="store.activeTab = tab.value"><i aria-hidden="true" :class="`mdi ${tab.icon}`" />{{ t(tab.value) }}<span v-if="tab.value === 'chat' && store.unreadChatCount" class="unread-badge">{{ store.unreadChatCount }}</span></button>
        </nav>
        <QueuePanel v-if="store.activeTab === 'queue' || store.activeTab === 'playing'" />
        <HistoryPanel v-else-if="store.activeTab === 'history'" />
        <CatalogPanel v-else-if="store.activeTab === 'catalog'" />
        <FavoritesPanel v-else-if="store.activeTab === 'favorites'" />
        <ChatPanel v-else-if="store.activeTab === 'chat'" />
      </section>
    </main>
    <nav v-if="!store.immersive" class="mobile-tabs" :aria-label="t('appTitle')">
      <button v-for="tab in tabs" :key="tab.value" :class="{ active: store.activeTab === tab.value }" :aria-pressed="store.activeTab === tab.value" @click="store.activeTab = tab.value"><span class="nav-icon"><i aria-hidden="true" :class="`mdi ${tab.icon}`" /><span v-if="tab.value === 'chat' && store.unreadChatCount" class="unread-badge">{{ store.unreadChatCount }}</span></span><span>{{ t(tab.value) }}</span></button>
    </nav>
    <ShareDialog :open="shareOpen" @close="shareOpen = false" />
    <RoomInfoDialog />
    <MemberProfileDialog />
  </div>
</template>
