import { errorText } from './i18n'
import type {
  ChatMessage, ChatPayload, MemberProfile, Period, PlaybackCommand, RoomSettings,
  RoomRankings, RoomStatistics, RoomWish, StickerPack, WishPayload, WishFulfillment,
  GatewayErrorBody,
  GatewaySession,
  HistoryEntry,
  NavidromeProof,
  PlaybackState,
  QueueEntry,
  QueueMutation,
  Room,
  RoomSnapshot,
  TrackRef,
} from '../types'

export class GatewayError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly code: string,
    public readonly details?: unknown,
  ) {
    super(message)
  }
}

function idempotencyKey(): string {
  return globalThis.crypto?.randomUUID?.() || `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export async function navidromeLogin(username: string, password: string): Promise<Record<string, unknown>> {
  const response = await fetch('/auth/login', {
    method: 'POST',
    headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
    credentials: 'same-origin',
  })
  const payload = await response.json().catch(() => ({}))
  if (!response.ok) throw new GatewayError(errorText('navidrome_login_failed'), response.status, 'navidrome_login_failed')
  return payload
}

export class GatewayClient {
  private session: GatewaySession | null = null

  constructor(private readonly prefix: string) {}

  get currentSession(): GatewaySession | null {
    return this.session
  }

  clearSession(): void {
    this.session = null
  }

  async exchange(proof: NavidromeProof): Promise<GatewaySession> {
    const session = await this.request<GatewaySession>('/auth/exchange', {
      method: 'POST',
      body: proof,
      authenticated: false,
    })
    this.session = session
    return session
  }

  async request<T>(
    path: string,
    options: { method?: string; body?: unknown; authenticated?: boolean } = {},
  ): Promise<T> {
    const method = options.method || 'GET'
    const authenticated = options.authenticated !== false
    const headers: Record<string, string> = { Accept: 'application/json' }
    if (options.body !== undefined) headers['Content-Type'] = 'application/json'
    if (authenticated) {
      if (!this.session?.sessionToken) throw new GatewayError(errorText('session_required'), 401, 'session_missing')
      headers.Authorization = `Bearer ${this.session.sessionToken}`
    }
    if (!['GET', 'HEAD'].includes(method)) headers['Idempotency-Key'] = idempotencyKey()
    const response = await fetch(`${this.prefix}/api/v1${path}`, {
      method,
      headers,
      credentials: 'same-origin',
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
    })
    if (response.status === 204) return undefined as T
    const payload = (await response.json().catch(() => ({}))) as GatewayErrorBody & T
    if (!response.ok) {
      throw new GatewayError(
        errorText(payload.error?.code || 'request_failed'),
        response.status,
        payload.error?.code || 'request_failed',
        payload.error?.details,
      )
    }
    return payload
  }

  rooms(): Promise<{ rooms: Room[] }> {
    return this.request('/rooms')
  }

  join(roomID: string): Promise<{ room: Room }> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/join`, { method: 'POST', body: {} })
  }

  redeem(roomID: string, invite: string): Promise<{ room: Room }> {
    return this.request('/invites/redeem', { method: 'POST', body: { roomID, invite } })
  }

  snapshot(roomID: string): Promise<RoomSnapshot> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/snapshot`)
  }

  history(roomID: string, offset = 0): Promise<{ items: HistoryEntry[] }> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/history?limit=100&offset=${offset}`)
  }

  playback(roomID: string, command: PlaybackCommand, expectedRevision: number, positionSeconds?: number): Promise<PlaybackState> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/playback`, {
      method: 'POST',
      body: { command, expectedRevision, ...(positionSeconds === undefined ? {} : { positionSeconds }) },
    })
  }

  addTrack(roomID: string, track: TrackRef, playNext = false): Promise<QueueMutation> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/queue/tracks`, {
      method: 'POST',
      body: { track, playNext },
    })
  }

  removeQueue(roomID: string, queueID: string): Promise<void> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/queue/${encodeURIComponent(queueID)}`, {
      method: 'DELETE',
    })
  }

  reorderQueue(roomID: string, entries: QueueEntry[]): Promise<{ queue: QueueEntry[] }> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/queue/order`, {
      method: 'PUT',
      body: { queueIDs: entries.map((entry) => entry.queueID) },
    })
  }

  websocketTicket(roomID: string): Promise<{ webSocketURL: string; expiresAt: string }> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/ws-ticket`, { method: 'POST', body: {} })
  }


  patchRoom(roomID: string, patch: Partial<RoomSettings>): Promise<Room> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}`, { method: 'PATCH', body: patch })
  }

  shuffleQueue(roomID: string): Promise<{ queue: QueueEntry[] }> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/queue/shuffle`, { method: 'POST', body: {} })
  }

  topQueue(roomID: string, queueID: string): Promise<{ queue: QueueEntry[] }> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/queue/${encodeURIComponent(queueID)}/top`, { method: 'POST', body: {} })
  }

  requeueHistory(roomID: string, historyID: string): Promise<QueueMutation> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/history/${encodeURIComponent(historyID)}/requeue`, { method: 'POST', body: {} })
  }

  stickers(roomID: string): Promise<{ packs: StickerPack[] }> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/stickers`)
  }

  chatMessages(roomID: string, limit = 50, before = ''): Promise<{ messages: ChatMessage[]; hasMore: boolean }> {
    const query = new URLSearchParams({ limit: String(limit) })
    if (before) query.set('before', before)
    return this.request(`/rooms/${encodeURIComponent(roomID)}/chat?${query}`)
  }

  sendMessage(roomID: string, payload: ChatPayload): Promise<ChatMessage> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/chat`, { method: 'POST', body: payload })
  }

  recallMessage(roomID: string, messageID: string): Promise<ChatMessage> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/chat/${encodeURIComponent(messageID)}/recall`, { method: 'POST', body: {} })
  }

  deleteMessage(roomID: string, messageID: string): Promise<void> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/chat/${encodeURIComponent(messageID)}`, { method: 'DELETE' })
  }

  statistics(roomID: string, period: Period): Promise<RoomStatistics> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/statistics?period=${period}`)
  }

  rankings(roomID: string, period: Period): Promise<RoomRankings> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/rankings?period=${period}`)
  }

  achievements(roomID: string): Promise<MemberProfile> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/achievements`)
  }

  memberProfile(roomID: string, username: string): Promise<MemberProfile> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/members/${encodeURIComponent(username)}/profile`)
  }

  selectBadge(roomID: string, badgeID: string): Promise<MemberProfile> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/achievements/selected`, { method: 'PUT', body: { badgeID } })
  }

  wishes(roomID: string, status: 'pending' | 'fulfilled' | '' = ''): Promise<{ wishes: RoomWish[] }> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/wishes${status ? `?status=${status}` : ''}`)
  }

  createWish(roomID: string, payload: WishPayload): Promise<RoomWish> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/wishes`, { method: 'POST', body: payload })
  }

  fulfillWish(roomID: string, wishID: string, payload: WishFulfillment): Promise<RoomWish> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/wishes/${encodeURIComponent(wishID)}/fulfill`, { method: 'POST', body: payload })
  }

  deleteWish(roomID: string, wishID: string): Promise<void> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/wishes/${encodeURIComponent(wishID)}`, { method: 'DELETE' })
  }

  removeMember(roomID: string, username: string): Promise<void> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/members/${encodeURIComponent(username)}`, { method: 'DELETE' })
  }

  leave(roomID: string): Promise<void> {
    return this.request(`/rooms/${encodeURIComponent(roomID)}/leave`, { method: 'POST', body: {} })
  }

  capabilities(): Promise<{ features: Record<string, boolean> }> {
    return this.request('/capabilities')
  }
}
