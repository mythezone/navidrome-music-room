<script setup lang="ts">
import { useRoomStore } from '../stores/room'
import { useI18n } from '../lib/i18n'
import { formatDuration } from '../lib/sync'
import TrackLinks from './TrackLinks.vue'
import TrackActions from './TrackActions.vue'
const store = useRoomStore()
const { t, locale } = useI18n()
</script>
<template>
  <section class="work-panel" aria-labelledby="history-title">
    <header class="panel-heading"><h2 id="history-title">{{ t('history') }}</h2><span class="count-pill">{{ store.history.length }}</span></header>
    <div v-if="store.history.length" class="track-list">
      <article v-for="entry in store.history" :key="entry.historyID" class="track-row history-row">
        <img v-if="entry.track.coverArtID" :src="store.coverURL(entry.track.coverArtID, 112)" alt="" loading="lazy" /><div v-else class="mini-art"><i aria-hidden="true" class="mdi mdi-history" /></div>
        <div class="track-main"><strong>{{ entry.track.title }}</strong><TrackLinks :artist="entry.track.artist" :artist-id="entry.track.artistID" :album="entry.track.album" :album-id="entry.track.albumID" /><small>{{ new Date(entry.startedAt).toLocaleString(locale) }} · {{ t('requestedBy', { name: entry.contributorDisplayName || entry.contributorUsername }) }}</small></div>
        <time>{{ formatDuration(entry.playedSeconds || entry.track.durationSeconds) }}</time>
        <div class="row-actions"><TrackActions :song="store.toSong(entry.track)" /><button class="icon-button" :disabled="!store.queueRemaining || store.busy" :title="t('requeue')" :aria-label="t('requeue')" @click="store.requeueHistory(entry)"><i aria-hidden="true" class="mdi mdi-playlist-plus" /></button></div>
      </article>
    </div>
    <div v-else class="panel-empty"><i aria-hidden="true" class="mdi mdi-history" /><h3>{{ t('empty') }}</h3></div>
    <button v-if="store.hasMoreHistory" class="secondary-button load-more" :disabled="store.historyLoading" @click="store.loadOlderHistory">{{ t(store.historyLoading ? 'loading' : 'olderHistory') }}</button>
  </section>
</template>
