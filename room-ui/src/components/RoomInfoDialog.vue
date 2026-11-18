<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import RoomDialog from './RoomDialog.vue'
import { useRoomStore } from '../stores/room'
import { useI18n } from '../lib/i18n'
import type { Member, Period, RoomRankings, RoomStatistics, RoomWish, WishPayload, ChatReference, MediaStat } from '../types'
const store = useRoomStore()
const { t, errorText } = useI18n()
const tab = ref('members')
const period = ref<Period>('week')
const periods: Period[] = ['day', 'week', 'month', 'year', 'all']
const loading = ref(false)
const error = ref('')
const stats = ref<RoomStatistics>()
const rankings = ref<RoomRankings>()
const memberRanks = [{ key: 'requested', label: 'topDJs' }, { key: 'listened', label: 'topListeners' }, { key: 'chat', label: 'topChatters' }] as const
const resourceRanks = ['topArtists', 'topAlbums', 'topTracks'] as const
const hours = computed(() => Array.from({ length: 24 }, (_, hour) => ({ hour, seconds: stats.value?.hourlyDistribution?.find(item => item.hour === hour)?.seconds || 0 })))
const maxActivity = computed(() => Math.max(1, ...hours.value.map(item => item.seconds)))
const status = ref<'pending' | 'fulfilled'>('pending')
const visibleWishes = computed(() => store.wishes.filter(item => item.status === status.value))
const wishForm = ref<WishPayload>({ kind: 'track', title: '', artist: '', reason: '' })
const submitting = ref(false)
const fulfilling = ref<RoomWish>()
const resourceQuery = ref('')
const resourceResults = ref<ChatReference[]>([])
const chosen = ref<ChatReference[]>([])
const response = ref('')
const searching = ref(false)
let request = 0
let resourceRequest = 0
watch(() => [store.infoOpen, tab.value, period.value], async () => {
  const current = ++request
  if (!store.infoOpen || tab.value === 'members') { loading.value = false; error.value = ''; return }
  error.value = ''
  loading.value = true
  try {
    if (tab.value === 'wishes') await store.loadWishes()
    else {
      stats.value = undefined
      rankings.value = undefined
      const result = await store.roomInformation(period.value)
      if (result && current === request) [stats.value, rankings.value] = result
    }
  } catch (cause) { if (current === request) error.value = errorText(cause) }
  finally { if (current === request) loading.value = false }
})
function openRank(kind: string, item: MediaStat): void {
  if (kind === 'topArtists') void store.navigateArtist(item.name)
  else if (kind === 'topAlbums') void store.navigateAlbum(item.name, item.id.includes(':') ? undefined : item.id)
  else void store.openReference({ kind: 'track', id: item.id, title: item.name, artist: item.artist })
}
function remove(member: Member): void { if (window.confirm(t('confirmRemove', { name: member.displayName || member.username }))) void store.removeMember(member) }
async function submitWish(): Promise<void> {
  submitting.value = true
  try { if (await store.createWish({ ...wishForm.value, title: wishForm.value.title.trim() })) { wishForm.value = { kind: 'track', title: '', artist: '', reason: '' }; status.value = 'pending' } }
  finally { submitting.value = false }
}
function startFulfill(wish: RoomWish): void { fulfilling.value = wish; resourceQuery.value = [wish.title, wish.artist].filter(Boolean).join(' '); chosen.value = []; resourceResults.value = []; response.value = ''; void search() }
async function search(): Promise<void> {
  const wish = fulfilling.value
  if (!wish) return
  const current = ++resourceRequest
  searching.value = true
  error.value = ''
  try {
    const result = await store.searchResources(resourceQuery.value)
    if (current !== resourceRequest || fulfilling.value?.wishID !== wish.wishID) return
    resourceResults.value = wish.kind === 'track' ? result.songs.map(store.trackReference) : wish.kind === 'album' ? result.albums.map(album => ({ kind: 'album', id: album.id, title: album.name, artist: album.artist })) : result.artists.map(artist => ({ kind: 'artist', id: artist.id, title: artist.name }))
  } catch (cause) { error.value = errorText(cause) }
  finally { if (current === resourceRequest) searching.value = false }
}
function choose(item: ChatReference): void {
  chosen.value = chosen.value.some(resource => resource.id === item.id) ? chosen.value.filter(resource => resource.id !== item.id) : [...chosen.value, item].slice(0, 5)
}
async function fulfill(): Promise<void> {
  if (!fulfilling.value) return
  submitting.value = true
  try { if (await store.fulfillWish(fulfilling.value.wishID, { responseNote: response.value, resources: chosen.value.map(store.wishResource) })) { fulfilling.value = undefined; status.value = 'fulfilled' } }
  finally { submitting.value = false }
}
function deleteWish(id: string): void { if (window.confirm(t('confirmDelete'))) void store.deleteWish(id) }
</script>
<template>
  <RoomDialog :open="store.infoOpen" :title="store.room?.name || t('appTitle')" wide @close="store.infoOpen = false">
    <nav class="segmented wrap" :aria-label="t('appTitle')"><button v-for="item in ['members', 'statistics', 'wishes']" :key="item" :class="{ active: tab === item }" :aria-pressed="tab === item" @click="tab = item">{{ t(item) }}</button></nav>
    <template v-if="tab === 'members'">
      <section v-for="online in [true, false]" :key="String(online)"><h3>{{ t(online ? 'online' : 'offline') }} · {{ store.members.filter(member => member.online === online).length }}</h3><ul class="member-list"><li v-for="member in store.members.filter(item => item.online === online)" :key="member.username"><button class="member-summary" @click="store.profileUsername = member.username"><span :class="['member-dot', { offline: !member.online }]" /><span class="user-avatar">{{ (member.displayName || member.username).slice(0, 1) }}</span><span>{{ member.displayName || member.username }}<small>@{{ member.username }}</small></span><span class="role-badge">{{ t(member.role) }}</span></button><button v-if="store.canManage && member.role !== 'owner' && !store.sameUser(member.username)" class="icon-button danger" :aria-label="t('remove')" @click="remove(member)"><i aria-hidden="true" class="mdi mdi-account-remove-outline" /></button></li></ul></section>
    </template>
    <template v-else-if="tab === 'statistics'">
      <nav class="segmented wrap" :aria-label="t('statistics')"><button v-for="item in periods" :key="item" :class="{ active: period === item }" @click="period = item">{{ t(item) }}</button></nav>
      <template v-if="stats && !loading">
        <div class="stat-grid"><div><strong>{{ stats.totalPlayCount }}</strong><span>{{ t('plays') }}</span></div><div><strong>{{ t('minutes', { count: Math.floor(stats.totalPlayedSeconds / 60) }) }}</strong><span>{{ t('listeningTime') }}</span></div><div><strong>{{ stats.totalChatCount }}</strong><span>{{ t('chatCount') }}</span></div><div><strong>{{ stats.activeMemberCount }}</strong><span>{{ t('activeMembers') }}</span></div></div>
        <div v-if="rankings" class="rank-columns"><section v-for="kind in memberRanks" :key="kind.key"><h3>{{ t(kind.label) }}</h3><ol class="podium"><li v-for="(member, index) in (rankings[kind.key] || []).slice(0, 3)" :key="member.username"><button @click="store.profileUsername = member.username"><span class="medal">{{ ['🥇', '🥈', '🥉'][index] }}</span><strong>{{ member.displayName || member.username }}</strong><small>{{ member.count }}</small></button></li></ol><p v-if="!rankings[kind.key]?.length">{{ t('empty') }}</p></section></div>
        <div v-if="rankings" class="rank-columns"><section v-for="kind in resourceRanks" :key="kind"><h3>{{ t(kind) }}</h3><ol class="ranking-list"><li v-for="item in rankings[kind] || []" :key="item.id"><button class="text-link" @click="openRank(kind, item)">{{ item.name }}</button><small>{{ item.count }}</small><button v-if="kind === 'topTracks'" class="icon-button" :aria-label="t('queueSong')" :disabled="!store.queueRemaining || store.busy" @click="store.queueReference({ kind: 'track', id: item.id, title: item.name, artist: item.artist })"><i aria-hidden="true" class="mdi mdi-plus" /></button></li></ol></section></div>
        <h3>{{ t('activity') }} · UTC</h3><div class="activity-chart" role="list" :aria-label="t('activity')"><div v-for="hour in hours" :key="hour.hour" class="activity-column" role="listitem" :aria-label="`${hour.hour}:00 · ${t('minutes', { count: Math.floor(hour.seconds / 60) })}`" :title="`${hour.hour}:00 · ${t('minutes', { count: Math.floor(hour.seconds / 60) })}`"><span :style="{ height: `${Math.max(2, hour.seconds / maxActivity * 100)}%` }" /><small>{{ hour.hour % 3 === 0 ? hour.hour : '' }}</small></div></div>
      </template>
    </template>
    <template v-else>
      <nav class="segmented" :aria-label="t('wishes')"><button v-for="item in (['pending', 'fulfilled'] as const)" :key="item" :class="{ active: status === item }" @click="status = item">{{ t(item) }}</button></nav>
      <form class="dialog-form wish-form" @submit.prevent="submitWish"><h3>{{ t('newWish') }}</h3><div class="form-columns"><label>{{ t('kind') }}<select v-model="wishForm.kind"><option v-for="kind in ['track', 'album', 'artist']" :key="kind" :value="kind">{{ t(kind) }}</option></select></label><label>{{ t('title') }}<input v-model="wishForm.title" required maxlength="200" /></label></div><label>{{ t('artist') }}<input v-model="wishForm.artist" maxlength="200" /></label><label>{{ t('reason') }}<textarea v-model="wishForm.reason" maxlength="1000" rows="2" /></label><button class="primary-button" :disabled="submitting || !wishForm.title.trim()">{{ t('submit') }}</button></form>
      <form v-if="fulfilling" class="dialog-form fulfillment-form" @submit.prevent="fulfill"><h3>{{ t('fulfill') }} · {{ fulfilling.title }}</h3><label>{{ t('searchLibrary') }}<span class="search-inline"><input v-model="resourceQuery" /><button type="button" class="secondary-button" :disabled="searching" @click="search">{{ t('search') }}</button></span></label><p>{{ t('attachResource') }}</p><div class="resource-results"><button v-for="item in resourceResults" :key="item.id" type="button" :class="['resource-choice', { selected: chosen.some(resource => resource.id === item.id) }]" :aria-pressed="chosen.some(resource => resource.id === item.id)" @click="choose(item)">{{ item.title }} · {{ item.artist }}</button></div><p v-if="searching">{{ t('loading') }}</p><label>{{ t('response') }}<textarea v-model="response" maxlength="1000" rows="2" /></label><div class="toolbar"><button class="primary-button" :disabled="!chosen.length || submitting">{{ t('fulfill') }}</button><button type="button" class="secondary-button" @click="fulfilling = undefined">{{ t('cancel') }}</button></div></form>
      <article v-for="wish in visibleWishes" :key="wish.wishID" class="wish-card"><header><strong>{{ wish.title }}</strong><span class="count-pill">{{ t(wish.kind) }}</span></header><p v-if="wish.artist"><button class="text-link" @click="store.navigateArtist(wish.artist)">{{ wish.artist }}</button></p><p>{{ wish.reason }}</p><button class="text-link" @click="store.profileUsername = wish.authorUsername">{{ wish.authorDisplayName || wish.authorUsername }}</button><p v-if="wish.responseNote">{{ wish.responseNote }}</p><div class="toolbar"><template v-for="resource in wish.resources || []" :key="resource.url"><button class="secondary-button compact" @click="store.openWishResource(resource)">{{ resource.title }}</button><button v-if="store.wishReference(resource)?.kind === 'track'" class="icon-button" :disabled="!store.queueRemaining || store.busy" :aria-label="t('queueSong')" @click="store.queueWishResource(resource)"><i aria-hidden="true" class="mdi mdi-plus" /></button></template><button v-if="wish.status === 'pending' && store.canFulfillWish" class="secondary-button compact" @click="startFulfill(wish)">{{ t('fulfill') }}</button><button v-if="store.canManage || (wish.status === 'pending' && store.sameUser(wish.authorUsername))" class="icon-button danger" :aria-label="t('delete')" @click="deleteWish(wish.wishID)"><i aria-hidden="true" class="mdi mdi-delete-outline" /></button></div></article>
      <p v-if="!visibleWishes.length && !loading">{{ t('empty') }}</p>
    </template>
    <div v-if="loading" class="panel-loading"><span class="spinner" />{{ t('loading') }}</div><p v-if="error || store.error" class="media-error" role="alert">{{ error || errorText(store.errorCode) }}</p>
  </RoomDialog>
</template>
