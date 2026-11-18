<script setup lang="ts">
import { useRoomStore } from '../stores/room'
import { formatDuration } from '../lib/sync'
import type { SubsonicSong } from '../types'
import TrackActions from './TrackActions.vue'
import TrackLinks from './TrackLinks.vue'
defineProps<{ songs: SubsonicSong[]; playNext?: boolean }>()
const store = useRoomStore()
</script>
<template>
  <div class="track-list compact-list">
    <article v-for="song in songs" :key="song.id" class="track-row song-row">
      <img v-if="song.coverArt" :src="store.coverURL(song.coverArt, 112)" alt="" loading="lazy" />
      <div v-else class="mini-art"><i aria-hidden="true" class="mdi mdi-music-note" /></div>
      <div class="track-main"><strong>{{ song.title }}</strong><TrackLinks :artist="song.artist" :artist-id="song.artistId" :album="song.album" :album-id="song.albumId" /></div>
      <time>{{ formatDuration(song.duration || 0) }}</time>
      <TrackActions :song="song" queue :play-next="playNext" />
    </article>
  </div>
</template>
