<script setup lang="ts">
import { useRoomStore } from '../stores/room'
import { formatDuration } from '../lib/sync'

const store = useRoomStore()
</script>

<template>
  <section class="work-panel" aria-labelledby="favorites-title">
    <header class="panel-heading"><div><span class="section-kicker">YOUR NAVIDROME</span><h2 id="favorites-title">收藏与歌单</h2></div><button class="icon-button" aria-label="刷新收藏" @click="store.loadFavorites"><i class="mdi mdi-refresh" /></button></header>
    <div v-if="store.favoritesLoading" class="panel-loading"><span class="spinner" /> 正在读取收藏</div>
    <template v-else>
      <div class="subheading-row"><h3>我的歌单</h3></div>
      <div v-if="store.playlists.length" class="playlist-grid">
        <button v-for="playlist in store.playlists" :key="playlist.id" class="playlist-card" @click="store.openPlaylist(playlist)"><span class="playlist-icon"><i class="mdi mdi-playlist-music" /></span><span><strong>{{ playlist.name }}</strong><small>{{ playlist.songCount || 0 }} 首 · {{ formatDuration(playlist.duration || 0) }}</small></span></button>
      </div>
      <p v-else class="muted-copy">你的 Navidrome 账号还没有歌单。</p>

      <template v-if="store.selectedPlaylist">
        <div class="subheading-row"><h3>{{ store.selectedPlaylist.name }}</h3><button class="secondary-button compact" :disabled="!store.queueRemaining" @click="store.addPlaylist(store.selectedPlaylist)"><i class="mdi mdi-playlist-plus" /> 加入待播</button></div>
        <div class="track-list compact-list">
          <article v-for="song in store.selectedPlaylist.entry || []" :key="song.id" class="track-row">
            <div class="mini-art"><i class="mdi mdi-music-note" /></div><div class="track-main"><strong>{{ song.title }}</strong><span>{{ song.artist }} · {{ song.album }}</span></div><time>{{ formatDuration(song.duration || 0) }}</time><button class="queue-button" :disabled="!store.queueRemaining" @click="store.addSong(song)"><i class="mdi mdi-plus" /><span>点播</span></button>
          </article>
        </div>
      </template>

      <div class="subheading-row"><h3>喜欢的歌曲</h3><span class="count-pill">{{ store.favorites.songs.length }} 首</span></div>
      <div v-if="store.favorites.songs.length" class="track-list compact-list">
        <article v-for="song in store.favorites.songs" :key="song.id" class="track-row">
          <img v-if="song.coverArt" :src="store.coverURL(song.coverArt, 112)" :alt="`${song.album || ''} 封面`" /><div v-else class="mini-art"><i class="mdi mdi-heart" /></div><div class="track-main"><strong>{{ song.title }}</strong><span>{{ song.artist }} · {{ song.album }}</span></div><time>{{ formatDuration(song.duration || 0) }}</time><button class="icon-button" aria-label="取消收藏" @click="store.toggleStar(song)"><i class="mdi mdi-heart" /></button><button class="queue-button" :disabled="!store.queueRemaining" @click="store.addSong(song)"><i class="mdi mdi-plus" /><span>点播</span></button>
        </article>
      </div>
      <div v-else class="panel-empty small"><i class="mdi mdi-heart-outline" /><h3>还没有收藏歌曲</h3><p>点歌台里的收藏会直接同步到 Navidrome。</p></div>
    </template>
  </section>
</template>
