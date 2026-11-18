import assert from 'node:assert/strict'
import { createServer } from 'node:http'
import { readFile } from 'node:fs/promises'
import path from 'node:path'
import { chromium } from 'playwright-core'

const output = process.env.ROOM_UI_OUT_DIR || '/tmp/room-ui-check'
const roomID = '0123456789abcdef0123456789abcdef'
const csp = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; media-src 'self' blob:"
const server = createServer(async (request, response) => {
  try {
    const url = new URL(request.url, 'http://localhost')
    const asset = url.pathname.includes('/assets/') ? path.basename(url.pathname) : ''
    const file = asset ? path.join(output, 'assets', asset) : path.join(output, 'index.html')
    const content = await readFile(file)
    response.writeHead(200, { 'Content-Type': asset.endsWith('.js') ? 'application/javascript' : asset.endsWith('.css') ? 'text/css' : asset.endsWith('.woff2') ? 'font/woff2' : 'text/html', 'Content-Security-Policy': csp })
    response.end(content)
  } catch { response.writeHead(404); response.end() }
})
await new Promise(resolve => server.listen(0, '127.0.0.1', resolve))
const origin = `http://127.0.0.1:${server.address().port}`
const roomURL = `${origin}/music-room/join/${roomID}/`
const stamp = new Date().toISOString()
const song = { id: 'song-1', title: 'Midnight Signals', artist: 'The Night Ensemble', artistId: 'artist-1', album: 'After Hours', albumId: 'album-1', duration: 180 }
const album = { id: 'album-1', name: 'After Hours', artist: song.artist, artistId: song.artistId, song: [song], songCount: 1 }
const artist = { id: 'artist-1', name: song.artist, albumCount: 1, album: [album] }
const track = { id: song.id, title: song.title, artist: song.artist, album: song.album, albumID: song.albumId, musicFolderID: 7, durationSeconds: 180 }
const member = { username: 'listener', displayName: 'Listener', role: 'member', active: true, online: true, joinedAt: stamp, lastSeenAt: stamp }
const owner = { ...member, username: 'owner', displayName: 'Room Owner', role: 'owner' }
const other = { ...member, username: 'other', displayName: 'Another Listener', online: false }
const badges = [{ badgeID: 'requested_1', code: 'requested_1', kind: 'requested', tier: 1, count: 4, target: 1, subject: '', unlocked: true }]
const profile = username => ({ ...(username === 'owner' ? owner : username === 'other' ? other : member), stats: { requestedCount: 4, listenedCount: 8, chatCount: 12, staySeconds: 7200 }, level: { level: 3, code: 'level_3', effectiveMinutes: 120, currentThresholdMinutes: 100, nextThresholdMinutes: 200, progress: .2 }, badges, selectedBadgeID: '' })
const mediaStat = { id: 'song-1', name: song.title, artist: song.artist, count: 4, seconds: 720 }
const stats = { period: 'week', totalPlayCount: 4, totalPlayedSeconds: 720, totalChatCount: 12, activeMemberCount: 3, topArtists: [{ ...mediaStat, id: song.artist, name: song.artist }], topAlbums: [{ ...mediaStat, id: album.id, name: album.name }], topTracks: [mediaStat], hourlyDistribution: Array.from({ length: 24 }, (_, hour) => ({ hour, count: hour % 4, seconds: (hour % 7) * 60 })) }
const rankings = { ...stats, requested: [{ username: 'listener', displayName: 'Listener', count: 4 }], listened: [{ username: 'owner', displayName: 'Room Owner', count: 8 }], chat: [{ username: 'other', displayName: 'Another Listener', count: 12 }] }
const stickerPacks = ['vibes', 'reactions'].map((packID, index) => ({ packID, stickers: [{ packID, stickerID: `${packID}:music`, emoji: index ? '❤️' : '🎵', code: 'music', labels: { en: index ? 'Love' : 'Music', 'zh-CN': index ? '喜欢' : '音乐', de: index ? 'Liebe' : 'Musik' } }] }))
const calls = []
const faults = []
const browser = await chromium.launch({ executablePath: process.env.NMR_CHROMIUM_PATH || '/usr/bin/chromium-browser', headless: true, args: ['--no-sandbox'] })

