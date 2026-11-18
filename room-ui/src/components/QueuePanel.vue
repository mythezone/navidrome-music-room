<script setup lang="ts">
import { ref } from 'vue'
import { useRoomStore } from '../stores/room'
import { useI18n } from '../lib/i18n'
import { formatDuration } from '../lib/sync'
import TrackLinks from './TrackLinks.vue'
import TrackActions from './TrackActions.vue'
const store = useRoomStore()
const { t } = useI18n()
const list = ref<HTMLElement>()
const draggingID = ref('')
const targetID = ref('')
function start(event: PointerEvent, id: string): void {
  if (!store.canReorderQueue || event.button !== 0) return
  event.preventDefault()
  draggingID.value = targetID.value = id
  list.value?.setPointerCapture(event.pointerId)
}
function move(event: PointerEvent): void {
  if (!draggingID.value || !list.value) return
  const row = document.elementFromPoint(event.clientX, event.clientY)?.closest<HTMLElement>('[data-queue-id]')
  if (row && list.value.contains(row)) targetID.value = row.dataset.queueId || draggingID.value
  const bounds = list.value.getBoundingClientRect()
  if (event.clientY < bounds.top + 40) list.value.scrollTop -= 20
  if (event.clientY > bounds.bottom - 40) list.value.scrollTop += 20
}
function cancel(): void { draggingID.value = targetID.value = '' }
function end(): void {
  const from = store.queue.findIndex(item => item.queueID === draggingID.value)
  const to = store.queue.findIndex(item => item.queueID === targetID.value)
  cancel()
  void store.reorderQueue(from, to)
}
</script>
<template>
  <section class="work-panel" aria-labelledby="queue-title">
    <header class="panel-heading"><h2 id="queue-title">{{ t('queue') }}</h2><span class="count-pill">{{ t('quota', { count: store.queueRemaining }) }}</span></header>
    <div class="toolbar"><button v-if="store.canManage" class="secondary-button compact" :disabled="store.queue.length < 3 || store.busy" @click="store.shuffleQueue"><i aria-hidden="true" class="mdi mdi-shuffle" />{{ t('shuffle') }}</button><button class="secondary-button compact" :disabled="!store.queueRemaining || store.busy" @click="store.randomFillQueue()"><i aria-hidden="true" class="mdi mdi-dice-multiple-outline" />{{ t('randomFill') }}</button></div>
    <div v-if="store.queue.length" ref="list" class="track-list room-track-list" data-testid="queue-list" @pointermove="move" @pointerup="end" @pointercancel="cancel" @lostpointercapture="cancel">
      <article v-for="(entry, index) in store.queue" :key="entry.queueID" :data-queue-id="entry.queueID" :class="['track-row', 'queue-row', { 'drag-target': draggingID && targetID === entry.queueID }]">
        <button v-if="store.canReorderQueue" class="icon-button drag-handle" :aria-label="t('reorder')" @pointerdown="start($event, entry.queueID)" @keydown.up.prevent="store.moveQueue(index, -1)" @keydown.down.prevent="store.moveQueue(index, 1)"><i aria-hidden="true" class="mdi mdi-drag-vertical" /></button><span v-else class="track-index">{{ index + 1 }}</span>
        <img v-if="entry.track.coverArtID" :src="store.coverURL(entry.track.coverArtID, 112)" alt="" /><div v-else class="mini-art"><i aria-hidden="true" class="mdi mdi-music-note" /></div>
        <div class="track-main"><strong>{{ entry.track.title }}</strong><TrackLinks :artist="entry.track.artist" :artist-id="entry.track.artistID" :album="entry.track.album" :album-id="entry.track.albumID" /><small>{{ t('requestedBy', { name: entry.contributorDisplayName || entry.contributorUsername }) }}</small></div>
        <time>{{ formatDuration(entry.track.durationSeconds) }}</time>
        <div class="row-actions"><TrackActions :song="store.toSong(entry.track)" />
          <template v-if="store.canReorderQueue"><button class="icon-button" :disabled="index === 0" :aria-label="t('top')" :title="t('top')" @click="store.topQueue(entry)"><i aria-hidden="true" class="mdi mdi-format-vertical-align-top" /></button><button class="icon-button" :disabled="index === 0" :aria-label="t('moveUp')" @click="store.moveQueue(index, -1)"><i aria-hidden="true" class="mdi mdi-chevron-up" /></button><button class="icon-button" :disabled="index === store.queue.length - 1" :aria-label="t('moveDown')" @click="store.moveQueue(index, 1)"><i aria-hidden="true" class="mdi mdi-chevron-down" /></button></template>
          <button v-if="store.canManage || store.sameUser(entry.contributorUsername)" class="icon-button danger" :aria-label="t('remove')" @click="store.removeQueue(entry)"><i aria-hidden="true" class="mdi mdi-close" /></button>
        </div>
      </article>
    </div>
    <div v-else class="panel-empty"><i aria-hidden="true" class="mdi mdi-playlist-music-outline" /><h3>{{ t('empty') }}</h3><p>{{ t('chooseSong') }}</p><button class="secondary-button" @click="store.activeTab = 'catalog'">{{ t('catalog') }}</button></div>
  </section>
</template>
