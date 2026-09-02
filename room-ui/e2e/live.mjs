#!/usr/bin/env node

import assert from 'node:assert/strict'
import { createHash, randomBytes } from 'node:crypto'
import { chromium } from 'playwright-core'

const baseURL = (process.env.NMR_BASE_URL || 'http://127.0.0.1:1970').replace(/\/$/, '')
const gatewayPrefix = (process.env.NMR_GATEWAY_PREFIX || '/music-room').replace(/\/$/, '')
const adminUsername = process.env.NMR_ADMIN_USERNAME || 'admin'
const adminPassword = required('NMR_ADMIN_PASSWORD')
const chromiumPath = process.env.NMR_CHROMIUM_PATH || '/usr/bin/chromium-browser'
const aclLibraryPath = process.env.NMR_ACL_LIBRARY_PATH || '/plugins/navidrome-music-room/room-data/acl-test'
const suffix = `${Date.now().toString(36)}${randomBytes(3).toString('hex')}`
const member = { username: `nmr_web_${suffix}`, password: randomBytes(18).toString('base64url') }
const isolated = { username: `nmr_acl_${suffix}`, password: randomBytes(18).toString('base64url') }
const artifacts = {
  desktop: process.env.NMR_DESKTOP_SCREENSHOT || '/tmp/navidrome-music-room-web-desktop.png',
  catalog: process.env.NMR_CATALOG_SCREENSHOT || '/tmp/navidrome-music-room-web-catalog.png',
  mobile: process.env.NMR_MOBILE_SCREENSHOT || '/tmp/navidrome-music-room-web-mobile.png',
}

function required(name) {
  const value = process.env[name]?.trim()
  if (!value) throw new Error(`${name} is required`)
  return value
}

function proof(account) {
  const salt = randomBytes(16).toString('hex')
  return {
    username: account.username,
    salt,
    token: createHash('md5').update(account.password + salt).digest('hex'),
  }
}

async function jsonResponse(response, label, expected = [200]) {
  const text = await response.text()
  let body = null
  try { body = text ? JSON.parse(text) : null } catch { body = text }
  assert.ok(expected.includes(response.status), `${label} returned ${response.status}: ${String(text).slice(0, 300)}`)
  return body
}

async function login(username, password) {
  return jsonResponse(await fetch(`${baseURL}/auth/login`, {
    method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify({ username, password }),
  }), 'Navidrome login')
}

