import { describe, expect, it } from 'vitest'
import { gatewayPrefix, hasCompleteProof, readNavidromeProof } from './api'

describe('gateway path and Navidrome proof', () => {
  it('keeps an edge-proxy prefix in API requests', () => {
    expect(gatewayPrefix('/music-room/admin/')).toBe('/music-room')
    expect(gatewayPrefix('/admin/')).toBe('')
  })

  it('reads the same OpenSubsonic proof Navidrome stores after login', () => {
    const values = new Map([
      ['username', 'admin'],
      ['subsonic-salt', '012345'],
      ['subsonic-token', '0123456789abcdef0123456789abcdef'],
    ])
    const proof = readNavidromeProof({ getItem: (key) => values.get(key) || null })
    expect(hasCompleteProof(proof)).toBe(true)
  })
})
