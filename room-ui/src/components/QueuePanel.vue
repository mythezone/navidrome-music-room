<script setup lang="ts">
import { useRoomStore } from '../stores/room'
import { formatDuration } from '../lib/sync'

const store = useRoomStore()

function removable(username: string): boolean {
  return Boolean(store.canManage || store.session?.user.username.toLowerCase() === username.toLowerCase())
}
</script>

<template>
  <section class="work-panel" aria-labelledby="queue-title">
    <header class="panel-heading">
      <div><span class="section-kicker">UP NEXT</span><h2 id="queue-title">待播放</h2></div>
      <span class="count-pill">{{ store.queue.length }} 首 · 还可点 {{ store.queueRemaining }} 首</span>
    </header>
    <div v-if="store.queue.length" class="track-list" data-testid="queue-list">
      <article v-for="(entry, index) in store.queue" :key="entry.queueID" class="track-row">
        <span class="track-index">{{ index + 1 }}</span>
        <img v-if="entry.track.coverArtID" :src="store.coverURL(entry.track.coverArtID, 112)" :alt="`${entry.track.album} 封面`" />
        <div v-else class="mini-art"><i class="mdi mdi-music-note" /></div>
        <div class="track-main"><strong>{{ entry.track.title }}</strong><span>{{ entry.track.artist }} · {{ entry.track.album }}</span><small>由 {{ entry.contributorDisplayName || entry.contributorUsername }} 点播</small></div>
        <time>{{ formatDuration(entry.track.durationSeconds) }}</time>
        <div class="row-actions">
          <template v-if="store.canManage">
            <button class="icon-button" :disabled="index === 0" aria-label="上移" @click="store.moveQueue(index, -1)"><i class="mdi mdi-chevron-up" /></button>
            <button class="icon-button" :disabled="index === store.queue.length - 1" aria-label="下移" @click="store.moveQueue(index, 1)"><i class="mdi mdi-chevron-down" /></button>
          </template>
          <button v-if="removable(entry.contributorUsername)" class="icon-button danger" aria-label="移除" @click="store.removeQueue(entry)"><i class="mdi mdi-close" /></button>
        </div>
      </article>
    </div>
    <div v-else class="panel-empty"><i class="mdi mdi-playlist-music-outline" /><h3>待播列表还是空的</h3><p>去点歌台添加一首歌，所有在线成员都会看到更新。</p><button class="secondary-button" @click="store.activeTab = 'catalog'">去点歌</button></div>
  </section>
</template>
