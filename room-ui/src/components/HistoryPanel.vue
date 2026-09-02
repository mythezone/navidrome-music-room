<script setup lang="ts">
import { useRoomStore } from '../stores/room'
import { formatDuration } from '../lib/sync'

const store = useRoomStore()
</script>

<template>
  <section class="work-panel" aria-labelledby="history-title">
    <header class="panel-heading"><div><span class="section-kicker">RECENTLY PLAYED</span><h2 id="history-title">播放历史</h2></div><span class="count-pill">{{ store.history.length }} 条</span></header>
    <div v-if="store.history.length" class="track-list">
      <article v-for="entry in store.history" :key="entry.historyID" class="track-row history-row">
        <img v-if="entry.track.coverArtID" :src="store.coverURL(entry.track.coverArtID, 112)" :alt="`${entry.track.album} 封面`" />
        <div v-else class="mini-art"><i class="mdi mdi-history" /></div>
        <div class="track-main"><strong>{{ entry.track.title }}</strong><span>{{ entry.track.artist }} · {{ entry.track.album }}</span><small>{{ new Date(entry.startedAt).toLocaleString() }} · {{ entry.contributorDisplayName || entry.contributorUsername }} 点播</small></div>
        <time>{{ formatDuration(entry.playedSeconds || entry.track.durationSeconds) }}</time>
      </article>
    </div>
    <div v-else class="panel-empty"><i class="mdi mdi-history" /><h3>还没有播放记录</h3><p>完整播完或切换歌曲后，记录会保存在插件独立数据库中。</p></div>
  </section>
</template>
