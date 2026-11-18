export function sameVersion(left, right) {
  return typeof left === 'string' && typeof right === 'string' && Boolean(left) && Boolean(right) && left.replace(/^v/, '') === right.replace(/^v/, '')
}

function delay(ms, signal) {
  return new Promise((resolve, reject) => {
    const abort = () => {
      clearTimeout(timer)
      reject(new DOMException('Aborted', 'AbortError'))
    }
    const timer = setTimeout(() => {
      signal?.removeEventListener('abort', abort)
      resolve()
    }, ms)
    if (signal?.aborted) abort()
    else signal?.addEventListener('abort', abort, { once: true })
  })
}

export async function waitForReady(api, version, paired, signal) {
  const deadline = Date.now() + 180000
  while (Date.now() < deadline) {
    if (signal?.aborted) throw new DOMException('Aborted', 'AbortError')
    const request = new AbortController()
    const abort = () => request.abort()
    signal?.addEventListener('abort', abort, { once: true })
    const timeout = setTimeout(abort, 5000)
    try {
      const health = await api.readyz({ signal: request.signal })
      if (health.status === 'ready' && sameVersion(health.version, version) && (!paired || (health.plugin === 'paired' && sameVersion(health.pluginVersion, version)))) return health
    } catch {
      // A restart can temporarily refuse connections or return a proxy error.
    } finally {
      clearTimeout(timeout)
      signal?.removeEventListener('abort', abort)
    }
    await delay(2000, signal)
  }
  throw Object.assign(new Error('Readiness confirmation timed out'), { code: paired ? 'plugin_ready_timeout' : 'gateway_ready_timeout' })
}

export async function runUpgrade(api, action, version, onStep, signal) {
  onStep(0, version)
  const result = await api.updateAction(action, action === 'install' ? version : undefined, { signal })
  const target = result?.version
  if (!target || (action === 'install' && (!result.activationScheduled || !sameVersion(target, version)))) {
    throw Object.assign(new Error('Unconfirmed update target'), { code: 'update_response_invalid' })
  }
  onStep(1, target)
  await delay(1000, signal)
  onStep(2, target)
  await waitForReady(api, target, false, signal)
  onStep(3, target)
  const activation = await api.activateNavidromePlugin()
  if (!activation.activated) throw Object.assign(new Error('Plugin activation failed'), { code: activation.reason })
  return waitForReady(api, target, true, signal)
}
