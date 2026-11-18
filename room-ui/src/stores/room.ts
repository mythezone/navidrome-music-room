import { computed, onScopeDispose, ref, shallowRef, watch } from 'vue'
import { defineStore } from 'pinia'
import { t, errorText } from '../lib/i18n'
import { eqFrequencies, eqPresets, prefersNativeAudio, readEq, type EqPreset } from '../lib/eq'

import { GatewayClient, GatewayError, navidromeLogin } from '../lib/gateway'
import { clearInvitation, readProof, storeNavidromeLogin, resourceFromURL, safeResourceURL } from '../lib/location'
import { SubsonicClient } from '../lib/subsonic'
import { clamp, effectivePosition, needsHardSeek } from '../lib/sync'
import type {
  ChatMessage, ChatPayload, ChatReference, StickerPack, RoomPermissions, UpdateNotice, RoomSettings,
  RoomWish, WishResource, WishPayload, WishFulfillment, RoomStatistics, RoomRankings, MemberProfile, Period, PlaybackCommand, TrackRef,
  CatalogResults,
  GatewayEvent,
  GatewaySession,
  HistoryEntry,
  LyricLine,
  Member,
  NavidromeProof,
  PlaybackState,
  QueueEntry,
  Room,
  RoomSnapshot,
  SubsonicAlbum,
  SubsonicArtist,
  SubsonicPlaylist,
  SubsonicSong,
} from '../types'

type Phase = 'booting' | 'login' | 'joining' | 'ready' | 'error'
type RoomTab = 'playing' | 'queue' | 'history' | 'catalog' | 'favorites' | 'chat'
type CatalogSection = 'songs' | 'albums' | 'artists'

const emptyCatalog = (): CatalogResults => ({ songs: [], albums: [], artists: [] })