async function setup({ mobile = false, manager = false, login = false } = {}) {
  const context = await browser.newContext({ viewport: mobile ? { width: 390, height: 844 } : { width: 1440, height: 1100 }, isMobile: mobile, hasTouch: mobile, locale: 'en-US', ...(mobile ? { userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148' } : {}) })
  const page = await context.newPage()
  const self = manager ? owner : member
  const room = { roomID, name: 'After Hours — Listening Room', ownerUsername: owner.username, ownerDisplayName: owner.displayName, status: 'open', queueLimit: 20, playbackMode: 'fifo', musicFolderIDs: [7], allowMemberSkip: true, preloadNextTrack: false, onlineCount: 2, capabilities: { chat: true, stickers: true, statistics: true, rankings: true, achievements: true, invites: true, queue: true, history: true }, createdAt: stamp, updatedAt: stamp }
  let playback = { currentTrack: track, contributorUsername: 'other', contributorDisplayName: 'Another Listener', revision: 1, positionSeconds: 12, status: 'paused', serverTime: stamp, pausedForEmpty: false }
  let queue = [1, 2, 3].map((n, position) => ({ queueID: `q${n}`, position, track: { ...track, id: `song-${n}`, title: n === 3 ? 'A very long song title that should wrap comfortably without causing any horizontal scrolling' : track.title }, contributorUsername: 'other', contributorDisplayName: 'Another Listener', createdAt: stamp }))
  const history = [{ historyID: 'h1', track, contributorUsername: 'owner', contributorDisplayName: 'Room Owner', startedAt: stamp, finishedAt: stamp, playedSeconds: 180 }]
  let messages = [{ messageID: 'm1', senderUsername: 'other', senderDisplayName: 'Another Listener', body: 'A perfect song for a quiet evening.', createdAt: stamp, references: [{ kind: 'track', id: song.id, title: song.title, artist: song.artist }], mentions: [] }]
  let wishes = [{ wishID: 'w1', roomID, authorUsername: 'listener', authorDisplayName: 'Listener', kind: 'track', title: song.title, artist: song.artist, reason: 'For our next listening session', status: 'pending', createdAt: stamp, updatedAt: stamp, resources: [], responseNote: '' }]
  let favorites = { song: [song], album: [album], artist: [artist] }
  page.on('pageerror', error => faults.push(error.message))
  await page.addInitScript(({ username, login }) => {
    window.__cspErrors = []
    document.addEventListener('securitypolicyviolation', event => window.__cspErrors.push(`${event.violatedDirective}: ${event.blockedURI}`))
    localStorage.setItem('nmr:locale', 'en')
    if (!login) { localStorage.setItem('username', username); localStorage.setItem('subsonic-salt', 'abcdef'); localStorage.setItem('subsonic-token', '0123456789abcdef0123456789abcdef') }
    window.WebSocket = class extends EventTarget {
      static OPEN = 1
      constructor() { super(); this.readyState = 1; window.__roomSocket = this; setTimeout(() => this.dispatchEvent(new Event('open')), 20) }
      close() { this.readyState = 3 }
    }
  }, { username: self.username, login })
  await page.route('**/api/v1/**', async route => {
    const request = route.request()
    const url = new URL(request.url())
    const suffix = url.pathname.split('/api/v1')[1]
    const method = request.method()
    const body = request.postDataJSON()
    calls.push({ suffix, method, body })
    const ok = data => route.fulfill({ json: data })
    const empty = () => route.fulfill({ status: 204, body: '' })
    if (suffix === '/auth/exchange') return ok({ sessionToken: 'test', expiresAt: new Date(Date.now() + 3600000).toISOString(), user: { ...self, adminRole: manager, musicFolderIDs: [7] }, navidromeBaseURL: origin, gatewayBaseURL: origin + '/music-room' })
    if (suffix.endsWith('/join')) return ok(room)
    if (suffix.endsWith('/ws-ticket')) return ok({ webSocketURL: origin.replace('http:', 'ws:') + '/socket', expiresAt: stamp })
    if (suffix.endsWith('/snapshot')) return ok({ room, self, members: [owner, member, other], playback, queue, history, generatedAt: stamp, permissions: { canManage: manager, canControl: manager, canSkip: room.allowMemberSkip || manager, canReorderQueue: manager, canFulfillWish: manager }, updateNotice: manager ? { currentVersion: '1.2.0', latestVersion: '1.2.1', channel: 'stable', releaseURL: 'https://example.test/release' } : undefined })
    if (suffix === `/rooms/${roomID}` && method === 'PATCH') { Object.assign(room, body); return ok(room) }
    if (suffix.endsWith('/playback')) { playback = { ...playback, revision: playback.revision + 1, positionSeconds: body.positionSeconds ?? 12, status: body.command === 'play' ? 'playing' : 'paused', serverTime: new Date().toISOString() }; return ok(playback) }
    if (suffix.endsWith('/queue/order')) { queue = body.queueIDs.map(id => queue.find(item => item.queueID === id)); return ok({ queue }) }
    if (suffix.endsWith('/queue/shuffle')) { queue = [queue[0], ...queue.slice(1).reverse()]; return ok({ queue }) }
    if (suffix.endsWith('/top')) { const id = suffix.split('/').at(-2); queue = [...queue.filter(item => item.queueID === id), ...queue.filter(item => item.queueID !== id)]; return ok({ queue }) }
    if (suffix.endsWith('/queue/tracks') || suffix.endsWith('/requeue')) { const entry = { queueID: `new-${queue.length}`, position: queue.length, track: body.track || track, contributorUsername: self.username, contributorDisplayName: self.displayName, createdAt: stamp }; queue = [...queue, entry]; return ok({ entry, queue, playback }) }
    if (suffix.includes('/queue/') && method === 'DELETE') { queue = queue.filter(item => item.queueID !== suffix.split('/').at(-1)); return empty() }
    if (suffix.endsWith('/history')) return ok({ items: [] })
    if (suffix.endsWith('/stickers')) return ok({ packs: stickerPacks })
    if (suffix.endsWith('/chat') && method === 'GET') return ok({ messages: url.searchParams.has('before') ? [] : messages, hasMore: !url.searchParams.has('before'), nextBefore: 'm1' })
    if (suffix.endsWith('/chat') && method === 'POST') { assert.ok('body' in body && !('text' in body)); const message = { ...body, messageID: `m${messages.length + 1}`, senderUsername: self.username, senderDisplayName: self.displayName, createdAt: new Date().toISOString() }; messages.push(message); return ok(message) }
    if (suffix.endsWith('/recall')) { const message = messages.find(item => item.messageID === suffix.split('/').at(-2)); Object.assign(message, { body: '', recalledAt: stamp }); return ok(message) }
    if (suffix.endsWith('/statistics')) return ok(stats)
    if (suffix.endsWith('/rankings')) return ok(rankings)
    if (suffix.endsWith('/achievements')) return ok(profile(self.username))
    if (suffix.endsWith('/profile')) return ok(profile(decodeURIComponent(suffix.split('/').at(-2))))
    if (suffix.endsWith('/achievements/selected')) return ok({ ...profile(self.username), selectedBadgeID: body.badgeID })
    if (suffix.endsWith('/wishes') && method === 'GET') return ok({ wishes })
    if (suffix.endsWith('/wishes') && method === 'POST') { const wish = { ...body, wishID: `w${wishes.length + 1}`, authorUsername: self.username, authorDisplayName: self.displayName, status: 'pending', createdAt: stamp, updatedAt: stamp, resources: [] }; wishes.push(wish); return ok(wish) }
    if (suffix.endsWith('/fulfill')) { assert.ok('responseNote' in body); assert.ok(body.resources.every(resource => resource.url && resource.title)); const wish = wishes.find(item => item.wishID === suffix.split('/').at(-2)); Object.assign(wish, body, { status: 'fulfilled' }); return ok(wish) }
    return route.fulfill({ status: 404, json: { error: { code: 'request_failed' } } })
  })
  await page.route('**/rest/*.view?**', async route => {
    const url = new URL(route.request().url())
    const method = path.basename(url.pathname, '.view')
    let data = {}
    if (method === 'getMusicFolders') data = { musicFolders: { musicFolder: [{ id: 7, name: 'Library' }] } }
    if (method === 'getRandomSongs') data = { randomSongs: { song: [song] } }
    if (method === 'getAlbumList2') data = { albumList2: { album: [album] } }
    if (method === 'getArtists') data = { artists: { index: [{ artist: [artist] }] } }
    if (method === 'getAlbum') data = { album }
    if (method === 'getArtist') data = { artist }
    if (method === 'getSong') data = { song: { ...song, id: url.searchParams.get('id') } }
    if (method === 'getStarred2') data = { starred2: favorites }
    if (method === 'getPlaylists') data = { playlists: { playlist: [{ id: 'p1', name: 'Night Playlist', songCount: 1 }] } }
    if (method === 'getPlaylist') data = { playlist: { id: 'p1', name: 'Night Playlist', entry: [song] } }
    if (method === 'search3') data = { searchResult3: { song: [song], album: [album], artist: [artist] } }
    if (method === 'getLyricsBySongId') data = { lyricsList: { structuredLyrics: [{ synced: true, line: Array.from({ length: 20 }, (_, i) => ({ start: i * 3000, value: `Listening beneath the stars — ${i + 1}` })) }] } }
    if (method === 'unstar') favorites = { song: [], album: [], artist: [] }
    if (method === 'stream') {
      const wav = Buffer.alloc(44 + 8000 * 2 * 180)
      wav.write('RIFF'); wav.writeUInt32LE(wav.length - 8, 4); wav.write('WAVEfmt ', 8); wav.writeUInt32LE(16, 16); wav.writeUInt16LE(1, 20); wav.writeUInt16LE(1, 22); wav.writeUInt32LE(8000, 24); wav.writeUInt32LE(16000, 28); wav.writeUInt16LE(2, 32); wav.writeUInt16LE(16, 34); wav.write('data', 36); wav.writeUInt32LE(wav.length - 44, 40)
      return route.fulfill({ contentType: 'audio/wav', body: wav })
    }
    return route.fulfill({ json: { 'subsonic-response': { status: 'ok', ...data } } })
  })
  await page.goto(roomURL)
  if (!login) await page.locator('.room-shell[data-connected="true"]').waitFor()
  return { page, context }
}
async function checkOverflow(page, label) {
  const dimensions = await page.evaluate(() => ({ width: innerWidth, scroll: document.documentElement.scrollWidth }))
  assert.ok(dimensions.scroll <= dimensions.width, `${label}: horizontal overflow ${JSON.stringify(dimensions)}`)
}
async function tab(page, name, mobile = false) { await page.locator(mobile ? '.mobile-tabs' : '.desktop-tabs').getByRole('button', { name: new RegExp(name) }).click() }
async function emit(page, type, payload) { await page.evaluate(({ type, payload }) => window.__roomSocket.dispatchEvent(new MessageEvent('message', { data: JSON.stringify({ type, roomID: '0123456789abcdef0123456789abcdef', payload }) })), { type, payload }) }

try {
  const { page: desktop } = await setup({ manager: true })
  assert.equal(await desktop.title(), 'After Hours — Listening Room · Listen together · Navidrome')
  const divider = desktop.getByRole('separator')
  await divider.focus(); await divider.press('ArrowRight'); assert.equal(await divider.getAttribute('aria-valuenow'), '42')
  await divider.press('Home'); assert.equal(await divider.getAttribute('aria-valuenow'), '28')
  await divider.press('End'); assert.equal(await divider.getAttribute('aria-valuenow'), '60')
  const bounds = await divider.boundingBox()
  await desktop.mouse.move(bounds.x + bounds.width / 2, bounds.y + 100); await desktop.mouse.down(); await desktop.mouse.move(580, bounds.y + 100); await desktop.mouse.up()
  assert.ok(Number(await desktop.evaluate(() => localStorage.getItem('nmr:split:v1'))) < 60)
  await desktop.waitForFunction(() => {
    const lyric = document.querySelector('.lyrics p.active')?.getBoundingClientRect()
    const panel = document.querySelector('.lyrics')?.getBoundingClientRect()
    return lyric && panel && Math.abs(lyric.y + lyric.height / 2 - panel.y - panel.height / 2) < 30
  })
  await checkOverflow(desktop, 'desktop')
  await desktop.screenshot({ path: path.join(output, 'desktop.png'), fullPage: true })
  const handles = desktop.getByRole('button', { name: 'Drag to reorder' })
  await handles.nth(0).focus(); await handles.nth(0).press('ArrowDown')
  await desktop.waitForFunction(() => document.querySelector('[data-queue-id]')?.getAttribute('data-queue-id') === 'q2')
  const fromHandle = await handles.nth(0).boundingBox()
  const toRow = await desktop.locator('[data-queue-id="q3"]').boundingBox()
  await desktop.mouse.move(fromHandle.x + 10, fromHandle.y + 10)
  await desktop.mouse.down()
  await desktop.mouse.move(toRow.x + 100, toRow.y + toRow.height / 2, { steps: 8 })
  await desktop.mouse.up()
  await desktop.waitForFunction(() => document.querySelector('[data-queue-id]')?.getAttribute('data-queue-id') === 'q1')
  await desktop.getByRole('button', { name: 'Room settings', exact: true }).click()
  await desktop.getByLabel('Room name', { exact: true }).fill('Renamed Room')
  await desktop.getByRole('button', { name: 'Save', exact: true }).click()
  await desktop.waitForFunction(() => document.title === 'Renamed Room · Listen together · Navidrome')
  await desktop.getByRole('button', { name: 'Members', exact: true }).click()
  await desktop.getByRole('button', { name: 'Statistics & rankings', exact: true }).click()
  await desktop.getByText('Top DJs', { exact: true }).waitFor()
  assert.equal(await desktop.locator('.activity-column').count(), 24)
  await desktop.getByRole('button', { name: 'Wishes', exact: true }).click()
  await desktop.getByRole('button', { name: 'Fulfill wish', exact: true }).click()
  await desktop.locator('.resource-choice').first().click()
  await desktop.getByLabel('Response note').fill('Available in our library')
  await desktop.locator('.fulfillment-form').getByRole('button', { name: 'Fulfill wish', exact: true }).click()
  await desktop.getByText('Available in our library', { exact: true }).waitFor()
  await desktop.getByRole('button', { name: 'Close', exact: true }).click()
  await desktop.getByRole('button', { name: 'Member profile', exact: true }).click()
  await desktop.getByText('Level 3 / 10', { exact: true }).waitFor()
  await desktop.locator('.badge-card').first().click()
  await desktop.getByRole('button', { name: 'Clear display badge' }).waitFor()
  await desktop.getByRole('button', { name: 'Close', exact: true }).click()
  await tab(desktop, 'Chat')
  await desktop.getByText('A perfect song for a quiet evening.', { exact: true }).waitFor()
  await desktop.getByRole('button', { name: 'Stickers', exact: true }).click()
  await desktop.getByRole('button', { name: 'Music', exact: true }).click()
  await desktop.getByRole('button', { name: 'Send', exact: true }).click()
  await desktop.locator('.chat-sticker').waitFor()
  await desktop.getByRole('textbox', { name: 'Chat', exact: true }).fill('@Mid')
  await desktop.getByRole('option').filter({ hasText: 'Midnight Signals' }).click()
  await desktop.getByRole('textbox', { name: 'Chat', exact: true }).fill('Listen to this!')
  await desktop.getByRole('button', { name: 'Send', exact: true }).click()
  await desktop.getByText('Listen to this!', { exact: true }).waitFor()
  await desktop.getByRole('button', { name: 'Recall (within 5 min)', exact: true }).last().click()
  await desktop.getByText('Message recalled', { exact: true }).waitFor()
  await desktop.locator('.artwork-button').click()
  await desktop.locator('.exit-immersive').waitFor()
  assert.equal(await desktop.locator('.workspace').count(), 0)
  await desktop.getByRole('button', { name: 'Expand / collapse lyrics' }).click()
  assert.equal(await desktop.locator('.lyrics-focused').count(), 1)
  await desktop.locator('.exit-immersive').click()

  const { page: mobile } = await setup({ mobile: true })
  await mobile.getByRole('button', { name: 'Next / Skip track', exact: true }).click()
  assert.ok(calls.some(call => call.body?.command === 'next'))
  await mobile.getByRole('button', { name: 'Equalizer', exact: true }).click()
  await mobile.getByText('Native audio on mobile preserves background playback. EQ is bypassed.').waitFor()
  const incoming = { messageID: 'incoming', senderUsername: 'other', senderDisplayName: 'Another Listener', body: 'Hello from the room!', createdAt: stamp, references: [], mentions: [] }
  await emit(mobile, 'chat_message', incoming)
  assert.equal(await mobile.locator('.mobile-tabs .unread-badge').innerText(), '1')
  for (const name of ['Playing', 'Queue', 'History', 'Music library', 'Favorites', 'Chat']) { await tab(mobile, name, true); await checkOverflow(mobile, `mobile ${name}`) }
  assert.equal(await mobile.locator('.mobile-tabs .unread-badge').count(), 0)
  await mobile.screenshot({ path: path.join(output, 'mobile-chat.png'), fullPage: true })
  await tab(mobile, 'Favorites', true)
  for (const name of ['Songs', 'Albums', 'Artists', 'Playlists']) { await mobile.locator('.workspace .segmented').getByRole('button', { name, exact: true }).click(); await checkOverflow(mobile, `mobile favorites ${name}`) }
  await mobile.getByRole('button', { name: 'Night Playlist' }).click()
  await mobile.locator('.workspace .song-row').waitFor()
  await mobile.getByRole('button', { name: 'Members', exact: true }).click()
  await mobile.getByRole('button', { name: 'Statistics & rankings', exact: true }).click()
  await mobile.getByText('Top DJs', { exact: true }).waitFor()
  await checkOverflow(mobile, 'mobile statistics')
  await mobile.getByRole('button', { name: 'Close', exact: true }).click()
  await tab(mobile, 'Playing', true)
  for (const language of ['zh-CN', 'zh-TW', 'ja', 'es', 'fr', 'de', 'en']) {
    await mobile.locator('.language-selector').selectOption(language)
    assert.equal(await mobile.locator('html').getAttribute('lang'), language)
    await checkOverflow(mobile, `mobile language ${language}`)
  }
  await mobile.screenshot({ path: path.join(output, 'mobile-playing.png'), fullPage: true })
  const { page: login } = await setup({ mobile: true, login: true })
  await login.getByRole('heading', { name: 'Join with Navidrome' }).waitFor()
  await login.locator('.language-selector').selectOption('fr')
  assert.equal(await login.title(), 'Écoutons ensemble · Navidrome')
  await checkOverflow(login, 'mobile login')
  for (const page of [desktop, mobile, login]) assert.deepEqual(await page.evaluate(() => window.__cspErrors), [], 'CSP violation')
  assert.deepEqual(faults, [], 'browser exceptions')
  console.log('UI checks passed: desktop split, queue keyboard reorder, settings, statistics, wishes, profile badges, chat, member skip, immersive lyrics, seven locales, 390px layout and CSP.')
} catch (error) {
  for (const context of browser.contexts()) for (const page of context.pages()) {
    console.error('UI failure state:', await page.locator('body').innerText())
    await page.screenshot({ path: path.join(output, 'failure.png'), fullPage: true })
  }
  throw error
} finally {
  await browser.close()
  await new Promise(resolve => server.close(resolve))
}
