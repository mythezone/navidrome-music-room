<script setup lang="ts">
import { useI18n } from '../lib/i18n'
import { useRoomStore } from '../stores/room'
import type { SubsonicSong } from '../types'
defineProps<{ song: SubsonicSong; queue?: boolean; playNext?: boolean }>()
const { t } = useI18n()
const store = useRoomStore()
</script>
<template>
  <div class="track-actions">
    <button class="icon-button" :aria-label="t(store.isStarred(song.id) ? 'unfavorite' : 'favorite')" :aria-pressed="store.isStarred(song.id)" @click="store.toggleStar(song)"><i aria-hidden="true" :class="['mdi', store.isStarred(song.id) ? 'mdi-heart' : 'mdi-heart-outline']" /></button>
    <button class="icon-button" :aria-label="t('shareTrack')" :title="t('shareTrack')" @click="store.suggestTrackToChat(song)"><i aria-hidden="true" class="mdi mdi-message-plus-outline" /></button>
    <button v-if="playNext && store.canControl" class="icon-button" :disabled="!store.queueRemaining || store.busy" :aria-label="t('playNext')" :title="t('playNext')" @click="store.playNext(song)"><i aria-hidden="true" class="mdi mdi-playlist-play" /></button>
    <button v-if="queue" class="queue-button" :disabled="!store.queueRemaining || store.busy" :aria-label="t('queueSong') + ' ' + song.title" @click="store.addSong(song)"><i aria-hidden="true" class="mdi mdi-plus" /><span>{{ t('queueSong') }}</span></button>
  </div>
</template>
