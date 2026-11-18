<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoomStore } from '../stores/room'
import { useI18n } from '../lib/i18n'
import type { ChatMessage, SubsonicSong } from '../types'
import TrackLinks from './TrackLinks.vue'
const store = useRoomStore()
const { t, locale } = useI18n()
const text = ref('')
const reply = ref<ChatMessage>()
const sending = ref(false)
const reviewing = ref(false)
const stickersOpen = ref(false)
const packID = ref('')
const stickerID = ref('')
const list = ref<HTMLElement>()
const composer = ref<HTMLTextAreaElement>()
const mentionQuery = ref<string | null>(null)
const mentionStart = ref(0)
const mentionEnd = ref(0)
const songResults = ref<SubsonicSong[]>([])
const activeSuggestion = ref(0)
const now = ref(Date.now())
let searchTimer: ReturnType<typeof setTimeout> | undefined
let clockTimer: ReturnType<typeof setInterval> | undefined
let searchRevision = 0
let loadingOlder = false
const currentPack = computed(() => store.stickerPacks.find(item => item.packID === packID.value) || store.stickerPacks[0])
const selectedSticker = computed(() => store.stickerPacks.flatMap(item => item.stickers).find(item => item.stickerID === stickerID.value))
const suggestions = computed(() => {
  if (mentionQuery.value === null) return []
  const query = mentionQuery.value.toLocaleLowerCase()
  return [
    ...store.members.filter(member => `${member.displayName} ${member.username}`.toLocaleLowerCase().includes(query)).slice(0, 5).map(member => ({ key: `user:${member.username}`, label: `@${member.displayName || member.username}`, member, song: undefined as SubsonicSong | undefined })),
    ...songResults.value.slice(0, 5).map(song => ({ key: `song:${song.id}`, label: `♫ ${song.title} · ${song.artist || ''}`, song, member: undefined })),
  ]
})
function inspectMention(): void {
  const cursor = composer.value?.selectionStart ?? text.value.length
  const match = text.value.slice(0, cursor).match(/(?:^|\s)@([^@\n]{0,60})$/)
  clearTimeout(searchTimer)
  const revision = ++searchRevision
  songResults.value = []
  activeSuggestion.value = 0
  if (!match) { mentionQuery.value = null; return }
  mentionQuery.value = match[1]
  mentionStart.value = cursor - match[1].length - 1
  mentionEnd.value = cursor
  if (!match[1].trim()) return
  searchTimer = setTimeout(async () => {
    try { const result = await store.searchResources(match[1]); if (revision === searchRevision) songResults.value = result.songs }
    catch (cause) { if (revision === searchRevision) store.failTransient(cause) }
  }, 250)
}
function chooseSuggestion(index: number): void {
  const item = suggestions.value[index]
  if (!item) return
  if (item.member) store.suggestUserToChat(item.member)
  else if (item.song) store.suggestTrackToChat(item.song)
  text.value = text.value.slice(0, mentionStart.value) + (item.member ? `@${item.member.username} ` : '') + text.value.slice(mentionEnd.value)
  mentionQuery.value = null
  ++searchRevision
  void nextTick(() => composer.value?.focus())
}
function keydown(event: KeyboardEvent): void {
  if (mentionQuery.value === null) return
  if (event.key === 'Escape') { event.preventDefault(); mentionQuery.value = null; ++searchRevision }
  if (suggestions.value.length && ['ArrowDown', 'ArrowUp'].includes(event.key)) {
    event.preventDefault()
    activeSuggestion.value = (activeSuggestion.value + (event.key === 'ArrowDown' ? 1 : -1) + suggestions.value.length) % suggestions.value.length
  }
  if (event.key === 'Enter' && suggestions.value.length && !event.isComposing) { event.preventDefault(); chooseSuggestion(activeSuggestion.value) }
}
function sticker(id?: string) { return store.stickerPacks.flatMap(pack => pack.stickers).find(item => item.stickerID === id) }
function quote(message: ChatMessage) {
  return store.messages.find(item => item.messageID === message.replyToID) || message.replyPreview
}
function canRecall(message: ChatMessage): boolean {
  const age = now.value - Date.parse(message.createdAt)
  return store.sameUser(message.senderUsername) && age >= 0 && age <= 300_000
}
async function send(): Promise<void> {
  if (sending.value || !(text.value.trim() || store.draftReferences.length || stickerID.value || store.draftMentions.length)) return
  sending.value = true
  try {
    if (await store.sendChat({ messageType: reviewing.value ? 'review' : 'chat', body: [text.value.trim(), ...store.draftMentions.filter(username => !text.value.includes(`@${username}`)).map(username => `@${username}`)].filter(Boolean).join(' '), references: [...store.draftReferences], mentions: [...store.draftMentions], replyToID: reply.value?.messageID, stickerID: stickerID.value || undefined })) {
      text.value = ''; reviewing.value = false; reply.value = undefined; stickerID.value = ''; stickersOpen.value = false; mentionQuery.value = null
      await nextTick(); scrollBottom(); composer.value?.focus()
    }
  } finally { sending.value = false }
}
function review(key: string): void {
  reviewing.value = true
  if (store.currentTrack) store.suggestTrackToChat(store.toSong(store.currentTrack))
  text.value = [text.value, t(key)].filter(Boolean).join(' ')
  composer.value?.focus()
}
function scrollBottom(): void { if (list.value) list.value.scrollTop = list.value.scrollHeight }
async function older(): Promise<void> {
  const height = list.value?.scrollHeight || 0
  const top = list.value?.scrollTop || 0
  loadingOlder = true
  await store.loadOlderMessages()
  await nextTick()
  if (list.value) list.value.scrollTop = top + list.value.scrollHeight - height
  loadingOlder = false
}
watch(() => store.messages.length, async () => {
  const atBottom = !list.value || list.value.scrollHeight - list.value.scrollTop - list.value.clientHeight < 90
  await nextTick()
  if (atBottom && !loadingOlder) scrollBottom()
})
onMounted(async () => { clockTimer = setInterval(() => { now.value = Date.now() }, 15_000); await store.loadMessages(); await nextTick(); scrollBottom() })
onBeforeUnmount(() => { clearTimeout(searchTimer); clearInterval(clockTimer); ++searchRevision })
function deleteMessage(message: ChatMessage): void { if (window.confirm(t('confirmDelete'))) void store.deleteChat(message) }
</script>
<template>
  <section class="work-panel chat-panel" aria-labelledby="chat-title">
    <header class="panel-heading"><h2 id="chat-title">{{ t('chat') }}</h2><button class="icon-button" :aria-label="t('refresh')" :disabled="store.messagesLoading" @click="store.loadMessages()"><i aria-hidden="true" class="mdi mdi-refresh" /></button></header>
    <div ref="list" class="message-list" role="log" :aria-label="t('chat')" aria-live="polite" aria-relevant="additions">
      <button v-if="store.hasMoreMessages" class="secondary-button load-more" :disabled="store.messagesLoading" @click="older">{{ t('olderMessages') }}</button>
      <p v-if="store.messagesLoading">{{ t('loading') }}</p>
      <p v-else-if="!store.messages.length" class="panel-empty">{{ t('empty') }}</p>
      <article v-for="message in store.messages" :key="message.messageID" :class="['chat-message', { own: store.sameUser(message.senderUsername) }]">
        <button class="user-avatar" :aria-label="`${t('profile')} ${message.senderDisplayName || message.senderUsername}`" @click="store.profileUsername = message.senderUsername">{{ (message.senderDisplayName || message.senderUsername || '?').slice(0, 1).toUpperCase() }}</button>
        <div class="message-content"><header><button class="text-link" @click="store.profileUsername = message.senderUsername">{{ message.senderDisplayName || message.senderUsername }}</button><span v-if="store.memberProfiles[message.senderUsername.toLowerCase()]?.level.level" class="level-chip">{{ t('level', { level: store.memberProfiles[message.senderUsername.toLowerCase()]?.level.level }) }}</span><span v-if="store.memberProfiles[message.senderUsername.toLowerCase()]?.selectedBadgeID" :title="t('badge')">🏅</span><time>{{ new Date(message.createdAt).toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' }) }}</time></header>
          <p v-if="message.deletedAt || message.recalledAt" class="muted-copy">{{ t(message.deletedAt ? 'deleted' : 'recalled') }}</p>
          <template v-else>
            <blockquote v-if="quote(message)"><strong>{{ quote(message)?.senderDisplayName }}</strong><span>{{ quote(message)?.deletedAt ? t('deleted') : quote(message)?.recalledAt ? t('recalled') : quote(message)?.body }}</span></blockquote>
            <p class="message-text">{{ message.body }}</p>
            <div v-if="message.mentions?.length" class="toolbar"><button v-for="username in message.mentions" :key="username" class="text-link" @click="store.profileUsername = username">@{{ username }}</button></div>
            <span v-if="sticker(message.stickerID)" class="chat-sticker" role="img" :aria-label="t('sticker', { emoji: sticker(message.stickerID)?.emoji || '♪' })">{{ sticker(message.stickerID)?.emoji || '♪' }}</span>
            <article v-for="reference in message.references || []" :key="`${reference.kind}:${reference.id}`" class="chat-track-card"><strong>{{ reference.title }}</strong><TrackLinks :artist="reference.artist" :artist-id="store.referencedSongs[reference.id]?.artistId" :album="store.referencedSongs[reference.id]?.album" :album-id="store.referencedSongs[reference.id]?.albumId" /><div class="toolbar"><button class="secondary-button compact" :disabled="!store.queueRemaining || store.busy" @click="store.queueReference(reference)"><i aria-hidden="true" class="mdi mdi-plus" />{{ t('queueSong') }}</button><button class="secondary-button compact" @click="store.openReference(reference)">{{ t(reference.kind === 'artist' ? 'openArtist' : 'openAlbum') }}</button></div></article>
            <div class="message-actions"><button class="text-link" @click="reply = message; composer?.focus()">{{ t('reply') }}</button><button v-if="canRecall(message)" class="text-link" @click="store.recallChat(message)">{{ t('recall') }}</button><button v-if="store.canManage" class="text-link danger" @click="deleteMessage(message)">{{ t('delete') }}</button></div>
          </template>
        </div>
      </article>
    </div>
    <div v-if="store.currentTrack" class="quick-reviews" :aria-label="t('quickReviews')"><button v-for="key in ['reviewLove', 'reviewVibe', 'reviewAgain']" :key="key" @click="review(key)">{{ t(key) }}</button></div>
    <form class="chat-composer" @submit.prevent="send">
      <div v-if="reply" class="draft-chip"><span>{{ t('reply') }} · {{ reply.senderDisplayName }}: {{ reply.body }}</span><button type="button" class="icon-button" :aria-label="t('cancel')" @click="reply = undefined"><i aria-hidden="true" class="mdi mdi-close" /></button></div>
      <div class="draft-chips"><span v-for="(reference, index) in store.draftReferences" :key="reference.id" class="draft-chip">♫ {{ reference.title }}<button type="button" :aria-label="t('remove')" @click="store.draftReferences.splice(index, 1)">×</button></span><span v-for="(username, index) in store.draftMentions" :key="username" class="draft-chip">@{{ username }}<button type="button" :aria-label="t('remove')" @click="store.draftMentions.splice(index, 1)">×</button></span><span v-if="selectedSticker" class="draft-chip">{{ selectedSticker.emoji }}<button type="button" :aria-label="t('remove')" @click="stickerID = ''">×</button></span></div>
      <div class="composer-field"><textarea ref="composer" v-model="text" rows="3" maxlength="2000" :placeholder="t('messagePlaceholder')" :aria-label="t('chat')" :aria-expanded="mentionQuery !== null && suggestions.length > 0" :aria-activedescendant="suggestions.length ? `suggestion-${activeSuggestion}` : undefined" aria-controls="chat-suggestions" @input="inspectMention" @click="inspectMention" @keydown="keydown" />
        <div v-if="mentionQuery !== null && suggestions.length" id="chat-suggestions" class="mention-popup" role="listbox" :aria-label="t('mention')"><button v-for="(item, index) in suggestions" :id="`suggestion-${index}`" :key="item.key" type="button" role="option" :aria-selected="index === activeSuggestion" :class="{ active: index === activeSuggestion }" @mousedown.prevent @click="chooseSuggestion(index)">{{ item.label }}</button></div>
      </div>
      <div v-if="stickersOpen" class="sticker-picker"><nav class="segmented wrap" :aria-label="t('stickers')"><button v-for="(pack, index) in store.stickerPacks" :key="pack.packID" type="button" :class="{ active: currentPack?.packID === pack.packID }" @click="packID = pack.packID">{{ t(['vibes', 'reactions'].includes(pack.packID) ? `pack.${pack.packID}` : 'stickerPack', { count: index + 1 }) }}</button></nav><div class="sticker-grid"><button v-for="sticker in currentPack?.stickers || []" :key="sticker.stickerID" type="button" :aria-label="sticker.labels?.[locale] || sticker.labels?.en || t('sticker', { emoji: sticker.emoji || '♪' })" @click="stickerID = sticker.stickerID; stickersOpen = false">{{ sticker.emoji || '♪' }}</button></div><p v-if="!store.stickerPacks.length">{{ t('empty') }}</p></div>
      <div class="composer-actions"><button type="button" class="icon-button" :aria-label="t('stickers')" :aria-expanded="stickersOpen" @click="stickersOpen = !stickersOpen"><i aria-hidden="true" class="mdi mdi-sticker-emoji" /></button><button class="primary-button" :disabled="sending || !(text.trim() || store.draftReferences.length || stickerID || store.draftMentions.length)">{{ t(sending ? 'loading' : 'send') }}<i aria-hidden="true" class="mdi mdi-send" /></button></div>
    </form>
  </section>
</template>
