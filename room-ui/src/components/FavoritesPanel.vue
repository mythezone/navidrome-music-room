<script setup lang="ts">
import { ref } from 'vue'
import { useRoomStore } from '../stores/room'
import { useI18n } from '../lib/i18n'
import SongList from './SongList.vue'
const store = useRoomStore()
const { t } = useI18n()
const section = ref('songs')
const sections = ['songs', 'albums', 'artists', 'playlists'] as const
</script>
<template>
  <section class="work-panel" aria-labelledby="favorites-title">
    <header class="panel-heading"><h2 id="favorites-title">{{ t('favorites') }}</h2><button class="icon-button" :aria-label="t('refresh')" @click="store.loadFavorites"><i aria-hidden="true" class="mdi mdi-refresh" /></button></header>
    <nav class="segmented" :aria-label="t('favorites')"><button v-for="item in sections" :key="item" :class="{ active: section === item }" :aria-pressed="section === item" @click="section = item">{{ t(item) }}</button></nav>
    <div v-if="store.favoritesLoading" class="panel-loading"><span class="spinner" />{{ t('loading') }}</div>
    <template v-else>
      <SongList v-if="section === 'songs' && store.favorites.songs.length" :songs="store.favorites.songs" />
      <div v-else-if="section === 'albums' && store.favorites.albums.length" class="cover-grid">
        <article v-for="album in store.favorites.albums" :key="album.id" class="cover-card"><button class="cover-card-open" @click="store.openAlbum(album)"><img v-if="album.coverArt" :src="store.coverURL(album.coverArt, 320)" alt="" /><span v-else class="cover-placeholder"><i aria-hidden="true" class="mdi mdi-album" /></span><strong>{{ album.name }}</strong></button><div class="toolbar"><button class="icon-button" :aria-label="t('unfavorite')" @click="store.toggleStar(album, 'album')"><i aria-hidden="true" class="mdi mdi-heart" /></button><button class="cover-queue" :disabled="!store.queueRemaining || store.busy" @click="store.addAlbum(album)">{{ t('queueAll') }}</button></div></article>
      </div>
      <div v-else-if="section === 'artists' && store.favorites.artists.length" class="artist-grid"><article v-for="artist in store.favorites.artists" :key="artist.id" class="artist-card"><button class="text-link" @click="store.openArtist(artist)">{{ artist.name }}</button><button class="icon-button" :aria-label="t('unfavorite')" @click="store.toggleStar(artist, 'artist')"><i aria-hidden="true" class="mdi mdi-heart" /></button></article></div>
      <template v-else-if="section === 'playlists' && store.playlists.length">
        <div class="playlist-grid"><button v-for="playlist in store.playlists" :key="playlist.id" class="playlist-card" @click="store.openPlaylist(playlist)"><span class="playlist-icon"><i aria-hidden="true" class="mdi mdi-playlist-music" /></span><span><strong>{{ playlist.name }}</strong><small>{{ playlist.songCount || 0 }} · {{ t('songs') }}</small></span></button></div>
        <template v-if="store.selectedPlaylist"><div class="subheading-row"><h3>{{ store.selectedPlaylist.name }}</h3><button class="secondary-button compact" :disabled="!store.queueRemaining || store.busy" @click="store.addPlaylist(store.selectedPlaylist)">{{ t('queueAll') }}</button></div><SongList :songs="store.selectedPlaylist.entry || []" /></template>
      </template>
      <div v-else class="panel-empty"><i aria-hidden="true" class="mdi mdi-heart-outline" /><h3>{{ t('empty') }}</h3></div>
    </template>
  </section>
</template>
