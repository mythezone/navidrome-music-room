import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { runUpgrade, sameVersion, waitForReady } from './updates'

const ready = (version, pluginVersion = version, plugin = 'paired') => ({ status: 'ready', version, pluginVersion, plugin })

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe('upgrade readiness', () => {
  it('normalizes release tag prefixes without accepting missing or different versions', () => {
    expect(sameVersion('v1.2.0', '1.2.0')).toBe(true)
    expect(sameVersion('1.2.0-beta.1', '1.2.0')).toBe(false)
    expect(sameVersion(undefined, undefined)).toBe(false)
    expect(sameVersion('', '')).toBe(false)
  })

  it('waits through restart, wrong versions and stale heartbeats before reporting success', async () => {
    const api = {
      updateAction: vi.fn().mockResolvedValue({ activationScheduled: true, version: 'v1.2.0' }),
      readyz: vi.fn()
        .mockRejectedValueOnce(new Error('connection refused'))
        .mockResolvedValueOnce(ready('1.1.0'))
        .mockResolvedValueOnce(ready('1.2.0', '1.1.0', 'stale'))
        .mockResolvedValueOnce(ready('1.2.0', '1.1.0'))
        .mockResolvedValueOnce(ready('1.2.0', '1.2.0', 'stale'))
        .mockResolvedValue(ready('1.2.0')),
      activateNavidromePlugin: vi.fn().mockResolvedValue({ activated: true }),
    }
    const steps = vi.fn()
    const result = runUpgrade(api, 'install', 'v1.2.0', steps)
    const assertion = expect(result).resolves.toEqual(ready('1.2.0'))
    expect(api.activateNavidromePlugin).not.toHaveBeenCalled()
    await vi.runAllTimersAsync()
    await assertion
    expect(api.updateAction).toHaveBeenCalledWith('install', 'v1.2.0', { signal: undefined })
    expect(api.activateNavidromePlugin).toHaveBeenCalledOnce()
    expect(api.readyz).toHaveBeenCalledTimes(6)
    expect(steps.mock.calls).toEqual([[0, 'v1.2.0'], [1, 'v1.2.0'], [2, 'v1.2.0'], [3, 'v1.2.0']])
  })

  it('uses the rollback response target, not the latest release', async () => {
    const api = {
      updateAction: vi.fn().mockResolvedValue({ version: 'v1.1.0', state: 'rollback_requested' }),
      readyz: vi.fn().mockResolvedValue(ready('1.1.0')),
      activateNavidromePlugin: vi.fn().mockResolvedValue({ activated: true }),
    }
    const assertion = expect(runUpgrade(api, 'rollback', '1.2.0', vi.fn())).resolves.toEqual(ready('1.1.0'))
    await vi.runAllTimersAsync()
    await assertion
    expect(api.updateAction).toHaveBeenCalledWith('rollback', undefined, { signal: undefined })
  })

  it.each([{ version: '1.3.0', activationScheduled: true }, { version: '1.2.0' }, {}])('rejects unconfirmed installation %j', async (response) => {
    const api = { updateAction: vi.fn().mockResolvedValue(response), readyz: vi.fn(), activateNavidromePlugin: vi.fn() }
    await expect(runUpgrade(api, 'install', '1.2.0', vi.fn())).rejects.toMatchObject({ code: 'update_response_invalid' })
    expect(api.readyz).not.toHaveBeenCalled()
    expect(api.activateNavidromePlugin).not.toHaveBeenCalled()
  })

  it('stops when signature verification or plugin activation fails', async () => {
    const api = { updateAction: vi.fn().mockRejectedValue({ code: 'update_signature_invalid' }), readyz: vi.fn().mockResolvedValue(ready('1.2.0')), activateNavidromePlugin: vi.fn().mockResolvedValue({ activated: false, reason: 'missing_navidrome_token' }) }
    await expect(runUpgrade(api, 'install', '1.2.0', vi.fn())).rejects.toMatchObject({ code: 'update_signature_invalid' })
    expect(api.readyz).not.toHaveBeenCalled()
    api.updateAction.mockResolvedValue({ version: '1.2.0', activationScheduled: true })
    const assertion = expect(runUpgrade(api, 'install', '1.2.0', vi.fn())).rejects.toMatchObject({ code: 'missing_navidrome_token' })
    await vi.runAllTimersAsync()
    await assertion
    expect(api.readyz).toHaveBeenCalledOnce()
  })

  it.each([false, true])('bounds polling with a descriptive timeout (paired=%s)', async (paired) => {
    const api = { readyz: vi.fn().mockResolvedValue(ready('1.1.0')) }
    const assertion = expect(waitForReady(api, '1.2.0', paired)).rejects.toMatchObject({ code: paired ? 'plugin_ready_timeout' : 'gateway_ready_timeout' })
    await vi.runAllTimersAsync()
    await assertion
    expect(api.readyz.mock.calls.length).toBeLessThanOrEqual(90)
  })

  it('aborts hanging health requests and eventually times out', async () => {
    const api = { readyz: vi.fn(({ signal }) => new Promise((resolve, reject) => signal.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')), { once: true }))) }
    const assertion = expect(waitForReady(api, '1.2.0', false)).rejects.toMatchObject({ code: 'gateway_ready_timeout' })
    await vi.runAllTimersAsync()
    await assertion
  })

  it('cancels polling when the UI unmounts', async () => {
    const api = { readyz: vi.fn().mockResolvedValue(ready('1.1.0')) }
    const controller = new AbortController()
    const assertion = expect(waitForReady(api, '1.2.0', true, controller.signal)).rejects.toMatchObject({ name: 'AbortError' })
    await vi.advanceTimersByTimeAsync(1000)
    controller.abort()
    await vi.runAllTimersAsync()
    await assertion
    expect(api.readyz).toHaveBeenCalledOnce()
    expect(vi.getTimerCount()).toBe(0)
  })
})