export const useRoomStore = defineStore('room', () => {
  const phase = ref<Phase>('booting')
  const roomID = ref('')
  const gatewayPrefix = ref('')
  const invitation = ref('')
  const proof = shallowRef<NavidromeProof | null>(null)
  const session = shallowRef<GatewaySession | null>(null)
  const snapshot = shallowRef<RoomSnapshot | null>(null)
  const activeTab = ref<RoomTab>('playing')
  const connected = ref(false)
  const reconnecting = ref(false)
  const error = ref('')
  const errorCode = ref('')
  const noticeMessage = ref<{ key: string; params?: Record<string, string | number> } | null>(null)
  const notice = computed(() => noticeMessage.value ? t(noticeMessage.value.key, noticeMessage.value.params) : '')
  const busy = ref(false)

  const currentTime = ref(0)
  const audioDuration = ref(0)
  const volume = ref(0.8)
  const isListening = ref(false)
  const autoplayBlocked = ref(false)
  const audioLoading = ref(false)
  const audioFailed = ref(false)
  const audioError = computed(() => audioFailed.value ? errorText('track_not_found') : '')
  const lyrics = ref<LyricLine[]>([])

  const folderID = ref(0)
  const folders = ref<Array<{ id: number; name: string }>>([])
  const catalog = shallowRef<CatalogResults>(emptyCatalog())
  const catalogMode = ref<'browse' | 'search' | 'album' | 'artist'>('browse')
  const catalogSection = ref<CatalogSection>('songs')
  const catalogTitle = computed(() => selectedAlbum.value?.name || selectedArtist.value?.name || (catalogMode.value === 'search' ? t('searchResults', { query: catalogQuery.value.trim() }) : t('catalog')))
  const catalogQuery = ref('')
  const catalogLoading = ref(false)
  const catalogFailure = shallowRef<unknown>(null)
  const catalogError = computed(() => catalogFailure.value ? errorText(catalogFailure.value) : '')
  const selectedAlbum = shallowRef<SubsonicAlbum | null>(null)
  const selectedArtist = shallowRef<SubsonicArtist | null>(null)
  const artistAlbums = shallowRef<SubsonicAlbum[]>([])
  const favorites = shallowRef<CatalogResults>(emptyCatalog())
  const playlists = shallowRef<SubsonicPlaylist[]>([])
  const selectedPlaylist = shallowRef<SubsonicPlaylist | null>(null)
  const favoritesLoading = ref(false)

  let gateway: GatewayClient | null = null
  let subsonic: SubsonicClient | null = null
  let socket: WebSocket | null = null
  let reconnectTimer: number | null = null
  let reconnectAttempt = 0
  let connectionGeneration = 0
  let sessionTimer: number | null = null
  let progressTimer: number | null = null
  let driftTimer: number | null = null
  let noticeTimer: number | null = null
  let manualDisconnect = false
  let audio: HTMLAudioElement | null = null
  let preloadAudio: HTMLAudioElement | null = null
  let loadedTrackID = ''
  let pendingSeek: number | null = null

  const room = computed<Room | null>(() => snapshot.value?.room || null)
  const self = computed<Member | null>(() => snapshot.value?.self || null)
  const playback = computed<PlaybackState | null>(() => snapshot.value?.playback || null)
  const queue = computed<QueueEntry[]>(() => snapshot.value?.queue || [])
  const history = computed<HistoryEntry[]>(() => snapshot.value?.history || [])
  const members = computed<Member[]>(() => snapshot.value?.members || [])
  const currentTrack = computed(() => playback.value?.currentTrack)
  const duration = computed(() => Math.max(audioDuration.value, currentTrack.value?.durationSeconds || 0))
  const canManage = computed(() => snapshot.value?.permissions?.canManage ?? Boolean(
    session.value?.user.adminRole || self.value?.role === 'owner' || self.value?.role === 'manager' ||
    (room.value && session.value && room.value.ownerUsername.toLowerCase() === session.value.user.username.toLowerCase()),
  ))
  const canControl = computed(() => snapshot.value?.permissions?.canControl ?? canManage.value)
  const canSkip = computed(() => snapshot.value?.permissions?.canSkip ?? Boolean(canManage.value || room.value?.allowMemberSkip ||
    (currentTrack.value && playback.value?.contributorUsername && sameUser(playback.value.contributorUsername))))
  const canReorderQueue = computed(() => snapshot.value?.permissions?.canReorderQueue ?? canManage.value)
  const canFulfillWish = computed(() => snapshot.value?.permissions?.canFulfillWish ?? canManage.value)
  const permissions = computed<RoomPermissions>(() => ({ canManage: canManage.value, canControl: canControl.value, canSkip: canSkip.value,
    canReorderQueue: canReorderQueue.value, canFulfillWish: canFulfillWish.value }))
  const updateNotice = ref<UpdateNotice | null>(null)
  const immersive = ref(false)
  const profileUsername = ref('')
  const infoOpen = ref(false)
  const adminURL = computed(() => `${gatewayPrefix.value}/admin/`)
  const messages = ref<ChatMessage[]>([])
  const memberProfiles = ref<Record<string, MemberProfile>>({})
  const profileRequests = new Set<string>()
  const referencedSongs = ref<Record<string, SubsonicSong>>({})
  const hasMoreMessages = ref(false)
  const messagesLoading = ref(false)
  const unreadChatCount = ref(0)
  const chatVisible = ref(false)
  const stickerPacks = ref<StickerPack[]>([])
  const draftReferences = ref<ChatReference[]>([])
  const draftMentions = ref<string[]>([])
  const wishes = ref<RoomWish[]>([])
  const hasMoreHistory = ref(true)
  const historyLoading = ref(false)
  const nativeBackgroundPlayback = prefersNativeAudio()
  const eqSettings = readEq()
  const eqEnabled = ref(eqSettings.enabled && !nativeBackgroundPlayback)
  const eqPreset = ref<EqPreset>(eqSettings.preset)
  const eqBands = ref(eqFrequencies.map((frequency, index) => ({ frequency, gain: eqSettings.gains[index] })))
  const eqAvailable = !nativeBackgroundPlayback && Boolean(window.AudioContext || (window as Window & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext)
  let audioContext: AudioContext | null = null
  let eqSource: MediaElementAudioSourceNode | null = null
  let eqFilters: BiquadFilterNode[] = []
  function sameUser(username: string): boolean {
    return Boolean(username && username.toLowerCase() === (self.value?.username || session.value?.user.username || '').toLowerCase())
  }
  watch(chatVisible, visible => { if (visible) unreadChatCount.value = 0 }, { flush: 'sync' })
  watch([canSkip, canControl], updateMediaSession)
  const onlineCount = computed(() => room.value?.onlineCount || members.value.filter((member) => member.online).length)
  const queueRemaining = computed(() => {
    if (!room.value || !session.value) return 0
    const own = queue.value.filter((entry) => entry.contributorUsername.toLowerCase() === session.value!.user.username.toLowerCase()).length
    return Math.max(0, room.value.queueLimit - own)
  })
  const activeLyricIndex = computed(() => {
    let active = -1
    lyrics.value.forEach((line, index) => {
      if (line.time <= currentTime.value) active = index
    })
    return active
  })
  const deepLink = computed(() => {
    if (!session.value || !roomID.value) return ''
    const query = new URLSearchParams({
      server: session.value.navidromeBaseURL,
      gateway: session.value.gatewayBaseURL,
      room: roomID.value,
    })
    if (invitation.value) query.set('invite', invitation.value)
    return `musicmate://join?${query.toString()}`
  })

  function configure(context: { roomID: string; gatewayPrefix: string; invitation: string }): void {
    roomID.value = context.roomID
    gatewayPrefix.value = context.gatewayPrefix
    invitation.value = context.invitation
    gateway = new GatewayClient(context.gatewayPrefix)
    proof.value = readProof()
  }

  async function start(): Promise<void> {
    if (!roomID.value || !gateway) {
      fail(new Error(errorText('room_id_invalid')), 'room_id_invalid')
      return
    }
    if (!proof.value) {
      phase.value = 'login'
      return
    }
    await enterWithProof(proof.value)
  }

  async function login(username: string, password: string): Promise<void> {
    busy.value = true
    error.value = ''
    try {
      const response = await navidromeLogin(username.trim(), password)
      proof.value = storeNavidromeLogin(response)
      await enterWithProof(proof.value)
    } catch (cause) {
      fail(cause, cause instanceof GatewayError ? cause.code : 'login_failed', 'login')
    } finally {
      busy.value = false
    }
  }

  async function enterWithProof(nextProof: NavidromeProof): Promise<void> {
    if (!gateway) return
    phase.value = 'joining'
    error.value = ''
    errorCode.value = ''
    try {
      session.value = await gateway.exchange(nextProof)
      subsonic = new SubsonicClient(session.value.navidromeBaseURL, nextProof)
      subsonic.assertSameOrigin()
      scheduleSessionRefresh()
      try {
        await gateway.join(roomID.value)
      } catch (cause) {
        if (!(cause instanceof GatewayError) || cause.code !== 'membership_required' || !invitation.value) throw cause
        await gateway.redeem(roomID.value, invitation.value)
        await gateway.join(roomID.value)
      }
      if (invitation.value) clearInvitation()
      await refreshSnapshot(true)
      await loadFoldersAndCatalog()
      phase.value = 'ready'
      const resource = resourceFromURL(window.location.href)
      if (resource) void openReference(resource)
      connectRealtime()
      startClocks()
    } catch (cause) {
      const code = cause instanceof GatewayError ? cause.code : cause instanceof Error && 'code' in cause ? String(cause.code) : 'join_failed'
      fail(cause, code, proof.value ? 'error' : 'login')
    }
  }

  function fail(cause: unknown, code: string, nextPhase: Phase = 'error'): void {
    error.value = errorText(code)
    errorCode.value = code
    phase.value = nextPhase
  }

  function showNotice(key: string, params?: Record<string, string | number>): void {
    noticeMessage.value = { key, params }
    if (noticeTimer !== null) window.clearTimeout(noticeTimer)
    noticeTimer = window.setTimeout(() => { noticeMessage.value = null }, 2800)
  }

  async function refreshSnapshot(forceAudio = false): Promise<void> {
    if (!gateway) return
    try {
      applySnapshot(await gateway.snapshot(roomID.value), forceAudio)
    } catch (cause) {
      if (cause instanceof GatewayError && cause.code === 'revision_conflict') return
      throw cause
    }
  }

  function applySnapshot(next: RoomSnapshot, forceAudio = false): void {
    const previous = snapshot.value?.playback
    if (next.updateNotice) updateNotice.value = next.updateNotice
    hasMoreHistory.value = next.history?.length >= 50
    snapshot.value = {
      ...next,
      members: Array.isArray(next.members) ? next.members : [],
      queue: Array.isArray(next.queue) ? next.queue : [],
      history: mergeHistory(next.history || [], history.value),
    }
    applyAudioPlayback(next.playback, forceAudio || previous?.currentTrack?.id !== next.playback.currentTrack?.id)
  }

  function scheduleSessionRefresh(): void {
    if (sessionTimer !== null) window.clearTimeout(sessionTimer)
    const expiry = Date.parse(session.value?.expiresAt || '')
    const delay = Number.isFinite(expiry) ? Math.max(30_000, expiry - Date.now() - 60_000) : 12 * 60_000
    sessionTimer = window.setTimeout(() => { void refreshSession() }, delay)
  }

  async function refreshSession(): Promise<void> {
    if (!gateway || !proof.value || phase.value !== 'ready') return
    try {
      session.value = await gateway.exchange(proof.value)
      scheduleSessionRefresh()
      connectRealtime(true)
    } catch (cause) {
      fail(cause, cause instanceof GatewayError ? cause.code : 'session_refresh_failed')
    }
  }

  async function loadFoldersAndCatalog(): Promise<void> {
    if (!subsonic || !room.value) return
    folders.value = (await subsonic.folders()).filter((item) => room.value!.musicFolderIDs.includes(item.id))
    folderID.value = folders.value[0]?.id || room.value.musicFolderIDs[0] || 0
    await Promise.all([browseCatalog(), loadFavorites()])
  }

  async function browseCatalog(): Promise<void> {
    if (!subsonic || !folderID.value) return
    catalogLoading.value = true
    catalogFailure.value = null
    try {
      catalog.value = await subsonic.browse(folderID.value)
      catalogMode.value = 'browse'
      catalogSection.value = 'songs'
      selectedAlbum.value = null
      selectedArtist.value = null
      artistAlbums.value = []
    } catch (cause) {
      catalogFailure.value = cause
    } finally {
      catalogLoading.value = false
    }
  }

  async function searchCatalog(): Promise<void> {
    if (!subsonic || !folderID.value) return
    const query = catalogQuery.value.trim()
    if (!query) {
      await browseCatalog()
      return
    }
    catalogLoading.value = true
    catalogFailure.value = null
    try {
      const results = await subsonic.search(query, folderID.value)
      catalog.value = results
      catalogMode.value = 'search'
      selectedAlbum.value = null
      selectedArtist.value = null
      artistAlbums.value = []
      const currentResults = results[catalogSection.value]
      if (!currentResults.length) {
        catalogSection.value = results.songs.length ? 'songs' : results.albums.length ? 'albums' : 'artists'
      }
    } catch (cause) {
      catalogFailure.value = cause
    } finally {
      catalogLoading.value = false
    }
  }

  async function openAlbum(album: SubsonicAlbum): Promise<void> {
    if (!subsonic) return
    activeTab.value = 'catalog'
    immersive.value = false
    catalogFailure.value = null
    catalogLoading.value = true
    try {
      const fromArtist = catalogMode.value === 'artist' && artistAlbums.value.some(item => item.id === album.id)
      selectedAlbum.value = await subsonic.album(album.id)
      if (!fromArtist) { selectedArtist.value = null; artistAlbums.value = [] }
      catalogMode.value = 'album'
      catalogSection.value = 'songs'
    } catch (cause) {
      catalogFailure.value = cause
    } finally {
      catalogLoading.value = false
    }
  }

  async function openArtist(artist: SubsonicArtist): Promise<void> {
    if (!subsonic) return
    activeTab.value = 'catalog'
    immersive.value = false
    catalogFailure.value = null
    catalogLoading.value = true
    try {
      const albums = await subsonic.artistAlbums(artist.id)
      selectedArtist.value = artist
      selectedAlbum.value = null
      artistAlbums.value = albums
      catalogMode.value = 'artist'
      catalogSection.value = 'albums'
    } catch (cause) {
      catalogFailure.value = cause
    } finally {
      catalogLoading.value = false
    }
  }

  function selectCatalogSection(section: CatalogSection): void {
    catalogSection.value = section
    if (catalogMode.value === 'album' || catalogMode.value === 'artist') returnToCatalog()
  }

  function returnToCatalog(): void {
    if (selectedAlbum.value && selectedArtist.value) {
      selectedAlbum.value = null
      catalogMode.value = 'artist'
      catalogSection.value = 'albums'
      return
    }
    selectedAlbum.value = null
    selectedArtist.value = null
    artistAlbums.value = []
    const query = catalogQuery.value.trim()
    catalogMode.value = query ? 'search' : 'browse'
  }

  async function loadFavorites(): Promise<void> {
    if (!subsonic) return
    favoritesLoading.value = true
    try {
      const [starred, playlistItems] = await Promise.all([subsonic.starred(), subsonic.playlists()])
      favorites.value = starred
      playlists.value = playlistItems
    } catch (cause) {
      failTransient(cause)
    } finally {
      favoritesLoading.value = false
    }
  }

  async function openPlaylist(playlist: SubsonicPlaylist): Promise<void> {
    if (!subsonic) return
    favoritesLoading.value = true
    try {
      selectedPlaylist.value = await subsonic.playlist(playlist.id)
    } catch (cause) {
      failTransient(cause)
    } finally {
      favoritesLoading.value = false
    }
  }

  function isStarred(id: string, kind: 'song' | 'album' | 'artist' = 'song'): boolean {
    const items = kind === 'song' ? favorites.value.songs : kind === 'album' ? favorites.value.albums : favorites.value.artists
    return items.some(item => item.id === id)
  }

  async function toggleStar(item: SubsonicSong | SubsonicAlbum | SubsonicArtist, kind: 'song' | 'album' | 'artist' = 'song'): Promise<void> {
    if (!subsonic) return
    try {
      await subsonic.setStarred(kind, item.id, !isStarred(item.id, kind))
      await loadFavorites()
    } catch (cause) { failTransient(cause) }
  }

  async function addSong(song: SubsonicSong, playNext = false): Promise<void> {
    if (!gateway || !subsonic || !folderID.value || busy.value || !queueRemaining.value) return
    busy.value = true
    try {
      const result = await gateway.addTrack(roomID.value, subsonic.toTrack(song, folderID.value), playNext)
      if (!snapshot.value) return
      snapshot.value = { ...snapshot.value, queue: result.queue || [] }
      if (result.playback) applyPlaybackEvent(result.playback)
      showNotice('queued', { title: song.title })
    } catch (cause) {
      failTransient(cause)
    } finally {
      busy.value = false
    }
  }

  async function addAlbum(album: SubsonicAlbum): Promise<void> {
    if (!gateway || !subsonic || !folderID.value || busy.value) return
    busy.value = true
    let added = 0
    try {
      const full = album.song ? album : await subsonic.album(album.id)
      for (const song of full.song || []) {
        if (queueRemaining.value <= 0) break
        const result = await gateway.addTrack(roomID.value, subsonic.toTrack(song, folderID.value))
        if (!snapshot.value) break
        snapshot.value = { ...snapshot.value, queue: result.queue || [] }
        if (result.playback) applyPlaybackEvent(result.playback)
        added += 1
      }
      if (added > 0) showNotice('queued', { title: `${full.name} (${added})` })
    } catch (cause) {
      failTransient(cause)
    } finally {
      busy.value = false
    }
  }

  async function addPlaylist(playlist: SubsonicPlaylist): Promise<void> {
    if (!subsonic) return
    try {
      const full = playlist.entry ? playlist : await subsonic.playlist(playlist.id)
      for (const song of full.entry || []) {
        if (queueRemaining.value <= 0) break
        await addSong(song)
      }
    } catch (cause) { failTransient(cause) }
  }

  function failTransient(cause: unknown): void {
    const message = errorText(cause)
    errorCode.value = cause && typeof cause === 'object' && 'code' in cause ? String(cause.code) : 'network'
    error.value = message
    window.setTimeout(() => {
      if (error.value === message && phase.value === 'ready') error.value = ''
    }, 4500)
  }

  function connectRealtime(replace = false): void {
    if (!gateway || phase.value !== 'ready') return
    if (socket && !replace && socket.readyState <= WebSocket.OPEN) return
    manualDisconnect = false
    const generation = ++connectionGeneration
    const previous = socket
    socket = null
    previous?.close()
    reconnecting.value = reconnectAttempt > 0
    void gateway.websocketTicket(roomID.value).then((ticket) => {
      if (manualDisconnect || phase.value !== 'ready' || generation !== connectionGeneration) return
      const current = new WebSocket(ticket.webSocketURL)
      socket = current
      current.addEventListener('open', () => {
        if (socket !== current) return
        connected.value = true
        reconnecting.value = false
        reconnectAttempt = 0
        if (messages.value.length || chatVisible.value) void loadMessages()
      })
      current.addEventListener('message', (event) => {
        if (socket !== current) return
        try {
          handleEvent(JSON.parse(String(event.data)) as GatewayEvent)
        } catch {
          failTransient(new Error(errorText('request_invalid')))
        }
      })
      current.addEventListener('close', () => {
        if (socket !== current) return
        connected.value = false
        socket = null
        scheduleReconnect()
      })
      current.addEventListener('error', () => {
        current.close()
      })
    }).catch((cause) => {
      if (generation !== connectionGeneration || manualDisconnect) return
      failTransient(cause)
      scheduleReconnect()
    })
  }

  function scheduleReconnect(): void {
    if (manualDisconnect || phase.value !== 'ready' || reconnectTimer !== null) return
    reconnecting.value = true
    const delay = Math.min(10_000, 750 * 2 ** reconnectAttempt)
    reconnectAttempt += 1
    reconnectTimer = window.setTimeout(() => {
      reconnectTimer = null
      void refreshSnapshot(true).catch(failTransient).finally(() => connectRealtime())
    }, delay)
  }

  function handleEvent(event: GatewayEvent): void {
    if (event.type === 'snapshot') {
      applySnapshot(event.payload as RoomSnapshot, true)
      return
    }
    if (event.type === 'chat_message' || event.type === 'chat_updated') {
      const message = event.payload as ChatMessage
      const exists = messages.value.some(item => item.messageID === message.messageID)
      if (event.type === 'chat_message' || exists) upsertMessage(message)
      if (event.type === 'chat_updated') {
        messages.value = messages.value.map(item => item.replyToID === message.messageID
          ? { ...item, replyPreview: { messageID: message.messageID, senderUsername: message.senderUsername, senderDisplayName: message.senderDisplayName, body: message.body, stickerID: message.stickerID || '', recalledAt: message.recalledAt, deletedAt: message.deletedAt } } : item)
      }
      if (event.type === 'chat_message' && !exists && !chatVisible.value && !sameUser(message.senderUsername)) unreadChatCount.value += 1
      return
    }
    if (event.type === 'wish_updated') { upsertWish(event.payload as RoomWish); return }
    if (event.type === 'wish_deleted') {
      const id = (event.payload as { wishID: string }).wishID
      wishes.value = wishes.value.filter(item => item.wishID !== id)
      return
    }
    if (event.type === 'update_available') { updateNotice.value = event.payload as UpdateNotice; return }
    if (!snapshot.value) return
    if (event.type === 'playback') {
      const next = event.payload as PlaybackState
      if (next.revision < snapshot.value.playback.revision) return
      applyPlaybackEvent(next)
      void refreshSnapshot().catch(failTransient)
      return
    }
    if (event.type === 'queue') {
      snapshot.value = { ...snapshot.value, queue: Array.isArray(event.payload) ? event.payload as QueueEntry[] : [] }
      return
    }
    if (event.type === 'history') {
      snapshot.value = { ...snapshot.value, history: mergeHistory(Array.isArray(event.payload) ? event.payload as HistoryEntry[] : [], history.value) }
      return
    }
    if (event.type === 'room_updated') {
      snapshot.value = { ...snapshot.value, room: event.payload as Room }
      void refreshSnapshot().catch(failTransient)
      return
    }
    if (event.type === 'presence') {
      const payload = event.payload as { onlineUsernames?: string[]; onlineCount?: number; members?: Member[] }
      const online = new Set((payload.onlineUsernames || []).map((value) => value.toLowerCase()))
      const nextMembers = payload.members || snapshot.value.members.map((member) => ({
        ...member,
        online: online.has(member.username.toLowerCase()),
      }))
      snapshot.value = {
        ...snapshot.value,
        members: nextMembers,
        room: { ...snapshot.value.room, onlineCount: payload.onlineCount ?? nextMembers.filter((member) => member.online).length },
      }
    }
  }

  function applyPlaybackEvent(next: PlaybackState): void {
    if (!snapshot.value) return
    const previous = snapshot.value.playback
    snapshot.value = { ...snapshot.value, playback: next }
    applyAudioPlayback(next, previous.currentTrack?.id !== next.currentTrack?.id || previous.status !== next.status)
  }

  function ensureAudio(): HTMLAudioElement | null {
    if (audio || typeof Audio === 'undefined') return audio
    audio = new Audio()
    audio.dataset.musicRoomAudio = 'true'
    audio.hidden = true
    audio.preload = 'auto'
    audio.crossOrigin = 'use-credentials'
    audio.setAttribute('playsinline', '')
    audio.setAttribute('webkit-playsinline', 'true')
    audio.volume = volume.value
    audio.addEventListener('loadedmetadata', () => {
      if (!audio) return
      audioDuration.value = Number.isFinite(audio.duration) ? audio.duration : currentTrack.value?.durationSeconds || 0
      if (pendingSeek !== null) seekAudio(pendingSeek)
    })
    audio.addEventListener('canplay', () => {
      audioLoading.value = false
      if (pendingSeek !== null) seekAudio(pendingSeek)
      if (playback.value?.status === 'playing' && isListening.value) void safePlay()
    })
    audio.addEventListener('timeupdate', () => {
      if (audio && isListening.value) currentTime.value = audio.currentTime || 0
    })
    audio.addEventListener('waiting', () => { audioLoading.value = true })
    audio.addEventListener('playing', () => {
      audioLoading.value = false
      autoplayBlocked.value = false
      updateMediaSession()
    })
    audio.addEventListener('pause', updateMediaSession)
    audio.addEventListener('error', () => {
      audioLoading.value = false
      audioFailed.value = true
    })
    document.body.append(audio)
    return audio
  }

  function applyAudioPlayback(next: PlaybackState, force = false): void {
    const player = ensureAudio()
    const track = next.currentTrack
    const target = effectivePosition(next)
    currentTime.value = target
    if (!player || !subsonic) return
    if (!track) {
      player.pause()
      player.removeAttribute('src')
      player.load()
      loadedTrackID = ''
      lyrics.value = []
      clearPreload()
      return
    }
    const changed = loadedTrackID !== track.id
    if (changed) {
      loadedTrackID = track.id
      audioDuration.value = 0
      lyrics.value = []
      pendingSeek = target
      audioLoading.value = true
      audioFailed.value = false
      player.src = subsonic.streamURL(track.id)
      player.load()
      void loadLyrics(track)
      preloadNext(next)
    } else if (force || needsHardSeek(player.currentTime, target)) {
      seekAudio(target)
    }
    if (next.status === 'playing' && isListening.value) void safePlay()
    else player.pause()
    updateMediaSession()
  }

  function seekAudio(position: number): void {
    if (!audio) return
    const bounded = clamp(position, 0, duration.value || position)
    try {
      audio.currentTime = bounded
      currentTime.value = bounded
      pendingSeek = null
    } catch {
      pendingSeek = bounded
    }
  }

  async function loadLyrics(track: NonNullable<PlaybackState['currentTrack']>): Promise<void> {
    if (!subsonic) return
    const lines = await subsonic.lyrics({
      id: track.id,
      title: track.title,
      artist: track.artist,
      album: track.album,
      albumId: track.albumID,
      coverArt: track.coverArtID,
      duration: track.durationSeconds,
    })
    if (currentTrack.value?.id === track.id) lyrics.value = lines
  }

  function preloadNext(state: PlaybackState): void {
    clearPreload()
    if (!subsonic || !room.value?.preloadNextTrack || !state.nextTrack) return
    preloadAudio = new Audio()
    preloadAudio.preload = 'metadata'
    preloadAudio.crossOrigin = 'use-credentials'
    preloadAudio.src = subsonic.streamURL(state.nextTrack.id)
    preloadAudio.load()
  }

  function clearPreload(): void {
    if (!preloadAudio) return
    preloadAudio.pause()
    preloadAudio.removeAttribute('src')
    preloadAudio.load()
    preloadAudio = null
  }

  async function safePlay(): Promise<void> {
    if (!audio || playback.value?.status !== 'playing' || !isListening.value) return
    try {
      if (eqEnabled.value && eqAvailable) ensureEqGraph()
      if (audioContext?.state === 'suspended') await audioContext.resume()
      await audio.play()
      autoplayBlocked.value = false
      audioFailed.value = false
    } catch {
      autoplayBlocked.value = true
    }
  }

  async function startListening(): Promise<void> {
    isListening.value = true
    autoplayBlocked.value = false
    if (playback.value) {
      seekAudio(effectivePosition(playback.value))
      await safePlay()
    }
    updateMediaSession()
  }

  function stopListening(): void {
    isListening.value = false
    autoplayBlocked.value = false
    audio?.pause()
    updateMediaSession()
  }

  async function togglePlayback(): Promise<void> {
    if (!playback.value) return
    if (!canControl.value) {
      if (isListening.value) stopListening()
      else await startListening()
      return
    }
    if (playback.value.status !== 'playing') {
      isListening.value = true
      // Keep this media call in the click gesture. The authoritative response
      // will immediately seek it to the server anchor.
      if (eqEnabled.value && eqAvailable) {
        try { ensureEqGraph(); void audioContext?.resume().catch(() => {}) }
        catch { eqEnabled.value = false; applyEqGains(); failTransient({ code: 'eq_unavailable' }) }
      }
      if (audio?.src) void audio.play().catch(() => {})
    }
    await command(playback.value.status === 'playing' ? 'pause' : 'play')
  }

  async function command(name: PlaybackCommand, position?: number): Promise<void> {
    if (!gateway || !playback.value) return
    busy.value = true
    try {
      const next = await gateway.playback(roomID.value, name, playback.value.revision, position)
      applyPlaybackEvent(next)
    } catch (cause) {
      if (cause instanceof GatewayError && cause.code === 'revision_conflict') {
        await refreshSnapshot(true).catch(failTransient)
      } else {
        failTransient(cause)
      }
    } finally {
      busy.value = false
    }
  }

  async function seek(position: number): Promise<void> {
    if (!canControl.value) return
    await command('seek', clamp(position, 0, duration.value))
  }

  async function next(): Promise<void> {
    if (canSkip.value) await command('next')
  }

  async function removeQueue(entry: QueueEntry): Promise<void> {
    if (!gateway || !session.value) return
    const own = entry.contributorUsername.toLowerCase() === session.value.user.username.toLowerCase()
    if (!canManage.value && !own) return
    try {
      await gateway.removeQueue(roomID.value, entry.queueID)
      if (snapshot.value) snapshot.value = { ...snapshot.value, queue: queue.value.filter((item) => item.queueID !== entry.queueID) }
    } catch (cause) {
      failTransient(cause)
    }
  }

  async function moveQueue(index: number, direction: -1 | 1): Promise<void> {
    await reorderQueue(index, index + direction)
  }

  async function reorderQueue(from: number, to: number): Promise<void> {
    if (!gateway || !snapshot.value || !canReorderQueue.value || from === to || from < 0 || to < 0 || from >= queue.value.length || to >= queue.value.length) return
    const nextQueue = [...queue.value]
    nextQueue.splice(to, 0, nextQueue.splice(from, 1)[0])
    try {
      const result = await gateway.reorderQueue(roomID.value, nextQueue)
      snapshot.value = { ...snapshot.value, queue: result.queue }
    } catch (cause) { failTransient(cause) }
  }

  function toSong(track: TrackRef): SubsonicSong {
    return { id: track.id, title: track.title, artist: track.artist, album: track.album, albumId: track.albumID,
      artistId: track.artistID, coverArt: track.coverArtID, duration: track.durationSeconds, musicFolderId: track.musicFolderID }
  }

  async function navigateArtist(name: string, id?: string): Promise<void> {
    if (!subsonic) return
    activeTab.value = 'catalog'
    immersive.value = false
    infoOpen.value = false
    if (id) { await openArtist({ id, name }); return }
    catalogQuery.value = name
    await searchCatalog()
    catalogSection.value = 'artists'
    const exact = catalog.value.artists.find(item => item.name.toLocaleLowerCase() === name.toLocaleLowerCase())
    if (exact) await openArtist(exact)
  }

  async function navigateAlbum(name: string, id?: string): Promise<void> {
    activeTab.value = 'catalog'
    immersive.value = false
    infoOpen.value = false
    if (id) { await openAlbum({ id, name }); return }
    catalogQuery.value = name
    await searchCatalog()
    catalogSection.value = 'albums'
  }

  async function openReference(reference: ChatReference): Promise<void> {
    if (reference.kind === 'artist') await navigateArtist(reference.title, reference.id)
    else if (reference.kind === 'album') await navigateAlbum(reference.title, reference.id)
    else {
      try {
        const song = referencedSongs.value[reference.id] || await subsonic?.song(reference.id)
        if (song) { referencedSongs.value[song.id] = song; await navigateAlbum(song.album || reference.title, song.albumId) }
      } catch (cause) { failTransient(cause) }
    }
  }

  async function queueReference(reference: ChatReference): Promise<void> {
    if (!subsonic) return
    try {
      if (reference.kind === 'track') await addSong(referencedSongs.value[reference.id] || await subsonic.song(reference.id))
      else if (reference.kind === 'album') await addAlbum({ id: reference.id, name: reference.title })
      else await openReference(reference)
    } catch (cause) { failTransient(cause) }
  }

  async function searchResources(query: string): Promise<CatalogResults> {
    return subsonic && query.trim() && folderID.value ? subsonic.search(query.trim(), folderID.value) : emptyCatalog()
  }

  function trackReference(song: SubsonicSong): ChatReference {
    referencedSongs.value[song.id] = song
    return { kind: 'track', id: song.id, title: song.title, artist: song.artist }
  }

  async function refreshRandom(): Promise<void> {
    if (!subsonic || !folderID.value) return
    catalogLoading.value = true
    catalogFailure.value = null
    try {
      catalog.value = { ...catalog.value, songs: await subsonic.randomSongs(folderID.value) }
      catalogQuery.value = ''
      selectedArtist.value = null
      returnToCatalog()
      catalogSection.value = 'songs'
    } catch (cause) { catalogFailure.value = cause }
    finally { catalogLoading.value = false }
  }

  async function randomFillQueue(count = 5): Promise<void> {
    if (!subsonic || !folderID.value || busy.value || !queueRemaining.value) return
    const size = Number.isFinite(count) ? Math.max(0, Math.min(5, Math.floor(count), queueRemaining.value)) : 0
    if (!size) return
    try {
      const songs = await subsonic.randomSongs(folderID.value, size)
      for (const song of songs.slice(0, size)) {
        if (!queueRemaining.value) break
        await addSong(song)
      }
    } catch (cause) { failTransient(cause) }
  }

  async function playNext(song: SubsonicSong): Promise<void> {
    if (canControl.value) await addSong(song, true)
  }

  async function previous(): Promise<void> { if (canControl.value) await command('previous') }

  async function topQueue(entry: QueueEntry): Promise<void> {
    if (!gateway || !snapshot.value || !canReorderQueue.value) return
    try {
      const result = await gateway.topQueue(roomID.value, entry.queueID)
      snapshot.value = { ...snapshot.value, queue: result.queue }
    } catch (cause) { failTransient(cause) }
  }

  async function shuffleQueue(): Promise<void> {
    if (!gateway || !snapshot.value || !canManage.value) return
    try {
      const result = await gateway.shuffleQueue(roomID.value)
      snapshot.value = { ...snapshot.value, queue: result.queue }
    } catch (cause) { failTransient(cause) }
  }

  async function requeueHistory(entry: HistoryEntry): Promise<void> {
    if (!gateway || !snapshot.value || !queueRemaining.value || busy.value) return
    busy.value = true
    try {
      const result = await gateway.requeueHistory(roomID.value, entry.historyID)
      snapshot.value = { ...snapshot.value, queue: result.queue }
      if (result.playback) applyPlaybackEvent(result.playback)
      showNotice('queued', { title: entry.track.title })
    } catch (cause) { failTransient(cause) }
    finally { busy.value = false }
  }

  function mergeHistory(first: HistoryEntry[], second: HistoryEntry[]): HistoryEntry[] {
    return [...new Map([...second, ...first].map(item => [item.historyID, item])).values()]
      .sort((a, b) => Date.parse(b.startedAt) - Date.parse(a.startedAt))
  }

  async function loadOlderHistory(): Promise<void> {
    if (!gateway || !snapshot.value || historyLoading.value || !hasMoreHistory.value) return
    historyLoading.value = true
    try {
      const result = await gateway.history(roomID.value, history.value.length)
      snapshot.value = { ...snapshot.value, history: mergeHistory(history.value, result.items || []) }
      hasMoreHistory.value = result.items?.length === 100
    } catch (cause) { failTransient(cause) }
    finally { historyLoading.value = false }
  }

  async function synchronizePlayback(): Promise<void> {
    try {
      await refreshSnapshot(true)
      if (playback.value) seekAudio(effectivePosition(playback.value))
      showNotice('synced')
    } catch (cause) { failTransient(cause) }
  }

  async function updateRoomSettings(patch: Partial<RoomSettings>): Promise<boolean> {
    if (!gateway || !snapshot.value || !canManage.value) return false
    try {
      const result = await gateway.patchRoom(roomID.value, patch)
      snapshot.value = { ...snapshot.value, room: result }
      await refreshSnapshot()
      showNotice('saved')
      return true
    } catch (cause) { failTransient(cause); return false }
  }

  function upsertMessage(message: ChatMessage): void {
    if (!message.messageID) return
    const index = messages.value.findIndex(item => item.messageID === message.messageID)
    if (index >= 0) messages.value[index] = { ...messages.value[index], ...message }
    else messages.value.push(message)
    messages.value.sort((a, b) => Date.parse(a.createdAt) - Date.parse(b.createdAt) || a.messageID.localeCompare(b.messageID))
    void loadSenderProfile(message.senderUsername)
    for (const reference of message.references || []) {
      if (reference.kind === 'track' && !referencedSongs.value[reference.id] && subsonic) {
        void subsonic.song(reference.id).then(song => { referencedSongs.value[song.id] = song }).catch(() => {})
      }
    }
  }

  async function loadMessages(older = false): Promise<void> {
    if (!gateway || messagesLoading.value) return
    messagesLoading.value = true
    try {
      const result = await gateway.chatMessages(roomID.value, 50, older ? messages.value[0]?.messageID : '')
      for (const message of result.messages || []) upsertMessage(message)
      hasMoreMessages.value = result.hasMore
      if (!stickerPacks.value.length) stickerPacks.value = (await gateway.stickers(roomID.value)).packs || []
    } catch (cause) { failTransient(cause) }
    finally { messagesLoading.value = false }
  }

  async function loadOlderMessages(): Promise<void> { if (hasMoreMessages.value) await loadMessages(true) }

  async function sendChat(payload: ChatPayload): Promise<boolean> {
    if (!gateway) return false
    try {
      const message = await gateway.sendMessage(roomID.value, payload)
      upsertMessage(message)
      draftReferences.value = []
      draftMentions.value = []
      return true
    } catch (cause) { failTransient(cause); return false }
  }

  async function recallChat(message: ChatMessage): Promise<void> {
    if (!gateway || !sameUser(message.senderUsername) || Date.now() - Date.parse(message.createdAt) > 300_000) return
    try { upsertMessage(await gateway.recallMessage(roomID.value, message.messageID)) }
    catch (cause) { failTransient(cause) }
  }

  async function deleteChat(message: ChatMessage): Promise<void> {
    if (!gateway || !canManage.value) return
    try {
      await gateway.deleteMessage(roomID.value, message.messageID)
      upsertMessage({ ...message, deletedAt: new Date().toISOString(), body: '', references: [], stickerID: '' })
    } catch (cause) { failTransient(cause) }
  }

  function suggestTrackToChat(song: SubsonicSong): void {
    if (!draftReferences.value.some(item => item.kind === 'track' && item.id === song.id)) {
      draftReferences.value = [...draftReferences.value, trackReference(song)].slice(0, 5)
    }
    activeTab.value = 'chat'
    immersive.value = false
  }

  function suggestUserToChat(member: Pick<Member, 'username'>): void {
    draftMentions.value = [...new Set([...draftMentions.value, member.username])].slice(0, 10)
    activeTab.value = 'chat'
    immersive.value = false
    profileUsername.value = ''
    infoOpen.value = false
  }

  async function loadWishes(status: 'pending' | 'fulfilled' | '' = ''): Promise<void> {
    if (!gateway) return
    wishes.value = (await gateway.wishes(roomID.value, status)).wishes || []
  }
  function upsertWish(wish: RoomWish): void {
    wishes.value = [wish, ...wishes.value.filter(item => item.wishID !== wish.wishID)]
  }
  async function createWish(payload: WishPayload): Promise<boolean> {
    if (!gateway) return false
    try { upsertWish(await gateway.createWish(roomID.value, payload)); return true }
    catch (cause) { failTransient(cause); return false }
  }
  async function fulfillWish(wishID: string, payload: WishFulfillment): Promise<boolean> {
    if (!gateway || !canFulfillWish.value) return false
    try { upsertWish(await gateway.fulfillWish(roomID.value, wishID, payload)); return true }
    catch (cause) { failTransient(cause); return false }
  }
  async function deleteWish(wishID: string): Promise<void> {
    if (!gateway) return
    try { await gateway.deleteWish(roomID.value, wishID); wishes.value = wishes.value.filter(item => item.wishID !== wishID) }
    catch (cause) { failTransient(cause) }
  }

  function wishResource(reference: ChatReference): WishResource {
    const url = new URL(window.location.href)
    url.search = ''
    url.hash = new URLSearchParams({ resource: reference.kind, id: reference.id, title: reference.title }).toString()
    return { title: reference.title, url: url.toString() }
  }
  function wishReference(resource: WishResource): ChatReference | null {
    const reference = resourceFromURL(resource.url)
    return reference ? { ...reference, title: resource.title } : null
  }
  async function openWishResource(resource: WishResource): Promise<void> {
    const reference = wishReference(resource)
    if (reference) await openReference(reference)
    else {
      const url = safeResourceURL(resource.url)
      if (url) window.open(url, '_blank', 'noopener,noreferrer')
    }
  }
  async function queueWishResource(resource: WishResource): Promise<void> {
    const reference = wishReference(resource)
    if (reference) await queueReference(reference)
  }

  async function roomInformation(period: Period): Promise<[RoomStatistics, RoomRankings] | undefined> {
    if (gateway) return Promise.all([gateway.statistics(roomID.value, period), gateway.rankings(roomID.value, period)])
  }
  async function memberProfile(username: string): Promise<MemberProfile | undefined> {
    if (!gateway) return
    const profile = sameUser(username) ? await gateway.achievements(roomID.value) : await gateway.memberProfile(roomID.value, username)
    memberProfiles.value[username.toLowerCase()] = profile
    return profile
  }
  async function loadSenderProfile(username: string): Promise<void> {
    const key = username.toLowerCase()
    if (memberProfiles.value[key] || profileRequests.has(key)) return
    profileRequests.add(key)
    try { await memberProfile(username) } catch { /* Chat remains readable when a sender profile is unavailable. */ }
    finally { profileRequests.delete(key) }
  }
  async function selectBadge(badgeID: string): Promise<void> {
    if (!gateway) return
    const profile = await gateway.selectBadge(roomID.value, badgeID)
    memberProfiles.value[profile.username.toLowerCase()] = profile
  }
  async function removeMember(member: Member): Promise<void> {
    if (!gateway || !canManage.value || member.role === 'owner') return
    try { await gateway.removeMember(roomID.value, member.username); await refreshSnapshot() }
    catch (cause) { failTransient(cause) }
  }
  async function leaveRoom(): Promise<void> {
    if (!gateway) return
    try {
      await gateway.leave(roomID.value)
      disconnect()
      window.location.assign('/app/')
    } catch (cause) { failTransient(cause) }
  }

  function persistEq(): void {
    try { localStorage.setItem('nmr:eq:v1', JSON.stringify({ enabled: eqEnabled.value, preset: eqPreset.value, gains: eqBands.value.map(band => band.gain) })) }
    catch { /* Storage may be disabled; keep the current sound. */ }
  }
  function ensureEqGraph(): boolean {
    if (!eqAvailable || !audio) return false
    if (eqSource) return true
    const AudioContextClass = window.AudioContext || (window as Window & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
    if (!AudioContextClass) return false
    audioContext ||= new AudioContextClass()
    eqSource = audioContext.createMediaElementSource(audio)
    eqFilters = eqFrequencies.map((frequency, index) => {
      const filter = audioContext!.createBiquadFilter()
      filter.type = index === 0 ? 'lowshelf' : index === 4 ? 'highshelf' : 'peaking'
      filter.frequency.value = frequency
      filter.Q.value = 1
      return filter
    })
    let previous: AudioNode = eqSource
    for (const filter of eqFilters) { previous.connect(filter); previous = filter }
    previous.connect(audioContext.destination)
    applyEqGains()
    return true
  }
  function applyEqGains(): void {
    if (!audioContext) return
    eqFilters.forEach((filter, index) => {
      filter.gain.setTargetAtTime(eqEnabled.value ? eqBands.value[index].gain : 0, audioContext!.currentTime, 0.025)
    })
  }
  async function setEqEnabled(enabled: boolean): Promise<void> {
    if (enabled && !eqAvailable) return
    try {
      if (enabled) { ensureAudio(); if (!ensureEqGraph()) return }
      eqEnabled.value = enabled
      applyEqGains()
      if (audioContext?.state === 'suspended') await audioContext.resume()
      persistEq()
    } catch { eqEnabled.value = false; applyEqGains(); failTransient({ code: 'eq_unavailable' }) }
  }
  function setEqBandGain(index: number, gain: number): void {
    if (!eqBands.value[index] || !Number.isFinite(gain)) return
    eqBands.value[index].gain = clamp(gain, -12, 12)
    eqPreset.value = 'custom'
    applyEqGains()
    persistEq()
  }
  async function applyEqPreset(preset: EqPreset): Promise<void> {
    eqPreset.value = preset
    if (preset !== 'custom') eqBands.value.forEach((band, index) => { band.gain = eqPresets[preset][index] })
    applyEqGains()
    persistEq()
    await setEqEnabled(true)
  }

  function setVolume(next: number): void {
    volume.value = clamp(next, 0, 1)
    if (audio) audio.volume = volume.value
  }

  function startClocks(): void {
    if (progressTimer === null) {
      progressTimer = window.setInterval(() => {
        if (audio && isListening.value && !audio.paused) currentTime.value = audio.currentTime || 0
        else if (playback.value?.status === 'playing') currentTime.value = effectivePosition(playback.value)
      }, 250)
    }
    if (driftTimer === null) {
      driftTimer = window.setInterval(() => {
        if (!audio || !isListening.value || playback.value?.status !== 'playing') return
        const expected = effectivePosition(playback.value)
        if (needsHardSeek(audio.currentTime, expected)) seekAudio(expected)
      }, 10_000)
    }
  }

  function updateMediaSession(): void {
    if (!('mediaSession' in navigator) || !currentTrack.value) return
    try {
      navigator.mediaSession.metadata = new MediaMetadata({
        title: currentTrack.value.title,
        artist: currentTrack.value.artist,
        album: currentTrack.value.album,
        artwork: subsonic && currentTrack.value.coverArtID
          ? [{ src: subsonic.coverURL(currentTrack.value.coverArtID, 512), sizes: '512x512' }]
          : [],
      })
      navigator.mediaSession.playbackState = audio && !audio.paused ? 'playing' : 'paused'
      navigator.mediaSession.setActionHandler('play', () => { void startListening() })
      navigator.mediaSession.setActionHandler('pause', stopListening)
      navigator.mediaSession.setActionHandler('nexttrack', canSkip.value ? () => { void next() } : null)
      navigator.mediaSession.setActionHandler('previoustrack', canControl.value ? () => { void previous() } : null)
      navigator.mediaSession.setActionHandler('seekto', canControl.value ? (details) => {
        if (details.seekTime !== undefined) void seek(details.seekTime)
      } : null)
    } catch {
      // Media Session is best-effort across browsers.
    }
  }

  function coverURL(id?: string, size = 720): string {
    return subsonic?.coverURL(id, size) || ''
  }

  function disconnect(): void {
    manualDisconnect = true
    connectionGeneration += 1
    connected.value = false
    reconnecting.value = false
    if (reconnectTimer !== null) window.clearTimeout(reconnectTimer)
    if (sessionTimer !== null) window.clearTimeout(sessionTimer)
    if (progressTimer !== null) window.clearInterval(progressTimer)
    if (driftTimer !== null) window.clearInterval(driftTimer)
    reconnectTimer = sessionTimer = progressTimer = driftTimer = null
    socket?.close()
    socket = null
    audio?.pause()
    clearPreload()
  }

  function handleVisibility(): void {
    if (document.visibilityState !== 'visible' || phase.value !== 'ready') return
    void refreshSnapshot(true).catch(failTransient)
    if (!socket || socket.readyState > WebSocket.OPEN) connectRealtime()
  }
  document.addEventListener('visibilitychange', handleVisibility)
  onScopeDispose(() => {
    document.removeEventListener('visibilitychange', handleVisibility)
    disconnect()
    if (noticeTimer !== null) window.clearTimeout(noticeTimer)
    audio?.remove()
    eqSource?.disconnect()
    eqFilters.forEach(filter => filter.disconnect())
    void audioContext?.close()
  })

  return {
    phase, roomID, invitation, session, snapshot, activeTab, connected, reconnecting,
    error, errorCode, notice, busy, currentTime, duration, volume, isListening,
    autoplayBlocked, audioLoading, audioError, lyrics, activeLyricIndex, folders,
    folderID, catalog, catalogMode, catalogSection, catalogTitle, catalogQuery, catalogLoading,
    catalogError, selectedAlbum, selectedArtist, artistAlbums, favorites, playlists, selectedPlaylist,
    favoritesLoading, room, self, playback, queue, history, members, currentTrack,
    canManage, onlineCount, queueRemaining, deepLink, configure, start, login,
    refreshSnapshot, browseCatalog, searchCatalog, selectCatalogSection, returnToCatalog,
    openAlbum, openArtist, loadFavorites,
    openPlaylist, toggleStar, addSong, addAlbum, addPlaylist, startListening,
    stopListening, togglePlayback, seek, next, removeQueue, moveQueue, setVolume,
    coverURL, showNotice, disconnect, failTransient, sameUser, toSong, isStarred,
    permissions, canControl, canSkip, canReorderQueue, canFulfillWish, updateNotice, immersive, profileUsername, infoOpen, adminURL,
    navigateArtist, navigateAlbum, openReference, queueReference, searchResources, trackReference, refreshRandom,
    previous, playNext, topQueue, shuffleQueue, randomFillQueue, requeueHistory, loadOlderHistory, hasMoreHistory, historyLoading,
    reorderQueue, synchronizePlayback, updateRoomSettings,
    memberProfiles, referencedSongs, messages, hasMoreMessages, messagesLoading, unreadChatCount, chatVisible, stickerPacks, draftReferences, draftMentions,
    loadMessages, loadOlderMessages, sendChat, recallChat, deleteChat, suggestTrackToChat, suggestUserToChat,
    wishResource, wishReference, openWishResource, queueWishResource, wishes, loadWishes, createWish, fulfillWish, deleteWish, roomInformation, memberProfile, selectBadge, removeMember, leaveRoom,
    eqEnabled, eqPreset, eqBands, eqAvailable, nativeBackgroundPlayback, setEqEnabled, setEqBandGain, applyEqPreset,
    handleEvent,
  }
})
