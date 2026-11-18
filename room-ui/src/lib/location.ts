import { errorText } from './i18n'

import type { NavidromeProof } from '../types'

const proofPattern = /^[a-f\d]{32}$/i
const saltPattern = /^[a-f\d]{6,64}$/i

export function routeContext(
  pathname = window.location.pathname,
  hash = window.location.hash,
): { gatewayPrefix: string; roomID: string; invitation: string } {
  const match = pathname.match(/^(.*)\/join\/([a-f\d]{32})(?:\/.*)?$/i)
  return {
    gatewayPrefix: match?.[1]?.replace(/\/$/, '') || '',
    roomID: match?.[2] || '',
    invitation: invitationFromHash(hash),
  }
}

export function invitationFromHash(hash = window.location.hash): string {
  return new URLSearchParams(hash.replace(/^#/, '')).get('invite')?.trim() || ''
}

export function clearInvitation(location = window.location, history = window.history): void {
  if (!location.hash) return
  history.replaceState(null, '', `${location.pathname}${location.search}`)
}

export function readProof(storage = window.localStorage): NavidromeProof | null {
  const proof = {
    username: storage.getItem('username') || '',
    salt: storage.getItem('subsonic-salt') || '',
    token: storage.getItem('subsonic-token') || '',
  }
  return proof.username && saltPattern.test(proof.salt) && proofPattern.test(proof.token) ? proof : null
}

export function storeNavidromeLogin(
  value: Record<string, unknown>,
  storage = window.localStorage,
): NavidromeProof {
  const username = String(value.username || '')
  const salt = String(value.subsonicSalt || '')
  const token = String(value.subsonicToken || '')
  if (!username || !saltPattern.test(salt) || !proofPattern.test(token)) {
    throw new Error(errorText('navidrome_login_failed'))
  }
  if (value.token) storage.setItem('token', String(value.token))
  if (value.id) storage.setItem('userId', String(value.id))
  storage.setItem('name', String(value.name || username))
  storage.setItem('username', username)
  storage.setItem('role', value.isAdmin ? 'admin' : 'regular')
  storage.setItem('subsonic-salt', salt)
  storage.setItem('subsonic-token', token)
  storage.setItem('is-authenticated', 'true')
  return { username, salt, token }
}

export function navidromeAppURL(): string {
  return `${window.location.origin}/app/`
}

export function resourceFromURL(value: string, origin = window.location.origin): import('../types').ChatReference | null {
  try {
    const url = new URL(value, origin)
    if (url.origin !== origin || !['http:', 'https:'].includes(url.protocol)) return null
    const params = new URLSearchParams(url.hash.slice(1))
    const kind = params.get('resource')
    const id = params.get('id')
    if (!id || !['track', 'album', 'artist'].includes(kind || '')) return null
    return { kind: kind as 'track' | 'album' | 'artist', id, title: params.get('title') || '' }
  } catch { return null }
}

export function safeResourceURL(value: string): string {
  try {
    const url = new URL(value)
    return ['http:', 'https:'].includes(url.protocol) && !url.username && !url.password ? url.href : ''
  } catch { return '' }
}
