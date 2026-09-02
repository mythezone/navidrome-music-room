import { beforeEach, describe, expect, it, vi } from 'vitest'

import { clearInvitation, readProof, routeContext, storeNavidromeLogin } from './location'

describe('room share links', () => {
  it('extracts a reverse-proxy prefix, room ID and fragment invite', () => {
    expect(routeContext('/music-room/join/0123456789abcdef0123456789abcdef/', '#invite=secret%20token')).toEqual({
      gatewayPrefix: '/music-room',
      roomID: '0123456789abcdef0123456789abcdef',
      invitation: 'secret token',
    })
  })

  it('returns an empty room for malformed links', () => {
    expect(routeContext('/music-room/join/not-a-room', '')).toEqual({ gatewayPrefix: '', roomID: '', invitation: '' })
  })
})

describe('Navidrome proof storage', () => {
  beforeEach(() => window.localStorage.clear())

  it('persists and reads only an OpenSubsonic salt/token proof', () => {
    const proof = storeNavidromeLogin({
      username: 'listener', name: 'Listener', isAdmin: false,
      token: 'short-ui-jwt', subsonicSalt: 'abc123', subsonicToken: '0123456789abcdef0123456789abcdef',
    })
    expect(proof).toEqual({ username: 'listener', salt: 'abc123', token: '0123456789abcdef0123456789abcdef' })
    expect(readProof()).toEqual(proof)
    expect(window.localStorage.getItem('role')).toBe('regular')
  })

  it('clears an invitation fragment without changing the room path', () => {
    const replaceState = vi.fn()
    clearInvitation(
      { hash: '#invite=secret', pathname: '/music-room/join/0123/', search: '?source=qr' } as Location,
      { replaceState } as unknown as History,
    )
    expect(replaceState).toHaveBeenCalledWith(null, '', '/music-room/join/0123/?source=qr')
  })
})
