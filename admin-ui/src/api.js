export function gatewayPrefix(pathname = window.location.pathname) {
  const markerIndex = pathname.search(/\/admin(?:\/|$)/)
  if (markerIndex < 0) return ''
  return pathname.slice(0, markerIndex).replace(/\/$/, '')
}

export function readNavidromeProof(storage) {
  try {
    storage ??= window.localStorage
    return {
      username: storage.getItem('username') || '',
      salt: storage.getItem('subsonic-salt') || '',
      token: storage.getItem('subsonic-token') || '',
    }
  } catch {
    return { username: '', salt: '', token: '' }
  }
}

export function hasCompleteProof(proof) {
  return Boolean(proof?.username && /^[a-f\d]{6,64}$/i.test(proof.salt) && /^[a-f\d]{32}$/i.test(proof.token))
}

function mutationKey() {
  if (globalThis.crypto?.randomUUID) return globalThis.crypto.randomUUID()
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
    const { method = 'GET', body, authenticated = true, base = this.apiBase, signal } = options
    const headers = { Accept: 'application/json' }
    if (body !== undefined) headers['Content-Type'] = 'application/json'
    if (authenticated) {
      if (!this.session?.sessionToken) throw Object.assign(new Error('Room session required'), { code: 'session_required' })
      headers.Authorization = `Bearer ${this.session.sessionToken}`
    }
    if (!['GET', 'HEAD'].includes(method)) headers['Idempotency-Key'] = mutationKey()
    const response = await fetch(`${base}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      credentials: 'same-origin',
      cache: 'no-store',
      signal,
    })
    if (response.status === 204) return null
    const payload = await response.json().catch(() => ({}))
    if (!response.ok) {
      const error = new Error(payload.error?.message || `Request failed (${response.status})`)
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

  readyz(options = {}) { return this.raw('/readyz', { ...options, base: this.prefix, authenticated: false }) }
  updateStatus() { return this.raw('/admin/updates') }
  updateAction(action, version, options = {}) {
    return this.raw('/admin/updates', { ...options, method: 'POST', body: { action, ...(version ? { version } : {}) } })
  }
  activateNavidromePlugin(storage, fetchImpl) { return activateNavidromePlugin(storage, fetchImpl) }

  async exportDiagnostics() {
    const payload = await this.raw('/admin/diagnostics')
    if (payload?.redacted !== true) throw Object.assign(new Error('Diagnostics must be redacted'), { code: 'diagnostics_invalid' })
    const url = URL.createObjectURL(new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' }))
    const link = document.createElement('a')
    try {
      link.href = url
      link.download = `music-room-diagnostics-${new Date().toISOString().replace(/[:.]/g, '-')}.json`
      document.body.appendChild(link)
      link.click()
    } finally {
      link.remove()
      setTimeout(() => URL.revokeObjectURL(url), 1000)
    }
  }

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
    if (!response.ok || envelope?.status !== 'ok') throw Object.assign(new Error('Could not read Navidrome music libraries'), { code: 'navidrome_unavailable' })
    return envelope.musicFolders?.musicFolder || []
  }
}

export async function activateNavidromePlugin(storage, fetchImpl = fetch) {
  let step = 'token'
  try {
    storage ??= window.localStorage
    const token = storage.getItem('token')
    if (!token) return { activated: false, reason: 'missing_navidrome_token' }
    // The UI JWT belongs exclusively to these same-origin Navidrome endpoints.
    const headers = { Accept: 'application/json', 'X-ND-Authorization': `Bearer ${token}` }
    const request = async (path, method, body) => {
      const response = await fetchImpl(path, {
        method, headers: { ...headers, ...(body ? { 'Content-Type': 'application/json' } : {}) },
        ...(body ? { body: JSON.stringify(body) } : {}),
        credentials: 'same-origin', cache: 'no-store', redirect: 'error',
      })
      if (!response.ok) throw Object.assign(new Error('Navidrome plugin request failed'), { status: response.status })
      return response
    }
    step = 'rescan'
    await request('/api/plugin/rescan', 'POST')
    step = 'list'
    const response = await request('/api/plugin', 'GET')
    const plugins = await response.json()
    if (!Array.isArray(plugins)) throw new Error('Invalid Navidrome plugin list')
    let plugin = plugins.find((entry) => entry?.id === 'navidrome-music-room' || entry?.name === 'Navidrome Music Room')
    if (!plugin?.id) return { activated: false, reason: 'navidrome_plugin_not_found', step }
    if (plugin.enabled === false) {
      step = 'enable'
      await request(`/api/plugin/${encodeURIComponent(plugin.id)}`, 'PUT', { enabled: true })
      plugin = { ...plugin, enabled: true }
    } else if (plugin.enabled !== true) {
      throw new Error('Invalid Navidrome plugin state')
    }
    return { activated: true, plugin }
  } catch (error) {
    return {
      activated: false,
      reason: step === 'token' ? 'missing_navidrome_token' : 'navidrome_plugin_activation_failed',
      step,
      ...(error.status ? { status: error.status } : {}),
    }
  }
}
