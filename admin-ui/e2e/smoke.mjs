import assert from 'node:assert/strict'
import { createServer } from 'node:http'
import { readFile } from 'node:fs/promises'
import path from 'node:path'
import { chromium } from 'playwright-core'
import { locales, messages } from '../src/i18n.js'

const buildDir = path.resolve(process.env.ADMIN_UI_OUT_DIR || '/tmp/admin-ui-check')
const executablePath = process.env.CHROMIUM_PATH || '/usr/bin/chromium-browser'
const server = createServer(async (request, response) => {
  const pathname = new URL(request.url, 'http://localhost').pathname
  const relative = pathname.replace(/^\/music-room\/admin\/?/, '') || 'index.html'
  const file = path.resolve(buildDir, relative)
  try {
    assert.ok(file.startsWith(`${buildDir}/`))
    const body = await readFile(file)
    response.writeHead(200, { 'Content-Type': file.endsWith('.js') ? 'text/javascript' : 'text/html' })
    response.end(body)
  } catch {
    response.writeHead(404)
    response.end()
  }
})
await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve))
const origin = `http://127.0.0.1:${server.address().port}`
const browser = await chromium.launch({ executablePath, headless: true, args: ['--no-sandbox'] })

async function fixture({ credentials = true, admin = true, managed = true } = {}) {
  const context = await browser.newContext({ locale: 'en-US', viewport: { width: 1360, height: 1000 } })
  if (credentials) await context.addInitScript(() => {
    localStorage.setItem('username', 'admin')
    localStorage.setItem('subsonic-salt', 'abcdef')
    localStorage.setItem('subsonic-token', 'a'.repeat(32))
    localStorage.setItem('token', 'navidrome-ui-jwt')
  })
  const state = {
    version: '1.1.0', pluginVersion: '1.0.0', plugin: 'stale', sessions: 0,
    requests: [], actions: [], mutations: [], crash: false, pending: false, healthReads: 0, pairedReads: 0,
    rooms: [{ roomID: 'room-1', name: 'Test room', status: 'open', onlineCount: 1, playbackMode: 'fifo', queueLimit: 20, musicFolderIDs: [1], allowMemberSkip: false }],
  }
  const page = await context.newPage()
  const pageErrors = []
  page.on('pageerror', (error) => pageErrors.push(error.message))
  const updateStatus = () => ({
    managedByLauncher: managed, repository: 'mythezone/navidrome-music-room',
    status: { currentVersion: state.version, channel: 'stable', latestVersion: 'v1.2.0', updateAvailable: state.version !== '1.2.0', releaseNotes: 'Release notes <img src=x onerror=alert(1)>', releaseURL: 'https://github.com/mythezone/navidrome-music-room/releases/tag/v1.2.0', releasePublishedAt: '2026-11-18T12:00:00Z', rollbackAvailable: state.version === '1.2.0', rollbackVersion: 'v1.1.0' },
  })
  await page.route('**/*', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    if (!url.pathname.includes('/api/') && !url.pathname.endsWith('/readyz') && !url.pathname.includes('/rest/')) return route.continue()
    const pathname = url.pathname
    const method = request.method()
    const body = request.postDataJSON()
    state.requests.push({ pathname, method, headers: request.headers(), body })
    const send = (payload, status = 200) => route.fulfill({ status, json: payload })
    if (pathname.endsWith('/auth/exchange')) {
      state.sessions++
      return send({ sessionToken: `gateway-${state.sessions}`, user: { username: 'admin', adminRole: admin, musicFolderIDs: [1] }, navidromeBaseURL: origin, gatewayBaseURL: `${origin}/music-room` })
    }
    if (pathname.endsWith('/getMusicFolders.view')) return send({ 'subsonic-response': { status: 'ok', musicFolders: { musicFolder: [{ id: 1, name: 'Library' }] } } })
    if (pathname.endsWith('/readyz')) {
      if (state.pending) {
        state.healthReads++
        if (state.healthReads === 1) return send({ error: { code: 'database_unavailable' } }, 503)
        if (state.plugin === 'paired' && state.pairedReads++ === 0) return send({ status: 'ready', version: state.version, pluginVersion: state.pluginVersion, plugin: 'stale' })
      }
      return send({ status: 'ready', version: state.version, pluginVersion: state.pluginVersion, plugin: state.plugin })
    }
    if (pathname.startsWith('/api/plugin')) {
      assert.equal(request.headers()['x-nd-authorization'], 'Bearer navidrome-ui-jwt')
      assert.equal(request.headers().authorization, undefined)
      if (pathname.endsWith('/rescan')) return send({})
      if (method === 'GET') return send([{ id: 'navidrome-music-room', name: 'Navidrome Music Room', enabled: false }])
      assert.deepEqual(body, { enabled: true })
      state.pluginVersion = state.version
      state.plugin = 'paired'
      return send({ enabled: true })
    }
    assert.equal(request.headers()['x-nd-authorization'], undefined)
    assert.equal(request.headers().authorization, `Bearer gateway-${state.sessions}`)
    if (pathname.endsWith('/admin/updates')) {
      if (method === 'GET') return send(updateStatus())
      state.actions.push(body)
      if (body.action === 'check') return send({ status: updateStatus().status })
      state.version = body.action === 'rollback' ? '1.1.0' : '1.2.0'
      state.plugin = 'stale'
      state.pending = true
      state.healthReads = 0
      state.pairedReads = 0
      return send({ version: `v${state.version}`, activationScheduled: body.action === 'install' }, 202)
    }
    if (pathname.endsWith('/admin/diagnostics')) return send({ redacted: true, format: 'navidrome-music-room-diagnostics/v1' })
    if (pathname.endsWith('/rooms') && method === 'GET') return send({ rooms: state.crash ? [null] : state.rooms })
    if (pathname.endsWith('/rooms') && method === 'POST') {
      state.mutations.push(body)
      const room = { ...body, roomID: `room-${state.rooms.length + 1}`, status: 'open', onlineCount: 0 }
      state.rooms.push(room)
      return send(room)
    }
    if (/\/rooms\/[^/]+$/.test(pathname) && method === 'PATCH') {
      state.mutations.push(body)
      const room = state.rooms.find((entry) => pathname.endsWith(`/${entry.roomID}`))
      Object.assign(room, body)
      return send(room)
    }
    if (pathname.endsWith('/members')) return send({ members: [{ username: 'admin', role: 'owner', online: true, active: true }] })
    if (pathname.endsWith('/invites') && method === 'GET') return send({ invites: [] })
    if (pathname.endsWith('/invites') && method === 'POST') return send({ inviteID: 'invite-1', shareURL: `${origin}/music-room/join/room-1/#invite=test-secret`, deepLink: 'musicmate://join?invite=test-secret' })
    throw new Error(`Unexpected request: ${method} ${pathname}`)
  })
  return { page, state, pageErrors, context }
}

