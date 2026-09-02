import { describe, expect, it } from 'vitest'

import type { PlaybackState } from '../types'
import { effectivePosition, formatDuration, needsHardSeek, parseLRC } from './sync'

function state(overrides: Partial<PlaybackState> = {}): PlaybackState {
  return {
    revision: 7,
    serverTime: '2026-09-02T12:00:00.000Z',
    anchorServerTime: '2026-09-02T12:00:00.000Z',
    status: 'playing',
    pausedForEmpty: false,
    positionSeconds: 14,
    currentTrack: {
      id: 'track-1', musicFolderID: 1, title: 'Track', artist: 'Artist', album: 'Album', durationSeconds: 180,
    },
    ...overrides,
  }
}

describe('authoritative playback clock', () => {
  it('advances a playing state from its server anchor', () => {
    expect(effectivePosition(state(), Date.parse('2026-09-02T12:00:03.250Z'))).toBe(17.25)
  })

  it('does not advance paused playback and clamps to duration', () => {
    expect(effectivePosition(state({ status: 'paused' }), Date.parse('2026-09-02T12:01:00Z'))).toBe(14)
    expect(effectivePosition(state({ positionSeconds: 179 }), Date.parse('2026-09-02T12:00:10Z'))).toBe(180)
  })

  it('only hard-seeks when drift exceeds the tolerance', () => {
    expect(needsHardSeek(20, 21.9)).toBe(false)
    expect(needsHardSeek(20, 22.3)).toBe(true)
  })
})

describe('media presentation helpers', () => {
  it('formats durations and parses multiple synchronized LRC timestamps', () => {
    expect(formatDuration(125.9)).toBe('2:05')
    expect(parseLRC('[00:02.50][00:04.000]Hello\nmetadata only')).toEqual([
      { time: 2.5, text: 'Hello' },
      { time: 4, text: 'Hello' },
    ])
  })
})
