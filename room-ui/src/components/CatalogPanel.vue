<script setup lang="ts">
import { computed } from 'vue'
import { useRoomStore } from '../stores/room'
import { formatDuration } from '../lib/sync'
import type { SubsonicAlbum } from '../types'

const store = useRoomStore()
const songs = computed(() => store.selectedAlbum?.song || store.catalog.songs)
const albums = computed(() => store.catalogMode === 'artist' ? store.artistAlbums : store.catalog.albums)
const isDetail = computed(() => store.catalogMode === 'album' || store.catalogMode === 'artist')
const sectionTitle = computed(() => {
  if (isDetail.value) return store.catalogTitle
  if (store.catalogMode === 'search') return store.catalogTitle
  return store.catalogSection === 'songs' ? '随机歌曲' : store.catalogSection === 'albums' ? '最新专辑' : '全部歌手'
})
const activeCount = computed(() => {
  if (store.catalogSection === 'songs') return songs.value.length
  if (store.catalogSection === 'albums') return albums.value.length
  return store.catalog.artists.length
})

const sections = computed(() => [
  { value: 'songs' as const, label: '歌曲', icon: 'mdi-music-note', count: store.catalog.songs.length },
  { value: 'albums' as const, label: '专辑', icon: 'mdi-album', count: store.catalog.albums.length },
  { value: 'artists' as const, label: '歌手', icon: 'mdi-account-music-outline', count: store.catalog.artists.length },
])

function submit(): void {
  void store.searchCatalog()
}

function folderChanged(): void {
  void store.browseCatalog()
}

function queueAlbum(event: Event, album: SubsonicAlbum): void {
  event.stopPropagation()
  void store.addAlbum(album)
}
</script>

<template>
  <section class="work-panel catalog-panel" aria-labelledby="catalog-title">
    <header class="panel-heading catalog-heading">
      <div><span class="section-kicker">NAVIDROME LIBRARY</span><h2 id="catalog-title">点歌台</h2></div>
      <span class="count-pill">个人额度 {{ store.queueRemaining }}</span>
    </header>

    <form class="catalog-tools" role="search" @submit.prevent="submit">
      <label class="search-field"><i class="mdi mdi-magnify" /><input v-model="store.catalogQuery" placeholder="搜索歌曲、专辑或歌手" aria-label="搜索曲库" /><button v-if="store.catalogQuery" type="button" class="icon-button" aria-label="清空" @click="store.catalogQuery = ''; store.browseCatalog()"><i class="mdi mdi-close" /></button></label>
      <select v-if="store.folders.length > 1" v-model="store.folderID" aria-label="音乐库" @change="folderChanged"><option v-for="folder in store.folders" :key="folder.id" :value="folder.id">{{ folder.name }}</option></select>
      <button class="primary-button" type="submit">搜索</button>
    </form>

    <nav class="catalog-kind-tabs" aria-label="选歌方式">
      <button
        v-for="section in sections"
        :key="section.value"
        type="button"
        :class="{ active: store.catalogSection === section.value }"
        :data-testid="`catalog-${section.value}`"
        @click="store.selectCatalogSection(section.value)"
      >
        <i :class="`mdi ${section.icon}`" />
        <span>{{ section.label }}</span>
        <small>{{ section.count }}</small>
      </button>
    </nav>

    <div v-if="store.catalogError" class="media-error" role="alert">{{ store.catalogError }}</div>
    <div v-if="store.catalogLoading" class="panel-loading"><span class="spinner" /> 正在读取 Navidrome 曲库</div>
    <template v-else>
      <div class="subheading-row">
        <button v-if="isDetail" type="button" class="back-button" @click="store.returnToCatalog"><i class="mdi mdi-arrow-left" /> 返回结果</button>
        <h3>{{ sectionTitle }}</h3>
        <span v-if="!isDetail" class="result-count">{{ activeCount }} 项</span>
        <button v-if="store.selectedAlbum" type="button" class="secondary-button compact" :disabled="!store.selectedAlbum.song?.length || !store.queueRemaining || store.busy" data-testid="queue-whole-album" @click="store.addAlbum(store.selectedAlbum)"><i class="mdi mdi-playlist-plus" /> 整张点播</button>
      </div>

      <div v-if="store.catalogSection === 'songs' && songs.length" class="track-list compact-list" data-testid="catalog-song-list">
        <article v-for="song in songs" :key="song.id" class="track-row">
          <img v-if="song.coverArt" :src="store.coverURL(song.coverArt, 112)" :alt="`${song.album || ''} 封面`" />
          <div v-else class="mini-art"><i class="mdi mdi-music-note" /></div>
          <div class="track-main"><strong>{{ song.title }}</strong><span>{{ song.artist || '未知歌手' }} · {{ song.album || '未知专辑' }}</span></div>
          <time>{{ formatDuration(song.duration || 0) }}</time>
          <button class="icon-button" :aria-label="song.starred ? '取消收藏' : '收藏'" @click="store.toggleStar(song)"><i :class="song.starred ? 'mdi mdi-heart' : 'mdi mdi-heart-outline'" /></button>
          <button class="queue-button" :disabled="!store.queueRemaining || store.busy" :aria-label="`点播 ${song.title}`" @click="store.addSong(song)"><i class="mdi mdi-plus" /><span>点播</span></button>
        </article>
      </div>

      <div v-else-if="store.catalogSection === 'albums' && albums.length" class="cover-grid" data-testid="catalog-album-grid">
        <article v-for="album in albums" :key="album.id" class="cover-card">
          <button type="button" class="cover-card-open" :aria-label="`查看专辑 ${album.name}`" @click="store.openAlbum(album)">
            <img v-if="album.coverArt" :src="store.coverURL(album.coverArt, 320)" :alt="`${album.name} 封面`" /><span v-else class="cover-placeholder"><i class="mdi mdi-album" /></span>
            <strong>{{ album.name }}</strong><span>{{ album.artist || '未知歌手' }}</span>
          </button>
          <button type="button" class="cover-queue" :disabled="!store.queueRemaining || store.busy" :aria-label="`整张点播 ${album.name}`" @click="queueAlbum($event, album)"><i class="mdi mdi-playlist-plus" /> 整张点播</button>
        </article>
      </div>

      <div v-else-if="store.catalogSection === 'artists' && store.catalog.artists.length" class="artist-grid" data-testid="catalog-artist-grid">
        <button v-for="artist in store.catalog.artists" :key="artist.id" type="button" class="artist-card" @click="store.openArtist(artist)">
          <span class="artist-avatar">{{ artist.name.slice(0, 1).toUpperCase() }}</span><span><strong>{{ artist.name }}</strong><small>{{ artist.albumCount || 0 }} 张专辑</small></span><i class="mdi mdi-chevron-right" />
        </button>
      </div>

      <div v-else class="panel-empty"><i class="mdi mdi-music-box-multiple-outline" /><h3>这里没有匹配的{{ store.catalogSection === 'songs' ? '歌曲' : store.catalogSection === 'albums' ? '专辑' : '歌手' }}</h3><p>请切换上方选歌方式、换一个搜索词，或确认你有房间音乐库的访问权限。</p></div>
    </template>
  </section>
</template>