try {
  const missing = await fixture({ credentials: false })
  await missing.page.goto(`${origin}/music-room/admin/`)
  await missing.page.getByText(messages.en.loginTitle, { exact: true }).waitFor()
  for (const locale of Object.keys(locales)) {
    await missing.page.locator('select').selectOption(locale)
    await missing.page.getByText(messages[locale].loginTitle, { exact: true }).waitFor()
    assert.equal(await missing.page.title(), messages[locale].pageTitle)
    assert.equal(await missing.page.locator('html').getAttribute('lang'), locale)
    assert.equal(await missing.page.evaluate(() => localStorage.getItem('nmr:locale')), locale)
  }
  await missing.page.reload()
  await missing.page.getByText(messages.de.loginTitle, { exact: true }).waitFor()
  assert.deepEqual(missing.pageErrors, [])
  await missing.context.close()

  const { page, state, pageErrors, context } = await fixture()
  await page.goto(`${origin}/music-room/admin/`)
  await page.getByText('Updates & System Health', { exact: true }).waitFor()
  await page.getByText('Gateway and plugin versions differ', { exact: true }).waitFor()
  await page.getByRole('button', { name: messages.en.rescanPlugin, exact: true }).click()
  await page.getByText(messages.en.pluginActivated, { exact: true }).waitFor()
  assert.equal(state.sessions, 2)

  await page.getByRole('button', { name: 'Create', exact: true }).click()
  let dialog = page.getByRole('dialog')
  assert.equal(await dialog.getByRole('checkbox', { name: messages.en.allowMemberSkip }).isChecked(), true)
  await dialog.locator('#room-name').fill('Created room')
  await dialog.getByRole('button', { name: 'Create room', exact: true }).click()
  await dialog.waitFor({ state: 'hidden' })
  assert.equal(state.mutations.at(-1).allowMemberSkip, true)
  await page.getByRole('button', { name: 'Room settings', exact: true }).click()
  dialog = page.getByRole('dialog')
  await dialog.getByRole('checkbox', { name: messages.en.allowMemberSkip }).uncheck()
  await dialog.getByRole('button', { name: 'Save', exact: true }).click()
  await dialog.waitFor({ state: 'hidden' })
  assert.equal(state.mutations.at(-1).allowMemberSkip, false)
  await page.getByText('Managers only', { exact: true }).waitFor()
  await page.getByRole('button', { name: 'Room settings', exact: true }).click()
  assert.equal(await page.getByRole('dialog').getByRole('checkbox', { name: messages.en.allowMemberSkip }).isChecked(), false)
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel', exact: true }).click()
  await page.getByRole('dialog').waitFor({ state: 'hidden' })

  await page.getByRole('button', { name: messages.en.share, exact: true }).click()
  await page.getByRole('button', { name: messages.en.generateInvite, exact: true }).click()
  await page.getByText('Invitation created', { exact: true }).waitFor()
  assert.equal(await page.getByRole('dialog').locator('svg title').textContent(), messages.en.qrTitle)
  await page.getByRole('button', { name: 'Done', exact: true }).click()
  await page.getByRole('dialog').waitFor({ state: 'hidden' })

  await page.getByRole('button', { name: 'Release notes', exact: true }).click()
  await page.getByText('Release notes <img src=x onerror=alert(1)>', { exact: true }).waitFor()
  assert.equal(await page.getByRole('dialog').locator('img').count(), 0)
  await page.getByRole('dialog').getByRole('button', { name: 'Close', exact: true }).click()
  await page.getByRole('dialog').waitFor({ state: 'hidden' })
  await page.getByRole('button', { name: 'Check for updates', exact: true }).click()
  await page.waitForFunction(() => !document.querySelector('main [role="progressbar"]'))
  assert.deepEqual(state.actions, [{ action: 'check' }])

  await page.getByRole('button', { name: 'Upgrade now', exact: true }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel', exact: true }).click()
  await page.getByRole('dialog').waitFor({ state: 'hidden' })
  assert.equal(state.actions.length, 1)
  await page.getByRole('button', { name: 'Upgrade now', exact: true }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Upgrade now', exact: true }).click()
  await page.getByText('Gateway and plugin are ready on 1.2.0.', { exact: true }).waitFor({ timeout: 20000 })
  assert.equal(state.sessions, 3)
  assert.deepEqual(state.actions.at(-1), { action: 'install', version: 'v1.2.0' })
  await page.getByRole('dialog').getByRole('button', { name: 'Done', exact: true }).click()
  await page.getByRole('dialog').waitFor({ state: 'hidden' })

  const downloadPromise = page.waitForEvent('download')
  await page.getByRole('button', { name: 'Download diagnostics', exact: true }).click()
  const download = await downloadPromise
  assert.match(download.suggestedFilename(), /^music-room-diagnostics-.*\.json$/)
  assert.equal(JSON.parse(await readFile(await download.path(), 'utf8')).redacted, true)

  await page.getByRole('button', { name: 'Rollback', exact: true }).click()
  await page.getByText(/Room changes since that backup will be lost/).waitFor()
  await page.getByRole('dialog').getByRole('button', { name: 'Rollback', exact: true }).click()
  await page.getByText('Gateway and plugin are ready on 1.1.0.', { exact: true }).waitFor({ timeout: 20000 })
  assert.equal(state.sessions, 4)
  assert.deepEqual(state.actions.at(-1), { action: 'rollback' })
  await page.getByRole('dialog').getByRole('button', { name: 'Done', exact: true }).click()
  await page.getByRole('dialog').waitFor({ state: 'hidden' })

  await page.screenshot({ path: path.join(buildDir, 'desktop.png'), fullPage: true })
  await page.setViewportSize({ width: 390, height: 844 })
  for (const locale of Object.keys(locales)) {
    await page.locator('select').selectOption(locale)
    await page.getByText(messages[locale].updates, { exact: true }).waitFor()
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `${locale}: mobile overflow`)
  }
  await page.screenshot({ path: path.join(buildDir, 'mobile.png'), fullPage: true })
  assert.deepEqual(pageErrors, [])
  await page.locator('select').selectOption('en')
  state.crash = true
  await page.getByRole('button', { name: 'Refresh', exact: true }).click()
  await page.getByRole('heading', { name: messages.en.recoveryTitle, exact: true }).waitFor()
  assert.equal(await page.getByRole('link', { name: messages.en.back }).getAttribute('href'), '/app/')
  state.crash = false
  await page.getByRole('button', { name: 'Reload', exact: true }).click()
  await page.getByText('Updates & System Health', { exact: true }).waitFor()
  await context.close()

  const manual = await fixture({ managed: false })
  await manual.page.goto(`${origin}/music-room/admin/`)
  await manual.page.getByText('docker compose pull && docker compose up -d', { exact: true }).waitFor()
  assert.equal(await manual.page.getByRole('button', { name: 'Upgrade now', exact: true }).count(), 0)
  assert.match(await manual.page.getByRole('link', { name: 'View GitHub release' }).first().getAttribute('href'), /github.com\/mythezone\/navidrome-music-room\/releases/)
  assert.deepEqual(manual.pageErrors, [])
  await manual.context.close()

  const member = await fixture({ admin: false })
  await member.page.goto(`${origin}/music-room/admin/`)
  await member.page.getByText(messages.en.notAdmin, { exact: true }).waitFor()
  assert.equal(await member.page.getByText('Updates & System Health', { exact: true }).count(), 0)
  assert.equal(member.state.requests.filter(({ pathname }) => pathname.includes('/admin/updates')).length, 0)
  assert.deepEqual(member.pageErrors, [])
  await member.context.close()
  process.stdout.write('Browser smoke passed: login, 7 locales, room settings, QR, updates, rollback, diagnostics, recovery, mobile and admin access.\n')
} finally {
  await browser.close()
  await new Promise((resolve) => server.close(resolve))
}
