import { computed, onScopeDispose, ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'

import { GatewayClient, GatewayError, navidromeLogin } from '../lib/gateway'
import { clearInvitation, readProof, storeNavidromeLogin } from '../lib/location'
import { SubsonicClient } from '../lib/subsonic'
import { clamp, effectivePosition, needsHardSeek } from '../lib/sync'
import type {
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
  const notice = ref('')
  const busy = ref(false)

  const currentTime = ref(0)
  const audioDuration = ref(0)
  const volume = ref(0.8)
  const isListening = ref(false)
  const autoplayBlocked = ref(false)
  const audioLoading = ref(false)
  const audioError = ref('')
  const lyrics = ref<LyricLine[]>([])

  const folderID = ref(0)
  const folders = ref<Array<{ id: number; name: string }>>([])
  const catalog = shallowRef<CatalogResults>(emptyCatalog())
  const catalogMode = ref<'browse' | 'search' | 'album' | 'artist'>('browse')
  const catalogSection = ref<CatalogSection>('songs')
  const catalogTitle = ref('浏览曲库')
  const catalogQuery = ref('')
  const catalogLoading = ref(false)
  const catalogError = ref('')
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
  const canManage = computed(() => Boolean(
    session.value?.user.adminRole || self.value?.role === 'owner' ||
    (room.value && session.value && room.value.ownerUsername.toLowerCase() === session.value.user.username.toLowerCase()),
  ))
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
      fail(new Error('分享链接中的房间 ID 无效'), 'room_id_invalid')
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
      connectRealtime()
      startClocks()
    } catch (cause) {
      const code = cause instanceof GatewayError ? cause.code : cause instanceof Error && 'code' in cause ? String(cause.code) : 'join_failed'
      fail(cause, code, proof.value ? 'error' : 'login')
    }
  }

  function fail(cause: unknown, code: string, nextPhase: Phase = 'error'): void {
    error.value = cause instanceof Error ? cause.message : String(cause || '未知错误')
    errorCode.value = code
    phase.value = nextPhase
  }

  function showNotice(message: string): void {
    notice.value = message
    if (noticeTimer !== null) window.clearTimeout(noticeTimer)
    noticeTimer = window.setTimeout(() => { notice.value = '' }, 2800)
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
    snapshot.value = {
      ...next,
      members: Array.isArray(next.members) ? next.members : [],
      queue: Array.isArray(next.queue) ? next.queue : [],
      history: Array.isArray(next.history) ? next.history : [],
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
    catalogError.value = ''
    try {
      catalog.value = await subsonic.browse(folderID.value)
      catalogMode.value = 'browse'
      catalogSection.value = 'songs'
      catalogTitle.value = '浏览曲库'
      selectedAlbum.value = null
      selectedArtist.value = null
      artistAlbums.value = []
    } catch (cause) {
      catalogError.value = cause instanceof Error ? cause.message : '曲库读取失败'
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
    catalogError.value = ''
    try {
      const results = await subsonic.search(query, folderID.value)
      catalog.value = results
      catalogMode.value = 'search'
      catalogTitle.value = `“${query}”的搜索结果`
      selectedAlbum.value = null
      selectedArtist.value = null
      artistAlbums.value = []
      const currentResults = results[catalogSection.value]
      if (!currentResults.length) {
        catalogSection.value = results.songs.length ? 'songs' : results.albums.length ? 'albums' : 'artists'
      }
    } catch (cause) {
      catalogError.value = cause instanceof Error ? cause.message : '搜索失败'
    } finally {
      catalogLoading.value = false
    }
  }

  async function openAlbum(album: SubsonicAlbum): Promise<void> {
    if (!subsonic) return
    catalogLoading.value = true
    try {
      selectedAlbum.value = await subsonic.album(album.id)
      selectedArtist.value = null
      artistAlbums.value = []
      catalogMode.value = 'album'
      catalogSection.value = 'songs'
      catalogTitle.value = selectedAlbum.value.name
    } catch (cause) {
      catalogError.value = cause instanceof Error ? cause.message : '专辑读取失败'
    } finally {
      catalogLoading.value = false
    }
  }

  async function openArtist(artist: SubsonicArtist): Promise<void> {
    if (!subsonic) return
    catalogLoading.value = true
    try {
      const albums = await subsonic.artistAlbums(artist.id)
      selectedArtist.value = artist
      selectedAlbum.value = null
      artistAlbums.value = albums
      catalogMode.value = 'artist'
      catalogSection.value = 'albums'
      catalogTitle.value = artist.name
    } catch (cause) {
      catalogError.value = cause instanceof Error ? cause.message : '歌手读取失败'
    } finally {
      catalogLoading.value = false
    }
  }

  function selectCatalogSection(section: CatalogSection): void {
    catalogSection.value = section
    if (catalogMode.value === 'album' || catalogMode.value === 'artist') returnToCatalog()
  }

  function returnToCatalog(): void {
    selectedAlbum.value = null
    selectedArtist.value = null
    artistAlbums.value = []
    const query = catalogQuery.value.trim()
    catalogMode.value = query ? 'search' : 'browse'
    catalogTitle.value = query ? `“${query}”的搜索结果` : '浏览曲库'
  }

  async function loadFavorites(): Promise<void> {
    if (!subsonic) return
    favoritesLoading.value = true
    try {
      const [starred, playlistItems] = await Promise.all([subsonic.starred(), subsonic.playlists()])
      favorites.value = starred
      playlists.value = playlistItems
    } catch {
      favorites.value = emptyCatalog()
      playlists.value = []
    } finally {
      favoritesLoading.value = false
    }
  }

  async function openPlaylist(playlist: SubsonicPlaylist): Promise<void> {
    if (!subsonic) return
    favoritesLoading.value = true
    try {
      selectedPlaylist.value = await subsonic.playlist(playlist.id)
    } finally {
      favoritesLoading.value = false
    }
  }

  async function toggleStar(song: SubsonicSong): Promise<void> {
    if (!subsonic) return
    const starred = !song.starred
    await subsonic.setStarred('song', song.id, starred)
    song.starred = starred ? new Date().toISOString() : undefined
    favorites.value = {
      ...favorites.value,
      songs: starred
        ? [song, ...favorites.value.songs.filter((item) => item.id !== song.id)]
        : favorites.value.songs.filter((item) => item.id !== song.id),
    }
    showNotice(starred ? '已收藏到 Navidrome' : '已取消收藏')
  }

  async function addSong(song: SubsonicSong): Promise<void> {
    if (!gateway || !subsonic || !folderID.value) return
    busy.value = true
    try {
      const result = await gateway.addTrack(roomID.value, subsonic.toTrack(song, folderID.value))
      if (!snapshot.value) return
      snapshot.value = { ...snapshot.value, queue: result.queue || [] }
      if (result.playback) applyPlaybackEvent(result.playback)
      showNotice(`已点播 ${song.title}`)
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
      if (added > 0) showNotice(`已点播专辑 ${full.name}（${added} 首）`)
    } catch (cause) {
      failTransient(cause)
    } finally {
      busy.value = false
    }
  }

  async function addPlaylist(playlist: SubsonicPlaylist): Promise<void> {
    if (!subsonic) return
    const full = playlist.entry ? playlist : await subsonic.playlist(playlist.id)
    for (const song of full.entry || []) {
      if (queueRemaining.value <= 0) break
      await addSong(song)
    }
  }

  function failTransient(cause: unknown): void {
    const message = cause instanceof Error ? cause.message : String(cause || '操作失败')
    error.value = message
    window.setTimeout(() => {
      if (error.value === message && phase.value === 'ready') error.value = ''
    }, 4500)
  }

  function connectRealtime(replace = false): void {
    if (!gateway || phase.value !== 'ready') return
    if (socket && !replace && socket.readyState <= WebSocket.OPEN) return
    manualDisconnect = false
    socket?.close()
    socket = null
    reconnecting.value = reconnectAttempt > 0
    void gateway.websocketTicket(roomID.value).then((ticket) => {
      if (manualDisconnect || phase.value !== 'ready') return
      socket = new WebSocket(ticket.webSocketURL)
      socket.addEventListener('open', () => {
        connected.value = true
        reconnecting.value = false
        reconnectAttempt = 0
      })
      socket.addEventListener('message', (event) => {
        try {
          handleEvent(JSON.parse(String(event.data)) as GatewayEvent)
        } catch {
          failTransient(new Error('收到无法识别的房间消息'))
        }
      })
      socket.addEventListener('close', () => {
        connected.value = false
        socket = null
        scheduleReconnect()
      })
      socket.addEventListener('error', () => {
        socket?.close()
      })
    }).catch((cause) => {
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
    if (!snapshot.value) return
    if (event.type === 'playback') {
      const next = event.payload as PlaybackState
      if (next.revision < snapshot.value.playback.revision) return
      applyPlaybackEvent(next)
      return
    }
    if (event.type === 'queue') {
      snapshot.value = { ...snapshot.value, queue: Array.isArray(event.payload) ? event.payload as QueueEntry[] : [] }
      return
    }
    if (event.type === 'history') {
      snapshot.value = { ...snapshot.value, history: Array.isArray(event.payload) ? event.payload as HistoryEntry[] : [] }
      return
    }
    if (event.type === 'room_updated') {
      snapshot.value = { ...snapshot.value, room: event.payload as Room }
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
      audioError.value = '当前歌曲无法从 Navidrome 加载'
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
      pendingSeek = target
      audioLoading.value = true
      audioError.value = ''
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
    lyrics.value = await subsonic.lyrics({
      id: track.id,
      title: track.title,
      artist: track.artist,
      album: track.album,
      albumId: track.albumID,
      coverArt: track.coverArtID,
      duration: track.durationSeconds,
    })
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
      await audio.play()
      autoplayBlocked.value = false
      audioError.value = ''
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
    audio?.remove()
    audio = null
    updateMediaSession()
  }

  async function togglePlayback(): Promise<void> {
    if (!playback.value) return
    if (!canManage.value) {
      if (isListening.value) stopListening()
      else await startListening()
      return
    }
    if (playback.value.status !== 'playing') {
      isListening.value = true
      // Keep this media call in the click gesture. The authoritative response
      // will immediately seek it to the server anchor.
      if (audio?.src) void audio.play().catch(() => {})
    }
    await command(playback.value.status === 'playing' ? 'pause' : 'play')
  }

  async function command(name: string, position?: number): Promise<void> {
    if (!gateway || !playback.value) return
    busy.value = true
    try {
      const next = await gateway.playback(roomID.value, name, playback.value.revision, position)
      applyPlaybackEvent(next)
    } catch (cause) {
      if (cause instanceof GatewayError && cause.code === 'revision_conflict') {
        await refreshSnapshot(true)
      } else {
        failTransient(cause)
      }
    } finally {
      busy.value = false
    }
  }

  async function seek(position: number): Promise<void> {
    if (!canManage.value) return
    await command('seek', clamp(position, 0, duration.value))
  }

  async function next(): Promise<void> {
    if (canManage.value) await command('next')
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
    if (!gateway || !snapshot.value || !canManage.value) return
    const target = index + direction
    if (target < 0 || target >= queue.value.length) return
    const nextQueue = [...queue.value]
    ;[nextQueue[index], nextQueue[target]] = [nextQueue[target], nextQueue[index]]
    try {
      const result = await gateway.reorderQueue(roomID.value, nextQueue)
      snapshot.value = { ...snapshot.value, queue: result.queue }
    } catch (cause) {
      failTransient(cause)
    }
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
      navigator.mediaSession.setActionHandler('nexttrack', canManage.value ? () => { void next() } : null)
      navigator.mediaSession.setActionHandler('seekto', canManage.value ? (details) => {
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
    coverURL, showNotice, disconnect,
  }
})
