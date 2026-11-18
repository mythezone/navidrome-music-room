import { createPinia, disposePinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { GatewayClient, GatewayError } from '../lib/gateway'
import { SubsonicClient } from '../lib/subsonic'
import { eqPresets } from '../lib/eq'
import { setLocale } from '../lib/i18n'
import { chatMessage, gatewaySession, roomSnapshot } from '../test/fixtures'
import { useRoomStore } from './room'

let pinia: ReturnType<typeof createPinia>
beforeEach(() => {
  localStorage.clear()
  setLocale('en')
  vi.useFakeTimers()
  vi.spyOn(HTMLMediaElement.prototype, 'pause').mockImplementation(() => {})
  vi.spyOn(HTMLMediaElement.prototype, 'load').mockImplementation(() => {})
  vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue()
  pinia = createPinia(); setActivePinia(pinia)
})
afterEach(() => { disposePinia(pinia); vi.useRealTimers(); vi.restoreAllMocks(); vi.unstubAllGlobals(); document.querySelectorAll('audio').forEach(audio => audio.remove()) })
function room() {
  const store = useRoomStore()
  store.configure({ roomID: '0123456789abcdef0123456789abcdef', gatewayPrefix: '/music-room', invitation: '' })
  store.session = gatewaySession()
  store.snapshot = roomSnapshot()
  return store
}
describe('room playback permissions', () => {
  it('permits room-wide skipping and own requests only when disabled room-wide', () => {
    const store = room()
    expect(store.canSkip).toBe(false)
    store.snapshot = { ...store.snapshot!, room: { ...store.room!, allowMemberSkip: true } }
    expect(store.canSkip).toBe(true)
    store.snapshot = { ...store.snapshot!, room: { ...store.room!, allowMemberSkip: false }, playback: { ...store.playback!, contributorUsername: 'LISTENER' } }
    expect(store.canSkip).toBe(true)
    store.snapshot = { ...store.snapshot!, playback: { ...store.playback!, currentTrack: undefined } }
    expect(store.canSkip).toBe(false)
  })
  it('allows managers and honors explicit server permissions including false', () => {
    const store = room()
    store.session = { ...store.session!, user: { ...store.session!.user, adminRole: true } }
    expect(store.canManage).toBe(true)
    expect(store.canSkip).toBe(true)
    store.snapshot = { ...store.snapshot!, permissions: { canManage: false, canControl: false, canSkip: false, canReorderQueue: false, canFulfillWish: false } }
    expect(store.permissions).toEqual({ canManage: false, canControl: false, canSkip: false, canReorderQueue: false, canFulfillWish: false })
    store.snapshot = { ...store.snapshot!, permissions: { ...store.snapshot!.permissions!, canSkip: true } }
    expect(store.canSkip).toBe(true)
  })
  it('gates next separately from manager control and uses the latest revision', async () => {
    const store = room()
    const playback = vi.spyOn(GatewayClient.prototype, 'playback').mockResolvedValue({ ...store.playback!, revision: 3 })
    await store.next(); expect(playback).not.toHaveBeenCalled()
    await store.previous(); expect(playback).not.toHaveBeenCalled()
    store.snapshot = { ...store.snapshot!, room: { ...store.room!, allowMemberSkip: true } }
    await store.next()
    expect(playback).toHaveBeenCalledWith(store.roomID, 'next', 2, undefined)
    expect(store.playback?.revision).toBe(3)
  })
  it('refreshes on revision conflicts instead of retrying a stale command', async () => {
    const store = room(); store.snapshot = { ...store.snapshot!, self: { ...store.self!, role: 'owner' } }
    const fresh = roomSnapshot(); fresh.playback.revision = 9
    vi.spyOn(GatewayClient.prototype, 'playback').mockRejectedValue(new GatewayError('conflict', 409, 'revision_conflict'))
    const snapshot = vi.spyOn(GatewayClient.prototype, 'snapshot').mockResolvedValue(fresh)
    await store.next()
    expect(snapshot).toHaveBeenCalledOnce()
    expect(store.playback?.revision).toBe(9)
  })
  it('enables the OS next-track handler for a member with skip permission', async () => {
    const setActionHandler = vi.fn()
    vi.stubGlobal('MediaMetadata', class { constructor(public metadata: unknown) {} })
    vi.stubGlobal('navigator', { mediaSession: { setActionHandler }, userAgent: 'Test', maxTouchPoints: 0 })
    const store = room()
    store.snapshot = { ...store.snapshot!, room: { ...store.room!, allowMemberSkip: true } }
    await nextTick()
    expect(setActionHandler).toHaveBeenCalledWith('nexttrack', expect.any(Function))
    store.snapshot = { ...store.snapshot!, room: { ...store.room!, allowMemberSkip: false } }
    await nextTick()
    expect(setActionHandler).toHaveBeenLastCalledWith('seekto', null)
    expect(setActionHandler).toHaveBeenCalledWith('nexttrack', null)
  })
})
describe('chat state', () => {
  it('counts unseen incoming messages once, excluding self, visible chat and updates', async () => {
    const store = room()
    vi.spyOn(GatewayClient.prototype, 'memberProfile').mockRejectedValue(new Error('no profile'))
    vi.spyOn(GatewayClient.prototype, 'achievements').mockRejectedValue(new Error('no profile'))
    function event(type: 'chat_message' | 'chat_updated', message = chatMessage()) { store.handleEvent({ type, roomID: store.roomID, serverTime: '', payload: message }) }
    event('chat_message'); event('chat_message'); expect(store.unreadChatCount).toBe(1)
    event('chat_message', chatMessage({ messageID: 'own', senderUsername: 'LISTENER' })); expect(store.unreadChatCount).toBe(1)
    event('chat_updated', chatMessage({ messageID: 'unloaded', recalledAt: new Date().toISOString(), body: '' })); expect(store.messages).toHaveLength(2)
    event('chat_updated', chatMessage({ recalledAt: new Date().toISOString(), body: '' })); expect(store.unreadChatCount).toBe(1)
    store.chatVisible = true; expect(store.unreadChatCount).toBe(0)
    event('chat_message', chatMessage({ messageID: 'visible' })); expect(store.unreadChatCount).toBe(0)
    expect(store.messages).toHaveLength(3)
    await nextTick()
  })
  it('preserves draft references after a failed send and deduplicates suggestions', async () => {
    const store = room()
    store.suggestTrackToChat({ id: 'song', title: 'Song' }); store.suggestTrackToChat({ id: 'song', title: 'Song' })
    expect(store.draftReferences).toHaveLength(1)
    vi.spyOn(GatewayClient.prototype, 'sendMessage').mockRejectedValue(new GatewayError('rate', 429, 'chat_rate_limited'))
    expect(await store.sendChat({ body: '', references: store.draftReferences })).toBe(false)
    expect(store.draftReferences).toHaveLength(1)
  })
})
describe('Web Audio EQ', () => {
  it('keeps preset gains, clamps custom values and persists preferences', async () => {
    const store = room()
    for (const preset of Object.keys(eqPresets) as Array<keyof typeof eqPresets>) {
      await store.applyEqPreset(preset)
      expect(store.eqBands.map(band => band.gain)).toEqual(eqPresets[preset])
      expect(store.eqPreset).toBe(preset)
    }
    store.setEqBandGain(0, 100)
    expect(store.eqBands[0].gain).toBe(12)
    expect(store.eqPreset).toBe('custom')
    expect(JSON.parse(localStorage.getItem('nmr:eq:v1')!).gains[0]).toBe(12)
    store.setEqBandGain(0, NaN); expect(store.eqBands[0].gain).toBe(12)
  })
  it('connects exactly five filters once and applies/bypasses their gains', async () => {
    const filters: Array<{ type: string; frequency: { value: number }; Q: { value: number }; gain: { setTargetAtTime: ReturnType<typeof vi.fn> }; connect: ReturnType<typeof vi.fn>; disconnect: ReturnType<typeof vi.fn> }> = []
    const source = { connect: vi.fn(), disconnect: vi.fn() }
    const createSource = vi.fn(() => source)
    vi.stubGlobal('AudioContext', class {
      state = 'running'; currentTime = 0; destination = {}
      createMediaElementSource = createSource
      createBiquadFilter() {
        const filter = { type: '', frequency: { value: 0 }, Q: { value: 0 }, gain: { setTargetAtTime: vi.fn() }, connect: vi.fn(), disconnect: vi.fn() }
        filters.push(filter); return filter
      }
      resume = vi.fn().mockResolvedValue(undefined)
      close = vi.fn().mockResolvedValue(undefined)
    })
    const store = room()
    await store.applyEqPreset('bass')
    expect(filters.map(filter => filter.frequency.value)).toEqual([60, 230, 910, 3600, 14000])
    expect(filters.map(filter => filter.type)).toEqual(['lowshelf', 'peaking', 'peaking', 'peaking', 'highshelf'])
    filters.forEach((filter, index) => expect(filter.gain.setTargetAtTime).toHaveBeenLastCalledWith(eqPresets.bass[index], 0, .025))
    await store.setEqEnabled(false)
    filters.forEach(filter => expect(filter.gain.setTargetAtTime).toHaveBeenLastCalledWith(0, 0, .025))
    await store.setEqEnabled(true)
    expect(createSource).toHaveBeenCalledOnce()
  })
  it('bypasses a saved EQ on mobile to preserve native background audio', async () => {
    localStorage.setItem('nmr:eq:v1', JSON.stringify({ enabled: true, preset: 'bass', gains: [6, 3, 0, -1, -2] }))
    vi.spyOn(navigator, 'userAgent', 'get').mockReturnValue('Mozilla/5.0 (iPhone)')
    const store = room()
    expect(store.nativeBackgroundPlayback).toBe(true)
    expect(store.eqEnabled).toBe(false)
    await store.setEqEnabled(true)
    expect(store.eqEnabled).toBe(false)
  })
})

describe('local audio synchronization', () => {
  it('retains the media element when locally paused and seeks on manual synchronization', async () => {
    const store = room()
    const state = roomSnapshot(); state.playback.status = 'playing'; state.playback.serverTime = new Date().toISOString()
    store.handleEvent({ type: 'snapshot', payload: state, roomID: store.roomID, serverTime: '' })
    await store.startListening()
    const audio = document.querySelector('audio')!
    expect(audio).toBeTruthy()
    store.stopListening()
    expect(document.querySelector('audio')).toBe(audio)
    await store.startListening()
    expect(store.isListening).toBe(true)
    const fresh = roomSnapshot(); fresh.playback.positionSeconds = 80
    vi.spyOn(GatewayClient.prototype, 'snapshot').mockResolvedValue(fresh)
    await store.synchronizePlayback()
    expect(audio.currentTime).toBe(80)
    expect(store.currentTime).toBe(80)
  })
})

describe('realtime reconnection', () => {
  it('ignores late events from a socket that has been replaced', async () => {
    const sockets: Array<EventTarget & { readyState: number }> = []
    vi.stubGlobal('WebSocket', class extends EventTarget {
      static OPEN = 1
      readyState = 1
      constructor() { super(); sockets.push(this) }
      close() { this.readyState = 3 }
    })
    localStorage.setItem('username', 'listener')
    localStorage.setItem('subsonic-salt', 'abcdef')
    localStorage.setItem('subsonic-token', '0123456789abcdef0123456789abcdef')
    vi.spyOn(GatewayClient.prototype, 'exchange').mockResolvedValue(gatewaySession())
    vi.spyOn(GatewayClient.prototype, 'join').mockResolvedValue({ room: roomSnapshot().room })
    vi.spyOn(GatewayClient.prototype, 'snapshot').mockResolvedValue(roomSnapshot())
    vi.spyOn(GatewayClient.prototype, 'websocketTicket').mockResolvedValue({ webSocketURL: 'ws://localhost/socket', expiresAt: '' })
    vi.spyOn(SubsonicClient.prototype, 'folders').mockResolvedValue([{ id: 7, name: 'Library' }])
    vi.spyOn(SubsonicClient.prototype, 'browse').mockResolvedValue({ songs: [], albums: [], artists: [] })
    vi.spyOn(SubsonicClient.prototype, 'starred').mockResolvedValue({ songs: [], albums: [], artists: [] })
    vi.spyOn(SubsonicClient.prototype, 'playlists').mockResolvedValue([])
    vi.spyOn(SubsonicClient.prototype, 'lyrics').mockResolvedValue([])
    const store = room()
    await store.start()
    sockets[0].dispatchEvent(new Event('open'))
    expect(store.connected).toBe(true)
    sockets[0].readyState = 3
    sockets[0].dispatchEvent(new Event('close'))
    await vi.advanceTimersByTimeAsync(750)
    expect(sockets).toHaveLength(2)
    sockets[1].dispatchEvent(new Event('open'))
    sockets[0].dispatchEvent(new Event('close'))
    sockets[0].dispatchEvent(new MessageEvent('message', { data: JSON.stringify({ type: 'chat_message', payload: chatMessage() }) }))
    expect(store.connected).toBe(true)
    expect(store.messages).toEqual([])
  })
})
