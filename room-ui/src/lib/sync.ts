import type { PlaybackState } from '../types'

export function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(maximum, Math.max(minimum, value))
}

export function effectivePosition(state: PlaybackState, now = Date.now()): number {
  const duration = Math.max(0, state.currentTrack?.durationSeconds || 0)
  let position = Math.max(0, state.positionSeconds || 0)
  if (state.status === 'playing') {
    const anchor = Date.parse(state.anchorServerTime || state.serverTime)
    if (Number.isFinite(anchor)) position += Math.max(0, (now - anchor) / 1000)
  }
  return duration > 0 ? clamp(position, 0, duration) : position
}

export function needsHardSeek(current: number, expected: number, threshold = 2.25): boolean {
  return !Number.isFinite(current) || Math.abs(current - expected) > threshold
}

export function formatDuration(value: number): string {
  const seconds = Math.max(0, Math.floor(Number.isFinite(value) ? value : 0))
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`
}

export function parseLRC(value: string): Array<{ time: number; text: string }> {
  const lines: Array<{ time: number; text: string }> = []
  for (const row of String(value || '').split(/\r?\n/)) {
    const matches = [...row.matchAll(/\[(\d{1,3}):(\d{2})(?:[.:](\d{1,3}))?\]/g)]
    const text = row.replace(/\[[^\]]+\]/g, '').trim()
    if (!text) continue
    for (const match of matches) {
      const fraction = match[3] ? Number(`0.${match[3].padEnd(3, '0').slice(0, 3)}`) : 0
      lines.push({ time: Number(match[1]) * 60 + Number(match[2]) + fraction, text })
    }
  }
  return lines.sort((left, right) => left.time - right.time)
}
