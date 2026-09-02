<script setup lang="ts">
import { computed } from 'vue'
import { useRoomStore } from '../stores/room'
import { formatDuration } from '../lib/sync'

const store = useRoomStore()
const progress = computed(() => Math.min(store.currentTime, store.duration || store.currentTime))
const actionLabel = computed(() => {
  if (!store.isListening) return '开始收听'
  if (store.canManage) return store.playback?.status === 'playing' ? '暂停房间' : '播放房间'
  return '暂停收听'
})

function seek(event: Event): void {
  void store.seek(Number((event.target as HTMLInputElement).value))
}
</script>

<template>
  <section class="now-playing" aria-labelledby="now-playing-title">
    <div class="section-kicker"><i class="mdi mdi-access-point" /> 全房间同步</div>
    <div class="artwork-wrap">
      <img
        v-if="store.currentTrack?.coverArtID"
        class="hero-artwork"
        :src="store.coverURL(store.currentTrack.coverArtID)"
        :alt="`${store.currentTrack.album} 封面`"
      />
      <div v-else class="hero-artwork artwork-placeholder"><i class="mdi mdi-music-note" /></div>
      <span v-if="store.audioLoading" class="artwork-loader"><span class="spinner" /></span>
    </div>

    <div class="track-copy" data-testid="current-track">
      <h2 id="now-playing-title">{{ store.currentTrack?.title || '等待点歌' }}</h2>
      <p>{{ store.currentTrack ? `${store.currentTrack.artist} · ${store.currentTrack.album}` : '从点歌台选择一首歌曲开始' }}</p>
      <small v-if="store.playback?.contributorDisplayName">由 {{ store.playback.contributorDisplayName }} 点播</small>
    </div>

    <div class="timeline">
      <input
        class="range"
        type="range"
        min="0"
        :max="Math.max(store.duration, 1)"
        step="0.1"
        :value="progress"
        :disabled="!store.canManage || !store.currentTrack"
        aria-label="播放进度"
        @change="seek"
      />
      <div><time data-testid="current-time">{{ formatDuration(progress) }}</time><time>{{ formatDuration(store.duration) }}</time></div>
    </div>

    <div class="transport">
      <button class="transport-side" type="button" :disabled="!store.canManage || !store.currentTrack" title="回到开头" @click="store.seek(0)"><i class="mdi mdi-skip-previous" /></button>
      <button class="transport-main" type="button" :disabled="!store.currentTrack || store.busy" :aria-label="actionLabel" data-testid="listen-button" @click="store.togglePlayback">
        <span v-if="store.busy" class="spinner" />
        <i v-else :class="store.isListening && store.playback?.status === 'playing' ? 'mdi mdi-pause' : 'mdi mdi-play'" />
      </button>
      <button class="transport-side" type="button" :disabled="!store.canManage || !store.currentTrack" title="下一首" @click="store.next"><i class="mdi mdi-skip-next" /></button>
    </div>
    <p class="transport-caption">{{ actionLabel }}<span v-if="!store.canManage"> · 不影响其他成员</span></p>

    <div v-if="store.autoplayBlocked" class="media-warning"><i class="mdi mdi-gesture-tap" /> 浏览器已阻止自动播放，请点击“开始收听”。</div>
    <div v-if="store.audioError" class="media-error" role="alert"><i class="mdi mdi-alert-circle-outline" /> {{ store.audioError }}</div>

    <div class="volume-row">
      <i :class="store.volume === 0 ? 'mdi mdi-volume-off' : 'mdi mdi-volume-medium'" />
      <input class="range" type="range" min="0" max="1" step="0.05" :value="store.volume" aria-label="音量" @input="store.setVolume(Number(($event.target as HTMLInputElement).value))" />
    </div>

    <div v-if="store.lyrics.length" class="lyrics" aria-label="歌词">
      <p
        v-for="(line, index) in store.lyrics"
        :key="`${line.time}-${index}`"
        :class="{ active: index === store.activeLyricIndex }"
      >{{ line.text }}</p>
    </div>
  </section>
</template>
