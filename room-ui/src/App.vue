<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import CatalogPanel from './components/CatalogPanel.vue'
import FavoritesPanel from './components/FavoritesPanel.vue'
import HistoryPanel from './components/HistoryPanel.vue'
import ComingSoonPanel from './components/ComingSoonPanel.vue'
import NowPlaying from './components/NowPlaying.vue'
import QueuePanel from './components/QueuePanel.vue'
import RoomHeader from './components/RoomHeader.vue'
import RoomLogin from './components/RoomLogin.vue'
import ShareDialog from './components/ShareDialog.vue'
import { routeContext } from './lib/location'
import { useRoomStore } from './stores/room'

const store = useRoomStore()
const shareOpen = ref(false)
type TabName = 'playing' | 'queue' | 'history' | 'catalog' | 'favorites' | 'chat'
const tabs: ReadonlyArray<{ value: TabName; label: string; icon: string; planned?: boolean }> = [
  { value: 'playing', label: '播放', icon: 'mdi-disc-player' },
  { value: 'queue', label: '待播放', icon: 'mdi-playlist-music' },
  { value: 'history', label: '历史', icon: 'mdi-history' },
  { value: 'catalog', label: '点歌台', icon: 'mdi-magnify' },
  { value: 'favorites', label: '收藏', icon: 'mdi-heart-outline' },
  { value: 'chat', label: '群聊', icon: 'mdi-message-text-outline', planned: true },
]

const statusCopy = computed(() => {
  if (store.phase === 'booting') return '正在读取房间链接'
  if (store.phase === 'joining') return '正在验证成员资格并同步房间'
  return '正在准备听歌房'
})

onMounted(() => {
  if (!window.matchMedia('(max-width: 820px)').matches) store.activeTab = 'queue'
  const context = routeContext()
  if (context) store.configure(context)
  void store.start()
})
</script>

<template>
  <RoomLogin v-if="store.phase === 'login'" />

  <main v-else-if="store.phase === 'error'" class="state-shell">
    <div class="state-card error-state">
      <span class="state-icon"><i class="mdi mdi-link-variant-off" /></span>
      <span class="section-kicker">{{ store.errorCode || 'ROOM_UNAVAILABLE' }}</span>
      <h1>暂时无法进入这个房间</h1>
      <p>{{ store.error }}</p>
      <a class="primary-button" href="/app/">返回 Navidrome</a>
    </div>
  </main>

  <main v-else-if="store.phase !== 'ready'" class="state-shell">
    <div class="state-card loading-state"><span class="room-mark pulse"><i class="mdi mdi-headphones" /></span><h1>{{ statusCopy }}</h1><p>登录信息和邀请码不会写入房间数据文件。</p><span class="spinner large" /></div>
  </main>

  <div v-else class="room-shell" :data-connected="store.connected" :data-reconnecting="store.reconnecting" :data-revision="store.playback?.revision || 0">
    <RoomHeader @share="shareOpen = true" />
    <div v-if="store.reconnecting" class="connection-banner"><span class="spinner" /> 网络已中断，正在恢复快照和实时连接…</div>
    <div v-if="store.error" class="toast error-toast" role="alert"><i class="mdi mdi-alert-circle-outline" /> {{ store.error }}</div>
    <div v-if="store.notice" class="toast notice-toast" role="status"><i class="mdi mdi-check-circle-outline" /> {{ store.notice }}</div>

    <main class="room-content">
      <div class="desktop-player"><NowPlaying /></div>
      <section class="workspace">
        <nav class="desktop-tabs" aria-label="房间功能">
          <button v-for="tab in tabs.filter((item) => item.value !== 'playing')" :key="tab.value" :class="{ active: store.activeTab === tab.value }" @click="store.activeTab = tab.value"><i :class="`mdi ${tab.icon}`" />{{ tab.label }}<i v-if="tab.planned" class="mdi mdi-progress-wrench mini-planned" /></button>
        </nav>
        <div class="mobile-player" v-if="store.activeTab === 'playing'"><NowPlaying /></div>
        <QueuePanel v-else-if="store.activeTab === 'queue'" />
        <HistoryPanel v-else-if="store.activeTab === 'history'" />
        <CatalogPanel v-else-if="store.activeTab === 'catalog'" />
        <FavoritesPanel v-else-if="store.activeTab === 'favorites'" />
        <ComingSoonPanel v-else feature="群聊" />
      </section>
    </main>

    <nav class="mobile-tabs" aria-label="房间功能">
      <button v-for="tab in tabs" :key="tab.value" :class="{ active: store.activeTab === tab.value }" @click="store.activeTab = tab.value"><span class="nav-icon"><i :class="`mdi ${tab.icon}`" /><i v-if="tab.planned" class="mdi mdi-progress-wrench mini-planned" /></span><span>{{ tab.label }}</span></button>
    </nav>
    <ShareDialog :open="shareOpen" @close="shareOpen = false" />
  </div>
</template>
