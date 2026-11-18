import { afterEach, describe, expect, it, vi } from 'vitest'
import { activateNavidromePlugin, GatewayAPI, gatewayPrefix, hasCompleteProof, readNavidromeProof } from './api'
import { detectLocale, formatDate, locales, localizeError, messages, translate } from './i18n'

const storage = (values = {}) => ({ getItem: (key) => values[key] ?? null })
const json = (body, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

describe('gateway path and Navidrome proof', () => {
  it.each([
    ['/music-room/admin/', '/music-room'], ['/admin/', ''], ['/admin', ''],
    ['/nested/rooms/admin/settings', '/nested/rooms'], ['/administrator', ''], ['/app/', ''],
  ])('resolves %s to %s', (pathname, prefix) => {
    expect(gatewayPrefix(pathname)).toBe(prefix)
  })

  it('reads OpenSubsonic credentials, never the Navidrome UI JWT', () => {
    const proof = readNavidromeProof(storage({ username: 'admin', 'subsonic-salt': '012345', 'subsonic-token': '0123456789abcdef0123456789abcdef', token: 'private-ui-jwt' }))
    expect(proof).toEqual({ username: 'admin', salt: '012345', token: '0123456789abcdef0123456789abcdef' })
    expect(hasCompleteProof(proof)).toBe(true)
  })

  it.each([
    undefined, {}, { username: 'a', salt: 'short', token: 'a'.repeat(32) },
    { username: 'a', salt: 'z12345', token: 'a'.repeat(32) },
    { username: 'a', salt: 'a'.repeat(65), token: 'a'.repeat(32) },
    { username: 'a', salt: 'abcdef', token: 'not-a-subsonic-token' },
    { username: '', salt: 'abcdef', token: 'a'.repeat(32) },
  ])('rejects incomplete or malformed proof %j', (proof) => {
    expect(hasCompleteProof(proof)).toBe(false)
  })

  it('treats missing or blocked storage as missing credentials', () => {
    expect(readNavidromeProof(storage())).toEqual({ username: '', salt: '', token: '' })
    expect(hasCompleteProof(readNavidromeProof({ getItem() { throw new Error('blocked') } }))).toBe(false)
  })
})

describe('locale resolution and translations', () => {
  it('prefers the explicit selection, then Navidrome settings, then browser preferences', () => {
    const navigator = { languages: ['ja-JP'], language: 'fr-FR' }
    expect(detectLocale(storage({ 'nmr:locale': 'es', i18nextLng: 'de', locale: 'zh-CN' }), navigator)).toBe('es')
    expect(detectLocale(storage({ i18nextLng: 'de', locale: 'zh-CN' }), navigator)).toBe('de')
    expect(detectLocale(storage({ locale: 'zh-CN' }), navigator)).toBe('zh-CN')
    expect(detectLocale(storage(), navigator)).toBe('ja')
    expect(detectLocale(storage(), { language: 'fr-FR' })).toBe('fr')
    expect(detectLocale(storage(), {})).toBe('en')
  })

  it.each([
    ['zh-TW', 'zh-TW'], ['zh-HK', 'zh-TW'], ['zh-Hant', 'zh-TW'], ['zh-Hant-HK', 'zh-TW'],
    ['zh_CN', 'zh-CN'], ['zh-SG', 'zh-CN'], ['zh', 'zh-CN'], ['ja-JP', 'ja'],
    ['es-MX', 'es'], ['fr-CA', 'fr'], ['de-AT', 'de'], ['en-GB', 'en'],
  ])('maps browser locale %s to %s', (input, expected) => {
    expect(detectLocale(storage(), { language: input })).toBe(expected)
  })

  it('skips unsupported values and tolerates disabled storage', () => {
    expect(detectLocale(storage({ 'nmr:locale': 'xx', i18nextLng: 'de-DE' }), {})).toBe('de')
    expect(detectLocale(storage(), { languages: ['xx', 'es'], language: 'ja' })).toBe('es')
    expect(detectLocale(storage(), { languages: ['xx'], language: 'xx' })).toBe('en')
    expect(detectLocale({ getItem() { throw new Error('blocked') } }, { language: 'ja' })).toBe('ja')
  })

  it('falls back to English and interpolates without interpreting HTML', () => {
    expect(translate('unknown', 'save')).toBe('Save')
    const original = messages.fr.save
    try {
      messages.fr.save = undefined
      expect(translate('fr', 'save')).toBe('Save')
    } finally { messages.fr.save = original }
    expect(translate('en', 'shareTitle', { name: '<b>Room</b>' })).toBe('Share “<b>Room</b>”')
    expect(translate('en', 'missing_key')).toBe('missing_key')
    expect(translate('en', 'toString')).toBe('toString')
  })

  it('provides every message and matching placeholders in all seven languages', () => {
    expect(Object.keys(locales)).toHaveLength(7)
    for (const locale of Object.keys(locales)) {
      expect(Object.keys(messages[locale])).toEqual(Object.keys(messages.en))
      for (const [key, value] of Object.entries(messages[locale])) {
        expect(value, `${locale}: ${key}`).toBeTruthy()
        expect((value.match(/\{\w+\}/g) || []).sort(), `${locale}: ${key}`).toEqual((messages.en[key].match(/\{\w+\}/g) || []).sort())
      }
    }
  })

  it.each(['room_closed', 'membership_required', 'library_access_required', 'personal_queue_limit', 'room_manager_required', 'skip_permission_required', 'launcher_required', 'plugin_not_paired', 'configuration_mismatch', 'update_check_failed', 'rollback_unavailable', 'update_signature_invalid', 'session_expired'])('localizes backend error %s in every locale', (code) => {
    for (const locale of Object.keys(locales)) {
      const translated = localizeError(locale, { code, message: 'Untranslated backend message' })
      expect(translated).not.toBe(translate(locale, 'error.request_failed'))
      expect(translated).not.toContain(code)
      if (locale !== 'en') expect(translated).not.toBe(localizeError('en', { code }))
    }
  })

  it('does not leak arbitrary backend or credential-bearing error messages', () => {
    expect(localizeError('de', { code: 'unknown', message: 'secret-token' })).toBe(translate('de', 'error.request_failed'))
    expect(localizeError('en', { reason: 'missing_navidrome_token' })).toMatch(/Sign in to Navidrome/)
  })

  it('formats dates in the active locale and handles missing or invalid dates', () => {
    const value = '2026-11-18T12:30:00Z'
    for (const locale of Object.keys(locales)) {
      expect(formatDate(value, locale)).toBe(new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)))
    }
    expect(formatDate(null, 'en')).toBe('—')
    expect(formatDate('invalid', 'en')).toBe('—')
  })
})

