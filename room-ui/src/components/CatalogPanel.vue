<script setup lang="ts">
import { computed } from 'vue'
import { useRoomStore } from '../stores/room'
import { useI18n } from '../lib/i18n'
import SongList from './SongList.vue'
const store = useRoomStore()
const { t } = useI18n()
const songs = computed(() => store.selectedAlbum?.song || store.catalog.songs)
const albums = computed(() => store.catalogMode === 'artist' ? store.artistAlbums : store.catalog.albums)
const sections = ['songs', 'albums', 'artists'] as const
</script>
<template>
  <section class="work-panel catalog-panel" aria-labelledby="catalog-title">
    <header class="panel-heading"><h2 id="catalog-title">{{ t('catalog') }}</h2><span class="count-pill">{{ t('quota', { count: store.queueRemaining }) }}</span></header>
    <form class="catalog-tools" role="search" @submit.prevent="store.searchCatalog">
      <label class="search-field"><i aria-hidden="true" class="mdi mdi-magnify" /><input v-model="store.catalogQuery" :placeholder="t('searchPlaceholder')" :aria-label="t('searchLibrary')" /><button v-if="store.catalogQuery" type="button" class="icon-button" :aria-label="t('clear')" @click="store.catalogQuery = ''; store.browseCatalog()"><i aria-hidden="true" class="mdi mdi-close" /></button></label>
      <select v-if="store.folders.length > 1" v-model="store.folderID" :aria-label="t('catalog')" @change="store.browseCatalog"><option v-for="folder in store.folders" :key="folder.id" :value="folder.id">{{ folder.name }}</option></select><button class="primary-button" type="submit">{{ t('search') }}</button>
    </form>
    <div class="toolbar"><button class="secondary-button compact" :disabled="store.catalogLoading" @click="store.refreshRandom">{{ t('randomRefresh') }}</button><button class="secondary-button compact" :disabled="store.busy || !store.queueRemaining" @click="store.randomFillQueue(1)"><i aria-hidden="true" class="mdi mdi-dice-5-outline" />{{ t('lucky') }}</button></div>
    <nav class="catalog-kind-tabs" :aria-label="t('catalog')"><button v-for="section in sections" :key="section" :class="{ active: store.catalogSection === section }" :aria-pressed="store.catalogSection === section" :data-testid="`catalog-${section}`" @click="store.selectCatalogSection(section)">{{ t(section) }}<small>{{ store.catalog[section].length }}</small></button></nav>
    <div v-if="store.catalogError" class="media-error" role="alert">{{ store.catalogError }}</div>
    <div v-if="store.catalogLoading" class="panel-loading"><span class="spinner" />{{ t('loading') }}</div>
    <template v-else>
      <div class="subheading-row"><button v-if="['album', 'artist'].includes(store.catalogMode)" class="back-button" @click="store.returnToCatalog"><i aria-hidden="true" class="mdi mdi-arrow-left" />{{ t('back') }}</button><h3>{{ store.catalogTitle }}</h3><button v-if="store.selectedAlbum" class="secondary-button compact" :disabled="!store.queueRemaining || store.busy" data-testid="queue-whole-album" @click="store.addAlbum(store.selectedAlbum)">{{ t('queueAll') }}</button></div>
      <SongList v-if="store.catalogSection === 'songs' && songs.length" :songs="songs" play-next data-testid="catalog-song-list" />
      <div v-else-if="store.catalogSection === 'albums' && albums.length" class="cover-grid" data-testid="catalog-album-grid">
        <article v-for="album in albums" :key="album.id" class="cover-card"><button class="cover-card-open" :aria-label="`${t('openAlbum')} ${album.name}`" @click="store.openAlbum(album)"><img v-if="album.coverArt" :src="store.coverURL(album.coverArt, 320)" alt="" loading="lazy" /><span v-else class="cover-placeholder"><i aria-hidden="true" class="mdi mdi-album" /></span><strong>{{ album.name }}</strong></button><button class="text-link" @click="store.navigateArtist(album.artist || '', album.artistId)">{{ album.artist || t('unknownArtist') }}</button><button class="icon-button" :aria-label="t(store.isStarred(album.id, 'album') ? 'unfavorite' : 'favorite')" :aria-pressed="store.isStarred(album.id, 'album')" @click="store.toggleStar(album, 'album')"><i aria-hidden="true" :class="['mdi', store.isStarred(album.id, 'album') ? 'mdi-heart' : 'mdi-heart-outline']" /></button><button class="cover-queue" :disabled="!store.queueRemaining || store.busy" @click="store.addAlbum(album)"><i aria-hidden="true" class="mdi mdi-playlist-plus" />{{ t('queueAll') }}</button></article>
      </div>
      <div v-else-if="store.catalogSection === 'artists' && store.catalog.artists.length" class="artist-grid" data-testid="catalog-artist-grid"><article v-for="artist in store.catalog.artists" :key="artist.id" class="artist-card"><button class="artist-open" @click="store.openArtist(artist)"><span class="artist-avatar">{{ artist.name.slice(0, 1).toUpperCase() }}</span><span><strong>{{ artist.name }}</strong><small>{{ artist.albumCount || 0 }} · {{ t('albums') }}</small></span></button><button class="icon-button" :aria-label="t(store.isStarred(artist.id, 'artist') ? 'unfavorite' : 'favorite')" :aria-pressed="store.isStarred(artist.id, 'artist')" @click="store.toggleStar(artist, 'artist')"><i aria-hidden="true" :class="['mdi', store.isStarred(artist.id, 'artist') ? 'mdi-heart' : 'mdi-heart-outline']" /></button></article></div>
      <div v-else class="panel-empty"><i aria-hidden="true" class="mdi mdi-music-box-multiple-outline" /><h3>{{ t('empty') }}</h3></div>
    </template>
  </section>
</template>
