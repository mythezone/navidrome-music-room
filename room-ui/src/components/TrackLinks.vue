<script setup lang="ts">
import { useRoomStore } from '../stores/room'
import { useI18n } from '../lib/i18n'
defineProps<{ artist?: string; artistId?: string; album?: string; albumId?: string }>()
const store = useRoomStore()
const { t } = useI18n()
</script>
<template>
  <span class="track-links">
    <button class="text-link" :title="t('openArtist')" @click="store.navigateArtist(artist || '', artistId)">{{ artist || t('unknownArtist') }}</button>
    <span aria-hidden="true"> · </span>
    <button class="text-link" :title="t('openAlbum')" @click="store.navigateAlbum(album || '', albumId)">{{ album || t('unknownAlbum') }}</button>
  </span>
</template>