describe('Navidrome plugin activation', () => {
  it('does not make a request without a UI JWT', async () => {
    const fetchImpl = vi.fn()
    expect(await activateNavidromePlugin(storage(), fetchImpl)).toEqual({ activated: false, reason: 'missing_navidrome_token' })
    expect(fetchImpl).not.toHaveBeenCalled()
  })

  it('rescans, finds by ID, then enables a disabled plugin using only same-origin Navidrome endpoints', async () => {
    const plugin = { id: 'navidrome-music-room', name: 'Localized name', enabled: false }
    const fetchImpl = vi.fn().mockResolvedValueOnce(new Response(null, { status: 204 })).mockResolvedValueOnce(json([plugin])).mockResolvedValueOnce(json({}))
    expect(await activateNavidromePlugin(storage({ token: 'ui-jwt' }), fetchImpl)).toEqual({ activated: true, plugin: { ...plugin, enabled: true } })
    expect(fetchImpl.mock.calls.map(([url, options]) => [url, options.method])).toEqual([
      ['/api/plugin/rescan', 'POST'], ['/api/plugin', 'GET'], ['/api/plugin/navidrome-music-room', 'PUT'],
    ])
    for (const [, options] of fetchImpl.mock.calls) {
      expect(options.headers['X-ND-Authorization']).toBe('Bearer ui-jwt')
      expect(options.headers.Authorization).toBeUndefined()
      expect(options.credentials).toBe('same-origin')
      expect(options.redirect).toBe('error')
    }
    expect(JSON.parse(fetchImpl.mock.calls[2][1].body)).toEqual({ enabled: true })
  })

  it('finds by name and URL-encodes the plugin ID', async () => {
    const fetchImpl = vi.fn().mockResolvedValueOnce(json({})).mockResolvedValueOnce(json([{ id: 'custom/id', name: 'Navidrome Music Room', enabled: false }])).mockResolvedValueOnce(json({}))
    await activateNavidromePlugin(storage({ token: 'jwt' }), fetchImpl)
    expect(fetchImpl.mock.calls[2][0]).toBe('/api/plugin/custom%2Fid')
  })

  it('does not PUT an already enabled plugin', async () => {
    const fetchImpl = vi.fn().mockResolvedValueOnce(json({})).mockResolvedValueOnce(json([{ id: 'navidrome-music-room', enabled: true }]))
    expect((await activateNavidromePlugin(storage({ token: 'jwt' }), fetchImpl)).activated).toBe(true)
    expect(fetchImpl).toHaveBeenCalledTimes(2)
  })

  it.each([
    [[], 'navidrome_plugin_not_found'],
    [{ plugins: [] }, 'navidrome_plugin_activation_failed'],
    [[{ id: 'navidrome-music-room' }], 'navidrome_plugin_activation_failed'],
  ])('reports missing or invalid plugin data %j', async (body, reason) => {
    const fetchImpl = vi.fn().mockResolvedValueOnce(json({})).mockResolvedValueOnce(json(body))
    expect(await activateNavidromePlugin(storage({ token: 'jwt' }), fetchImpl)).toMatchObject({ activated: false, reason })
  })

  it.each(['rescan', 'list', 'enable'])('reports %s failures with no token leakage', async (step) => {
    const fetchImpl = vi.fn()
    if (step !== 'rescan') fetchImpl.mockResolvedValueOnce(json({}))
    if (step === 'enable') fetchImpl.mockResolvedValueOnce(json([{ id: 'navidrome-music-room', enabled: false }]))
    fetchImpl.mockResolvedValueOnce(json({ message: 'ui-jwt' }, 403))
    const result = await activateNavidromePlugin(storage({ token: 'ui-jwt' }), fetchImpl)
    expect(result).toEqual({ activated: false, reason: 'navidrome_plugin_activation_failed', step, status: 403 })
    expect(JSON.stringify(result)).not.toContain('ui-jwt')
  })

  it('returns structured failures for unavailable storage or network', async () => {
    expect(await activateNavidromePlugin({ getItem() { throw new Error('blocked') } }, vi.fn())).toMatchObject({ activated: false, reason: 'missing_navidrome_token' })
    expect(await activateNavidromePlugin(storage({ token: 'jwt' }), vi.fn().mockRejectedValue(new TypeError('offline')))).toMatchObject({ activated: false, reason: 'navidrome_plugin_activation_failed', step: 'rescan' })
  })
})

