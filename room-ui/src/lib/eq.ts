import { clamp } from './sync'

export const eqFrequencies = [60, 230, 910, 3600, 14000] as const
export const eqPresets = {
  flat: [0, 0, 0, 0, 0], bass: [6, 3, 0, -1, -2], vocal: [-2, 0, 4, 3, 0],
  warm: [3, 2, 0, -1, -2], bright: [-2, -1, 0, 3, 5],
} as const
export type EqPreset = keyof typeof eqPresets | 'custom'
export function prefersNativeAudio(browser = navigator): boolean {
  return /Android|iPhone|iPad|iPod/i.test(browser.userAgent) ||
    (browser.maxTouchPoints > 1 && /Macintosh/i.test(browser.userAgent)) ||
    Boolean((browser as Navigator & { userAgentData?: { mobile?: boolean } }).userAgentData?.mobile)
}
export function readEq(storage?: Pick<Storage, 'getItem'>): { enabled: boolean; preset: EqPreset; gains: number[] } {
  try {
    const value = JSON.parse((storage ?? localStorage).getItem('nmr:eq:v1') || '{}')
    if (Array.isArray(value.gains) && value.gains.length === 5 && value.gains.every(Number.isFinite)) {
      return { enabled: value.enabled === true, preset: Object.hasOwn(eqPresets, value.preset) ? value.preset : 'custom', gains: value.gains.map((gain: number) => clamp(gain, -12, 12)) }
    }
  } catch { /* Invalid preferences fall back to flat. */ }
  return { enabled: false, preset: 'flat', gains: [...eqPresets.flat] }
}