async function navidrome(path, token, { method = 'GET', body, expected = [200] } = {}) {
  return jsonResponse(await fetch(`${baseURL}/api${path}`, {
    method,
    headers: {
      accept: 'application/json',
      'x-nd-authorization': `Bearer ${token}`,
      ...(body === undefined ? {} : { 'content-type': 'application/json' }),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  }), `${method} /api${path}`, expected)
}

async function roomAPI(path, { method = 'GET', token = '', body, expected = [200] } = {}) {
  return jsonResponse(await fetch(`${baseURL}${gatewayPrefix}/api/v1${path}`, {
    method,
    headers: {
      accept: 'application/json',
      ...(token ? { authorization: `Bearer ${token}` } : {}),
      ...(body === undefined ? {} : { 'content-type': 'application/json', 'idempotency-key': randomBytes(16).toString('hex') }),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  }), `${method} room API ${path}`, expected)
}

async function exchange(account, expected = [200]) {
  return roomAPI('/auth/exchange', { method: 'POST', body: proof(account), expected })
}

function subsonicURL(method, auth, parameters = {}) {
  const url = new URL(`${baseURL}/rest/${method}.view`)
  const values = { u: auth.username, s: auth.salt, t: auth.token, v: '1.16.1', c: 'NMR-Web-E2E', f: 'json', ...parameters }
  for (const [key, value] of Object.entries(values)) url.searchParams.set(key, String(value))
  return url
}

async function subsonic(method, auth, parameters = {}) {
  const response = await fetch(subsonicURL(method, auth, parameters))
  const payload = await jsonResponse(response, `OpenSubsonic ${method}`)
  const envelope = payload?.['subsonic-response']
  assert.equal(envelope?.status, 'ok', `${method} failed: ${JSON.stringify(envelope?.error)}`)
  return envelope
}

async function waitForExchange(account, timeout = 45_000) {
  const started = Date.now()
  let last
  while (Date.now() - started < timeout) {
    try { return await exchange(account) } catch (error) { last = error }
    await new Promise((resolve) => setTimeout(resolve, 10_000))
  }
  throw last || new Error(`plugin did not authorize ${account.username}`)
}

async function loginRoomPage(page, url, account) {
  await page.goto(url, { waitUntil: 'domcontentloaded' })
  const loginForm = page.locator('input[autocomplete="username"]')
  if (await loginForm.isVisible().catch(() => false)) {
    await loginForm.fill(account.username)
    await page.locator('input[autocomplete="current-password"]').fill(account.password)
    await page.getByRole('button', { name: '登录并加入' }).click()
  }
}

async function waitForRoom(page) {
  await page.locator('.room-shell[data-connected="true"]').waitFor({ state: 'visible', timeout: 30_000 })
  await page.locator('[data-testid="current-track"]:visible').waitFor({ state: 'visible' })
}

async function audioState(page) {
  return page.evaluate(() => {
    const audio = document.querySelector('audio[data-music-room-audio="true"]')
    return audio ? { currentTime: audio.currentTime, duration: audio.duration, paused: audio.paused, src: audio.currentSrc || audio.src } : null
  })
}

async function anonymizeDocumentationCapture(page) {
  const artwork = `data:image/svg+xml,${encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 800"><defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop stop-color="#252b3a"/><stop offset="1" stop-color="#11131a"/></linearGradient><radialGradient id="r"><stop stop-color="#63dfd1" stop-opacity=".52"/><stop offset="1" stop-color="#ff647a" stop-opacity=".04"/></radialGradient></defs><rect width="800" height="800" rx="44" fill="url(#g)"/><circle cx="400" cy="400" r="285" fill="url(#r)" stroke="#63dfd1" stroke-opacity=".3" stroke-width="3"/><circle cx="400" cy="400" r="112" fill="#0d0e13" stroke="#ff647a" stroke-width="18"/><circle cx="400" cy="400" r="20" fill="#f6f7fb"/><path d="M530 230v290c0 54-45 98-101 98-45 0-81-29-81-66s36-66 81-66c17 0 33 4 46 12V279l-203 43v250c0 54-45 98-101 98-45 0-81-29-81-66s36-66 81-66c17 0 33 4 46 12V274z" fill="#f6f7fb" fill-opacity=".88"/></svg>')}`
  await page.locator('.hero-artwork:visible').evaluate((image, source) => {
    image.src = source
    image.alt = 'Music Room sample artwork'
  }, artwork)
  await page.locator('.room-identity h1').evaluate((node) => { node.textContent = 'Navidrome 一起听歌' })
  await page.locator('[data-testid="current-track"]:visible h2').evaluate((node) => { node.textContent = 'Example Track' })
  await page.locator('[data-testid="current-track"]:visible p').evaluate((node) => { node.textContent = 'Example Artist · Example Album' })
}

async function anonymizeCatalogCapture(page) {
  await page.locator('.cover-card').evaluateAll((cards) => {
    cards.slice(0, 24).forEach((card, index) => {
      const title = card.querySelector('strong')
      const artist = card.querySelector('.cover-card-open > span:last-child')
      const image = card.querySelector('img')
      if (title) title.textContent = `Example Album ${index + 1}`
      if (artist) artist.textContent = 'Example Artist'
      if (image) image.style.filter = 'saturate(.25) brightness(.7)'
    })
  })
  await page.locator('.artist-card strong').evaluateAll((artists) => {
    artists.forEach((artist, index) => { artist.textContent = `Example Artist ${index + 1}` })
  })
}

async function anonymizeQueueCapture(page) {
  await page.locator('.track-row').evaluateAll((rows) => {
    rows.forEach((row, index) => {
      const title = row.querySelector('strong')
      const detail = row.querySelector('.track-main > span')
      const image = row.querySelector('img')
      if (title) title.textContent = `Example Track ${index + 1}`
      if (detail) detail.textContent = 'Example Artist · Example Album'
      if (image) image.style.filter = 'saturate(.25) brightness(.7)'
    })
  })
  await page.locator('.notice-toast').evaluateAll((notices) => {
    notices.forEach((notice) => { notice.textContent = '✓ 已点播整张专辑（13 首）' })
  })
}

let adminToken = ''
let libraryID = 0
let aclLibraryID = 0
let room = null
let browser = null
const createdUsers = []
const directMediaResponses = []
const gatewayMediaResponses = []

try {
  const adminLogin = await login(adminUsername, adminPassword)
  adminToken = adminLogin.token
  assert.ok(adminToken, 'Navidrome login did not return a UI token')

  const libraries = await navidrome('/library', adminToken)
  const primary = libraries.find((item) => item.totalSongs > 0) || libraries[0]
  assert.ok(primary?.id, 'Navidrome has no scanned music library')
  libraryID = Number(primary.id)

  const aclLibrary = await navidrome('/library', adminToken, {
    method: 'POST',
    body: { name: `NMR ACL ${suffix}`, path: aclLibraryPath, defaultNewUsers: false },
  })
  aclLibraryID = Number(aclLibrary.id)
  assert.ok(aclLibraryID && aclLibraryID !== libraryID, 'failed to create an isolated test library')

  for (const [account, assignedLibrary] of [[member, libraryID], [isolated, aclLibraryID]]) {
    const created = await navidrome('/user', adminToken, {
      method: 'POST', body: { userName: account.username, name: account.username, email: '', password: account.password, isAdmin: false },
    })
    createdUsers.push(created.id)
    await navidrome(`/user/${created.id}/library`, adminToken, { method: 'PUT', body: { libraryIds: [assignedLibrary] } })
  }

  // The .ndp publishes its authorized-user lease every 30 seconds. Waiting for
  // one complete cycle avoids turning authorization polling into auth traffic.
  await new Promise((resolve) => setTimeout(resolve, 32_000))

  const adminAuth = proof({ username: adminUsername, password: adminPassword })
  const folders = (await subsonic('getMusicFolders', adminAuth)).musicFolders?.musicFolder || []
  assert.ok(folders.some((folder) => Number(folder.id) === libraryID), 'admin proof cannot see the primary library')
  const search = await subsonic('search3', adminAuth, { query: '*', artistCount: 0, albumCount: 0, songCount: 200, musicFolderId: libraryID })
  const songs = search.searchResult3?.song || []
  const song = songs.find((item) => item.contentType === 'audio/mpeg' && Number(item.duration) > 20) || songs.find((item) => Number(item.duration) > 20)
  assert.ok(song?.id, 'the primary library has no browser-playable test song')

  const adminSession = await exchange({ username: adminUsername, password: adminPassword })
  const memberSession = await waitForExchange(member)
  const isolatedSession = await waitForExchange(isolated)
  assert.ok(memberSession.user.musicFolderIDs.includes(libraryID), 'allowed member did not receive the primary ACL')
  assert.ok(!isolatedSession.user.musicFolderIDs.includes(libraryID), 'isolated member unexpectedly received the primary ACL')

  room = await roomAPI('/rooms', {
    method: 'POST', token: adminSession.sessionToken, expected: [201],
    body: { name: `Web 1.0 双客户端 ${suffix}`, musicFolderIDs: [libraryID], queueLimit: 20, playbackMode: 'fifo', preloadNextTrack: true },
  })
  await roomAPI(`/rooms/${room.roomID}/queue/tracks`, {
    method: 'POST', token: adminSession.sessionToken, expected: [201], body: { track: { id: song.id, musicFolderID: libraryID } },
  })
  const invite = await roomAPI(`/rooms/${room.roomID}/invites`, {
    method: 'POST', token: adminSession.sessionToken, expected: [201], body: { label: 'Web 1.0 浏览器验收', maxUses: 10, singleUse: false },
  })
  const browserBaseURL = (process.env.NMR_BROWSER_BASE_URL || new URL(invite.shareURL).origin).replace(/\/$/, '')

  browser = await chromium.launch({
    executablePath: chromiumPath,
    headless: true,
    args: ['--no-sandbox', '--disable-dev-shm-usage', '--autoplay-policy=no-user-gesture-required'],
  })
  const adminContext = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  const memberContext = await browser.newContext({ viewport: { width: 1280, height: 900 } })
  const adminPage = await adminContext.newPage()
  const memberPage = await memberContext.newPage()
  for (const page of [adminPage, memberPage]) {
    page.on('response', (response) => {
      const path = new URL(response.url()).pathname
      if (path.includes('/rest/stream.view')) directMediaResponses.push({ status: response.status(), path })
      if (path.startsWith(`${gatewayPrefix}/`) && path.includes('stream')) gatewayMediaResponses.push({ status: response.status(), path })
    })
  }

  await Promise.all([
    loginRoomPage(adminPage, `${browserBaseURL}${gatewayPrefix}/join/${room.roomID}/`, { username: adminUsername, password: adminPassword }),
    loginRoomPage(memberPage, invite.shareURL, member),
  ])
  await Promise.all([waitForRoom(adminPage), waitForRoom(memberPage)])
  assert.equal(new URL(memberPage.url()).hash, '', 'successful invite was not removed from the address bar')
  assert.match(await adminPage.locator('[data-testid="current-track"]:visible').innerText(), new RegExp(song.title.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))

  await memberPage.locator('[data-testid="listen-button"]:visible').click()
  await adminPage.locator('[data-testid="listen-button"]:visible').click()
  await adminPage.waitForFunction(() => {
    const audio = document.querySelector('audio[data-music-room-audio="true"]')
    return audio && !audio.paused && audio.currentTime > 0.4
  }, null, { timeout: 30_000 })
  await memberPage.waitForFunction(() => {
    const audio = document.querySelector('audio[data-music-room-audio="true"]')
    return audio && !audio.paused && audio.currentTime > 0.4
  }, null, { timeout: 30_000 })

  const [adminAudio, memberAudio] = await Promise.all([audioState(adminPage), audioState(memberPage)])
  assert.ok(adminAudio?.src && memberAudio?.src, 'one browser did not load the Navidrome stream')
  assert.ok(Math.abs(adminAudio.currentTime - memberAudio.currentTime) < 2.25, `browser playback drift is too large: ${adminAudio.currentTime}/${memberAudio.currentTime}`)
  const range = await adminPage.evaluate(async () => {
    const audio = document.querySelector('audio[data-music-room-audio="true"]')
    const response = await fetch(audio.src, { headers: { Range: 'bytes=0-255' } })
    return { status: response.status, bytes: (await response.arrayBuffer()).byteLength, contentRange: response.headers.get('content-range') }
  })
  assert.equal(range.status, 206, 'browser Range request was not served as partial content')
  assert.equal(range.bytes, 256, 'browser Range request returned the wrong byte count')

  const seekTarget = Math.min(9, Math.max(2, Number(song.duration) - 2))
  await adminPage.locator('.timeline input[type="range"]:visible').evaluate((input, value) => {
    input.value = String(value)
    input.dispatchEvent(new Event('change', { bubbles: true }))
  }, seekTarget)
  await memberPage.waitForFunction((target) => {
    const audio = document.querySelector('audio[data-music-room-audio="true"]')
    return audio && Math.abs(audio.currentTime - target) < 2.75
  }, seekTarget, { timeout: 10_000 })

  // Exercise the real Navidrome library paths that back the request desk. This
  // prevents an empty Songs/Albums view from hiding behind the lower-level API
  // search used to seed the playback test above.
  await adminPage.locator('.desktop-tabs button').filter({ hasText: '点歌台' }).click()
  await adminPage.locator('[data-testid="catalog-song-list"]').waitFor({ state: 'visible', timeout: 20_000 })
  const songRows = await adminPage.locator('[data-testid="catalog-song-list"] .track-row').count()
  assert.ok(songRows > 0, 'the Songs catalog is empty')

  await adminPage.locator('[data-testid="catalog-albums"]').click()
  await adminPage.locator('[data-testid="catalog-album-grid"] .cover-card').first().waitFor({ state: 'visible' })
  const albumCards = await adminPage.locator('[data-testid="catalog-album-grid"] .cover-card').count()
  assert.ok(albumCards > 0, 'the Albums catalog is empty')
  await anonymizeDocumentationCapture(adminPage)
  await anonymizeCatalogCapture(adminPage)
  await adminPage.screenshot({ path: artifacts.catalog })

  await adminPage.locator('[data-testid="catalog-album-grid"] .cover-card-open').first().click()
  await adminPage.locator('[data-testid="queue-whole-album"]').waitFor({ state: 'visible' })
  const detailRows = await adminPage.locator('[data-testid="catalog-song-list"] .track-row').count()
  assert.ok(detailRows > 0, 'an album detail contains no songs')
  const queueBeforeAlbum = (await roomAPI(`/rooms/${room.roomID}/snapshot`, { token: adminSession.sessionToken })).queue.length
  await adminPage.locator('[data-testid="queue-whole-album"]').click()
  await adminPage.waitForFunction(() => !document.querySelector('[data-testid="queue-whole-album"]')?.hasAttribute('disabled'), null, { timeout: 30_000 })
  const queueAfterAlbum = (await roomAPI(`/rooms/${room.roomID}/snapshot`, { token: adminSession.sessionToken })).queue.length
  assert.ok(queueAfterAlbum > queueBeforeAlbum, 'Request whole album did not append any tracks')

  await adminPage.getByRole('button', { name: '返回结果' }).click()
  await adminPage.locator('[data-testid="catalog-artists"]').click()
  await adminPage.locator('[data-testid="catalog-artist-grid"] .artist-card').first().waitFor({ state: 'visible' })
  const artistCards = await adminPage.locator('[data-testid="catalog-artist-grid"] .artist-card').count()
  assert.ok(artistCards > 0, 'the Artists catalog is empty')
  await adminPage.locator('[data-testid="catalog-artist-grid"] .artist-card').first().click()
  await adminPage.locator('[data-testid="catalog-album-grid"] .cover-card').first().waitFor({ state: 'visible' })

  await adminPage.locator('.desktop-tabs button').filter({ hasText: '待播放' }).click()
  await anonymizeDocumentationCapture(adminPage)
  await anonymizeQueueCapture(adminPage)
  await adminPage.screenshot({ path: artifacts.desktop })
  const beforeReloadRevision = await memberPage.locator('.room-shell').getAttribute('data-revision')
  await memberPage.reload({ waitUntil: 'domcontentloaded' })
  await waitForRoom(memberPage)
  assert.equal(new URL(memberPage.url()).hash, '', 'refresh unexpectedly restored an invite fragment')
  const afterReloadRevision = await memberPage.locator('.room-shell').getAttribute('data-revision')
  assert.ok(Number(afterReloadRevision) >= Number(beforeReloadRevision), 'refresh restored an older playback revision')
  await memberPage.locator('[data-testid="listen-button"]:visible').click()
  await memberPage.waitForFunction(() => {
    const audio = document.querySelector('audio[data-music-room-audio="true"]')
    return audio && !audio.paused && audio.currentTime > 0
  }, null, { timeout: 20_000 })

  const mobileContext = await browser.newContext({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 1 })
  const mobilePage = await mobileContext.newPage()
  await loginRoomPage(mobilePage, `${browserBaseURL}${gatewayPrefix}/join/${room.roomID}/`, member)
  await waitForRoom(mobilePage)
  await mobilePage.locator('.mobile-tabs').waitFor({ state: 'visible' })
  await mobilePage.locator('.mobile-tabs button').filter({ hasText: '点歌台' }).click()
  await mobilePage.locator('[data-testid="catalog-song-list"]').waitFor({ state: 'visible', timeout: 20_000 })
  await mobilePage.locator('[data-testid="catalog-albums"]').click()
  await mobilePage.locator('[data-testid="catalog-album-grid"] .cover-card').first().waitFor({ state: 'visible' })
  const mobileMetrics = await mobilePage.evaluate(() => ({ width: innerWidth, scrollWidth: document.documentElement.scrollWidth }))
  assert.ok(mobileMetrics.scrollWidth <= mobileMetrics.width + 1, `mobile layout overflows horizontally: ${JSON.stringify(mobileMetrics)}`)
  await mobilePage.locator('.room-identity h1').evaluate((node) => { node.textContent = 'Navidrome 一起听歌' })
  await anonymizeCatalogCapture(mobilePage)
  await mobilePage.screenshot({ path: artifacts.mobile })

  const aclContext = await browser.newContext({ viewport: { width: 390, height: 844 } })
  const aclPage = await aclContext.newPage()
  await loginRoomPage(aclPage, invite.shareURL, isolated)
  await aclPage.getByText('暂时无法进入这个房间').waitFor({ timeout: 20_000 })
  assert.match(await aclPage.locator('.error-state').innerText(), /library_access_required|music library|音乐库/i)
  assert.equal(await aclPage.locator('audio[data-music-room-audio="true"]').count(), 0, 'ACL-denied browser created a media player')

  assert.ok(directMediaResponses.some((item) => item.status === 200 || item.status === 206), 'no real browser requested audio from Navidrome')
  assert.equal(gatewayMediaResponses.length, 0, 'browser routed media through the room gateway')

  process.stdout.write(`${JSON.stringify({
    passed: true,
    roomID: room.roomID,
    song: { id: song.id, title: song.title, contentType: song.contentType },
    browsers: { independentClients: 2, refreshReconnect: true, mobileViewport: mobileMetrics, aclIsolation: true, catalog: { songRows, albumCards, artistCards, wholeAlbumQueued: queueAfterAlbum - queueBeforeAlbum } },
    audio: { directStreamResponses: directMediaResponses.length, range, driftSeconds: Math.abs(adminAudio.currentTime - memberAudio.currentTime), gatewayMediaResponses: 0 },
    screenshots: artifacts,
  }, null, 2)}\n`)
} finally {
  if (browser) await browser.close().catch(() => {})
  if (room && adminToken) {
    try {
      const adminSession = await exchange({ username: adminUsername, password: adminPassword })
      await roomAPI(`/rooms/${room.roomID}`, { method: 'DELETE', token: adminSession.sessionToken, expected: [204] })
    } catch {}
  }
  if (adminToken) {
    for (const userID of createdUsers.reverse()) {
      try { await navidrome(`/user/${userID}`, adminToken, { method: 'DELETE' }) } catch {}
    }
    if (aclLibraryID) {
      try { await navidrome(`/library/${aclLibraryID}`, adminToken, { method: 'DELETE' }) } catch {}
    }
  }
}
