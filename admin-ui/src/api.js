const adminMarker = '/admin'

export function gatewayPrefix(pathname = window.location.pathname) {
  const markerIndex = pathname.indexOf(adminMarker)
  if (markerIndex < 0) return ''
  return pathname.slice(0, markerIndex).replace(/\/$/, '')
}

export function readNavidromeProof(storage = window.localStorage) {
  return {
    username: storage.getItem('username') || '',
    salt: storage.getItem('subsonic-salt') || '',
    token: storage.getItem('subsonic-token') || '',
  }
}

export function hasCompleteProof(proof) {
  return Boolean(proof.username && /^[a-f\d]{6,64}$/i.test(proof.salt) && /^[a-f\d]{32}$/i.test(proof.token))
}

function mutationKey() {
  if (window.crypto?.randomUUID) return window.crypto.randomUUID()
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export class GatewayAPI {
  constructor(prefix = gatewayPrefix()) {
    this.prefix = prefix
    this.apiBase = `${prefix}/api/v1`
    this.session = null
  }

  async exchange(proof) {
    const response = await this.raw('/auth/exchange', {
      method: 'POST',
      body: proof,
      authenticated: false,
    })
    this.session = response
    return response
  }

  async raw(path, options = {}) {
    const { method = 'GET', body, authenticated = true } = options
    const headers = { Accept: 'application/json' }
    if (body !== undefined) headers['Content-Type'] = 'application/json'
    if (authenticated) {
      if (!this.session?.sessionToken) throw new Error('尚未建立听歌房会话')
      headers.Authorization = `Bearer ${this.session.sessionToken}`
    }
    if (!['GET', 'HEAD'].includes(method)) headers['Idempotency-Key'] = mutationKey()
    const response = await fetch(`${this.apiBase}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      credentials: 'same-origin',
    })
    if (response.status === 204) return null
    const payload = await response.json().catch(() => ({}))
    if (!response.ok) {
      const error = new Error(payload.error?.message || `请求失败 (${response.status})`)
      error.code = payload.error?.code || 'request_failed'
      error.status = response.status
      throw error
    }
    return payload
  }

  rooms() { return this.raw('/rooms') }
  createRoom(room) { return this.raw('/rooms', { method: 'POST', body: room }) }
  updateRoom(id, room) { return this.raw(`/rooms/${encodeURIComponent(id)}`, { method: 'PATCH', body: room }) }
  deleteRoom(id) { return this.raw(`/rooms/${encodeURIComponent(id)}`, { method: 'DELETE' }) }
  setRoomOpen(id, open) { return this.raw(`/rooms/${encodeURIComponent(id)}/${open ? 'reopen' : 'close'}`, { method: 'POST', body: {} }) }
  members(id) { return this.raw(`/rooms/${encodeURIComponent(id)}/members`) }
  removeMember(id, username) { return this.raw(`/rooms/${encodeURIComponent(id)}/members/${encodeURIComponent(username)}`, { method: 'DELETE' }) }
  invites(id) { return this.raw(`/rooms/${encodeURIComponent(id)}/invites`) }
  createInvite(id, invite) { return this.raw(`/rooms/${encodeURIComponent(id)}/invites`, { method: 'POST', body: invite }) }
  revokeInvite(id, inviteID) { return this.raw(`/rooms/${encodeURIComponent(id)}/invites/${encodeURIComponent(inviteID)}`, { method: 'DELETE' }) }

  async musicFolders(proof) {
    if (!this.session?.navidromeBaseURL) return []
    const endpoint = new URL(`${this.session.navidromeBaseURL.replace(/\/$/, '')}/rest/getMusicFolders.view`)
    endpoint.search = new URLSearchParams({
      u: proof.username,
      s: proof.salt,
      t: proof.token,
      v: '1.16.1',
      c: 'NavidromeMusicRoomAdmin',
      f: 'json',
    })
    const response = await fetch(endpoint, { credentials: 'same-origin' })
    const payload = await response.json().catch(() => ({}))
    const envelope = payload['subsonic-response']
    if (!response.ok || envelope?.status !== 'ok') throw new Error(envelope?.error?.message || '无法读取 Navidrome 音乐库')
    return envelope.musicFolders?.musicFolder || []
  }
}