describe('GatewayAPI administration', () => {
  function authenticatedAPI() {
    const api = new GatewayAPI('/music-room')
    api.session = { sessionToken: 'gateway-session' }
    return api
  }

  it('reads /readyz outside /api/v1 without authentication or caching', async () => {
    const fetchMock = vi.fn().mockResolvedValue(json({ status: 'ready', version: '1.2.0' }))
    vi.stubGlobal('fetch', fetchMock)
    expect(await new GatewayAPI('/music-room').readyz()).toEqual({ status: 'ready', version: '1.2.0' })
    expect(fetchMock).toHaveBeenCalledWith('/music-room/readyz', expect.objectContaining({ method: 'GET', cache: 'no-store' }))
    expect(fetchMock.mock.calls[0][1].headers.Authorization).toBeUndefined()
  })

  it('authenticates status and action calls with the gateway session only', async () => {
    const fetchMock = vi.fn().mockImplementation(async () => json({ status: { currentVersion: '1.1.0' } }))
    vi.stubGlobal('fetch', fetchMock)
    const api = authenticatedAPI()
    await api.updateStatus()
    await api.updateAction('check')
    await api.updateAction('install', 'v1.2.0')
    await api.updateAction('rollback')
    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual(Array(4).fill('/music-room/api/v1/admin/updates'))
    expect(fetchMock.mock.calls[0][1].method).toBe('GET')
    expect(fetchMock.mock.calls.slice(1).map(([, options]) => JSON.parse(options.body))).toEqual([{ action: 'check' }, { action: 'install', version: 'v1.2.0' }, { action: 'rollback' }])
    for (const [, options] of fetchMock.mock.calls) {
      expect(options.headers.Authorization).toBe('Bearer gateway-session')
      expect(options.headers['X-ND-Authorization']).toBeUndefined()
      if (options.method === 'POST') expect(options.headers['Idempotency-Key']).toBeTruthy()
    }
  })

  it('requires authentication and preserves backend error codes', async () => {
    const fetchMock = vi.fn().mockResolvedValue(json({ error: { code: 'launcher_required', message: 'Launcher required' } }, 409))
    vi.stubGlobal('fetch', fetchMock)
    await expect(new GatewayAPI('').updateStatus()).rejects.toMatchObject({ code: 'session_required' })
    expect(fetchMock).not.toHaveBeenCalled()
    await expect(authenticatedAPI().updateAction('install')).rejects.toMatchObject({ code: 'launcher_required', status: 409 })
  })

  it('passes allowMemberSkip through create and edit payloads', async () => {
    const fetchMock = vi.fn().mockImplementation(async () => json({ roomID: 'room' }))
    vi.stubGlobal('fetch', fetchMock)
    const api = authenticatedAPI()
    await api.createRoom({ name: 'Room', allowMemberSkip: true })
    await api.updateRoom('room/id', { name: 'Room', allowMemberSkip: false })
    expect(fetchMock.mock.calls.map(([, options]) => JSON.parse(options.body).allowMemberSkip)).toEqual([true, false])
    expect(fetchMock.mock.calls[1][0]).toBe('/music-room/api/v1/rooms/room%2Fid')
  })

  it('downloads the authenticated redacted JSON and revokes its blob URL', async () => {
    vi.useFakeTimers()
    const payload = { redacted: true, format: 'navidrome-music-room-diagnostics/v1' }
    const fetchMock = vi.fn().mockResolvedValue(json(payload))
    vi.stubGlobal('fetch', fetchMock)
    const link = { click: vi.fn(), remove: vi.fn() }
    vi.stubGlobal('document', { createElement: vi.fn(() => link), body: { appendChild: vi.fn() } })
    const createURL = vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:diagnostics')
    const revokeURL = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
    await authenticatedAPI().exportDiagnostics()
    expect(fetchMock).toHaveBeenCalledWith('/music-room/api/v1/admin/diagnostics', expect.objectContaining({ headers: expect.objectContaining({ Authorization: 'Bearer gateway-session' }) }))
    expect(JSON.parse(await createURL.mock.calls[0][0].text())).toEqual(payload)
    expect(link.href).toBe('blob:diagnostics')
    expect(link.download).toMatch(/^music-room-diagnostics-.*\.json$/)
    expect(link.click).toHaveBeenCalledOnce()
    expect(link.remove).toHaveBeenCalledOnce()
    await vi.runAllTimersAsync()
    expect(revokeURL).toHaveBeenCalledWith('blob:diagnostics')
  })

  it('refuses an unredacted diagnostics response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(json({ redacted: false })))
    await expect(authenticatedAPI().exportDiagnostics()).rejects.toMatchObject({ code: 'diagnostics_invalid' })
  })
})
