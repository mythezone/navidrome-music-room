<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRoomStore } from '../stores/room'
import { formatDuration } from '../lib/sync'
import { useI18n } from '../lib/i18n'
import { eqPresets } from '../lib/eq'
import TrackLinks from './TrackLinks.vue'
import TrackActions from './TrackActions.vue'
const store = useRoomStore()
const { t } = useI18n()
const eqOpen = ref(false)
const lyricsFocus = ref(false)
const lyrics = ref<HTMLElement>()
const progress = computed(() => Math.min(store.currentTime, store.duration || store.currentTime))
const actionLabel = computed(() => t(store.canControl ? store.playback?.status === 'playing' ? 'pauseRoom' : 'playRoom' : store.isListening ? 'pauseLocal' : 'listen'))
const paused = computed(() => store.canControl ? store.playback?.status !== 'playing' : !store.isListening)
watch(() => [store.activeLyricIndex, store.lyrics, lyricsFocus.value, store.immersive, store.activeTab], async () => {
  await nextTick()
  const active = lyrics.value?.querySelector<HTMLElement>(`[data-lyric-index="${store.activeLyricIndex}"]`)
  if (active && lyrics.value) lyrics.value.scrollTo({ top: active.getBoundingClientRect().top - lyrics.value.getBoundingClientRect().top + lyrics.value.scrollTop - lyrics.value.clientHeight / 2 + active.clientHeight / 2, behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'instant' : 'smooth' })
}, { flush: 'post', immediate: true })
</script>
<template>
  <section :class="['now-playing', { 'lyrics-focused': store.immersive && lyricsFocus }]" aria-labelledby="now-playing-title">
    <div class="section-kicker"><i aria-hidden="true" class="mdi mdi-access-point" />{{ t('connected') }}</div>
    <div class="artwork-wrap">
      <button class="artwork-button" :aria-label="t(store.immersive ? 'exitImmersive' : 'immersive')" @click="store.immersive = !store.immersive">
        <img v-if="store.currentTrack?.coverArtID" class="hero-artwork" :src="store.coverURL(store.currentTrack.coverArtID)" :alt="store.currentTrack.album" />
        <span v-else class="hero-artwork artwork-placeholder"><i aria-hidden="true" class="mdi mdi-music-note" /></span>
        <span v-if="store.audioLoading" class="artwork-loader"><span class="spinner" /></span>
      </button>
      <button class="online-pill" @click="store.infoOpen = true"><span class="connection-dot" />{{ t('onlineCount', { count: store.onlineCount }) }}</button>
    </div>
    <div class="track-copy" data-testid="current-track">
      <h2 id="now-playing-title">{{ store.currentTrack?.title || t('waiting') }}</h2>
      <p v-if="store.currentTrack"><TrackLinks :artist="store.currentTrack.artist" :artist-id="store.currentTrack.artistID" :album="store.currentTrack.album" :album-id="store.currentTrack.albumID" /></p><p v-else>{{ t('chooseSong') }}</p>
      <small v-if="store.playback?.contributorUsername">{{ t('requestedBy', { name: store.playback.contributorDisplayName || store.playback.contributorUsername }) }}</small>
    </div>
    <div class="toolbar player-tools">
      <TrackActions v-if="store.currentTrack" :song="store.toSong(store.currentTrack)" />
      <button class="icon-button" :title="t('sync')" :aria-label="t('sync')" @click="store.synchronizePlayback"><i aria-hidden="true" class="mdi mdi-sync" /></button>
      <button class="icon-button" :title="t('eq')" :aria-label="t('eq')" :aria-expanded="eqOpen" @click="eqOpen = !eqOpen"><i aria-hidden="true" class="mdi mdi-tune-vertical" /></button>
    </div>
    <div class="timeline">
      <input class="range" type="range" min="0" :max="Math.max(store.duration, 1)" step="0.1" :value="progress" :disabled="!store.canControl || !store.currentTrack" :aria-label="t('progress')" @change="store.seek(Number(($event.target as HTMLInputElement).value))" />
      <div><time data-testid="current-time">{{ formatDuration(progress) }}</time><time>{{ formatDuration(store.duration) }}</time></div>
    </div>
    <div class="transport">
      <button v-if="store.canControl" class="transport-side" :disabled="store.busy" :title="t('previous')" :aria-label="t('previous')" @click="store.previous"><i aria-hidden="true" class="mdi mdi-skip-previous" /></button>
      <button class="transport-main" :disabled="(!store.currentTrack && !store.queue.length) || store.busy" :aria-label="actionLabel" data-testid="listen-button" @click="store.togglePlayback"><span v-if="store.busy" class="spinner" /><i aria-hidden="true" v-else :class="['mdi', paused ? 'mdi-play' : 'mdi-pause']" /></button>
      <button v-if="store.canControl || store.canSkip" class="transport-side" :disabled="!store.canSkip || !store.currentTrack || store.busy" :title="t('next')" :aria-label="t('next')" @click="store.next"><i aria-hidden="true" class="mdi mdi-skip-next" /></button>
    </div>
    <p class="transport-caption">{{ actionLabel }}</p>
    <p v-if="!store.canControl" class="transport-caption">{{ t('localHint') }}<br v-if="store.canSkip" /><span v-if="store.canSkip">{{ t(store.room?.allowMemberSkip ? 'skipRoom' : 'skipOwn') }}</span></p>
    <button v-if="store.canControl && store.playback?.status === 'playing'" class="text-link local-listen" @click="store.isListening ? store.stopListening() : store.startListening()">{{ t(store.isListening ? 'pauseLocal' : 'listen') }}</button>
    <div v-if="store.autoplayBlocked" class="media-warning">{{ t('autoplay') }}</div>
    <div v-if="store.audioError" class="media-error" role="alert">{{ store.audioError }}</div>
    <div class="volume-row"><i aria-hidden="true" class="mdi mdi-volume-medium" /><input class="range" type="range" min="0" max="1" step="0.05" :value="store.volume" :aria-label="t('volume')" @input="store.setVolume(Number(($event.target as HTMLInputElement).value))" /></div>
    <section v-if="eqOpen" class="eq-panel" :aria-label="t('eq')">
      <label class="toggle-label"><input type="checkbox" :checked="store.eqEnabled" :disabled="!store.eqAvailable" @change="store.setEqEnabled(($event.target as HTMLInputElement).checked)" />{{ t('eqEnable') }}</label>
      <p v-if="store.nativeBackgroundPlayback">{{ t('eqMobile') }}</p>
      <p v-else-if="!store.eqAvailable">{{ t('error.eq_unavailable') }}</p>
      <div class="segmented wrap"><button v-for="(_, preset) in eqPresets" :key="preset" :disabled="!store.eqAvailable" :class="{ active: store.eqPreset === preset }" @click="store.applyEqPreset(preset)">{{ t(preset) }}</button><button :disabled="!store.eqAvailable" :class="{ active: store.eqPreset === 'custom' }" @click="store.applyEqPreset('custom')">{{ t('custom') }}</button></div>
      <div class="eq-bands"><label v-for="(band, index) in store.eqBands" :key="band.frequency"><span>{{ band.frequency < 1000 ? `${band.frequency} Hz` : `${band.frequency / 1000} kHz` }}</span><input type="range" min="-12" max="12" step="1" :disabled="!store.eqAvailable" :value="band.gain" :aria-label="`${t('eq')} ${band.frequency} Hz`" @input="store.setEqBandGain(index, Number(($event.target as HTMLInputElement).value))" /><output>{{ band.gain }} dB</output></label></div>
    </section>
    <button v-if="store.immersive && store.lyrics.length" class="text-link" :aria-expanded="lyricsFocus" @click="lyricsFocus = !lyricsFocus">{{ t('lyricsFocus') }}</button>
    <div v-if="store.lyrics.length" ref="lyrics" class="lyrics" :aria-label="t('lyrics')" :role="store.immersive ? 'button' : undefined" :tabindex="store.immersive ? 0 : undefined" @click="store.immersive && (lyricsFocus = !lyricsFocus)" @keydown.enter.prevent="store.immersive && (lyricsFocus = !lyricsFocus)" @keydown.space.prevent="store.immersive && (lyricsFocus = !lyricsFocus)">
      <p v-for="(line, index) in store.lyrics" :key="`${line.time}-${index}`" :data-lyric-index="index" :class="{ active: index === store.activeLyricIndex }">{{ line.text }}</p>
    </div>
  </section>
</template>
