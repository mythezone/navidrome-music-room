#!/usr/bin/env node

import assert from 'node:assert/strict'
import { createHash, randomBytes } from 'node:crypto'

const baseURL = required('NMR_BASE_URL').replace(/\/$/, '')
const gatewayPrefix = (process.env.NMR_GATEWAY_PREFIX || '/music-room').replace(/\/$/, '')
const admin = credentials('NMR_ADMIN')
const member = credentials('NMR_MEMBER')
const roomName = process.env.NMR_ROOM_NAME || 'Navidrome Music Room acceptance'
const searchQuery = process.env.NMR_SEARCH_QUERY || '*'
const clientName = 'MusicMate-Acceptance'

function required(name) {
  const value = process.env[name]?.trim()
  if (!value) throw new Error(`${name} is required`)
  return value
}

function credentials(prefix) {
  return {
    username: required(`${prefix}_USERNAME`),
    password: required(`${prefix}_PASSWORD`),
  }
}

function proofFor(account) {
  const salt = randomBytes(16).toString('hex')
  const token = createHash('md5').update(account.password + salt).digest('hex')
  return { username: account.username, salt, token }
}

function subsonicURL(method, proof, parameters = {}) {
  const result = new URL(`${baseURL}/rest/${method}.view`)
  for (const [key, value] of Object.entries({
    u: proof.username,
    t: proof.token,
    s: proof.salt,
    v: '1.16.1',
    c: clientName,
    f: 'json',
    ...parameters,
  })) {
    result.searchParams.set(key, String(value))
  }
  return result
}

async function subsonic(method, proof, parameters = {}) {
  const response = await fetch(subsonicURL(method, proof, parameters), {
    headers: { accept: 'application/json' },
    signal: AbortSignal.timeout(30_000),
  })
  assert.equal(response.status, 200, `${method} returned HTTP ${response.status}`)
  const payload = await response.json()
  const envelope = payload['subsonic-response']
  assert.equal(envelope?.status, 'ok', `${method} failed: ${JSON.stringify(envelope?.error)}`)
  return envelope
}

async function gateway(path, { method = 'GET', token = '', body, expected = [200] } = {}) {
  const headers = { accept: 'application/json' }
  if (token) headers.authorization = `Bearer ${token}`
  if (body !== undefined) headers['content-type'] = 'application/json'
  const response = await fetch(`${baseURL}${gatewayPrefix}/api/v1${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: AbortSignal.timeout(30_000),
  })
  let payload = null
  if (response.status !== 204) {
    const text = await response.text()
    payload = text ? JSON.parse(text) : null
  }
  assert.ok(expected.includes(response.status), `${method} ${path} returned HTTP ${response.status}: ${JSON.stringify(payload)}`)
  return { status: response.status, payload, headers: response.headers }
}

async function exchange(proof) {
  return (await gateway('/auth/exchange', { method: 'POST', body: proof })).payload
}

function connectWebSocket(url) {
  const socket = new WebSocket(url)
  const buffered = []
  const waiters = []
  let terminalError

  socket.addEventListener('message', (message) => {
    const event = JSON.parse(String(message.data))
    const waiterIndex = waiters.findIndex((waiter) => waiter.type === event.type)
    if (waiterIndex >= 0) {
      const [waiter] = waiters.splice(waiterIndex, 1)
      clearTimeout(waiter.timer)
      waiter.resolve(event)
    } else {
      buffered.push(event)
    }
  })
  socket.addEventListener('error', () => {
    terminalError = new Error('WebSocket connection failed')
    for (const waiter of waiters.splice(0)) {
      clearTimeout(waiter.timer)
      waiter.reject(terminalError)
    }
  })

  const opened = new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error('WebSocket open timed out')), 5_000)
    socket.addEventListener('open', () => {
      clearTimeout(timer)
      resolve()
    }, { once: true })
    socket.addEventListener('error', () => {
      clearTimeout(timer)
      reject(terminalError || new Error('WebSocket connection failed'))
    }, { once: true })
  })

  function next(type, timeout = 5_000) {
    const index = buffered.findIndex((event) => event.type === type)
    if (index >= 0) return Promise.resolve(buffered.splice(index, 1)[0])
    if (terminalError) return Promise.reject(terminalError)
    return new Promise((resolve, reject) => {
      const waiter = { type, resolve, reject }
      waiter.timer = setTimeout(() => {
        const index = waiters.indexOf(waiter)
        if (index >= 0) waiters.splice(index, 1)
        reject(new Error(`WebSocket ${type} event timed out`))
      }, timeout)
      waiters.push(waiter)
    })
  }

  return { socket, opened, next }
}

const adminProof = proofFor(admin)
const memberProof = proofFor(member)

const folders = (await subsonic('getMusicFolders', adminProof)).musicFolders?.musicFolder || []
assert.ok(folders.length > 0, 'admin has no accessible Navidrome music folders')
const folderID = Number(folders[0].id)
assert.ok(Number.isInteger(folderID) && folderID > 0, 'Navidrome returned an invalid music folder ID')

const search = await subsonic('search3', adminProof, {
  query: searchQuery,
  artistCount: 0,
  albumCount: 0,
  songCount: 20,
  musicFolderId: folderID,
})
const tracks = search.searchResult3?.song || []
assert.ok(tracks.length > 0, `search3 returned no songs for ${JSON.stringify(searchQuery)}`)
const searchedTrack = tracks[0]

const song = (await subsonic('getSong', adminProof, { id: searchedTrack.id })).song
assert.equal(song?.id, searchedTrack.id, 'getSong did not return the selected track')
assert.ok(song.albumId, 'selected song has no album ID')
const album = (await subsonic('getAlbum', adminProof, { id: song.albumId })).album
assert.equal(album?.id, song.albumId, 'getAlbum did not return the selected album')

let coverBytes = 0
if (song.coverArt) {
  const response = await fetch(subsonicURL('getCoverArt', adminProof, { id: song.coverArt }), {
    signal: AbortSignal.timeout(30_000),
  })
  assert.equal(response.status, 200, `getCoverArt returned HTTP ${response.status}`)
  coverBytes = (await response.arrayBuffer()).byteLength
  assert.ok(coverBytes > 0, 'getCoverArt returned an empty body')
}

const lyrics = await subsonic('getLyricsBySongId', adminProof, { id: song.id })
const structuredLyrics = lyrics.lyricsList?.structuredLyrics || []

const rangeResponse = await fetch(subsonicURL('stream', adminProof, { id: song.id }), {
  headers: { range: 'bytes=0-255' },
  signal: AbortSignal.timeout(30_000),
})
assert.equal(rangeResponse.status, 206, `Range stream returned HTTP ${rangeResponse.status}`)
const rangeBytes = (await rangeResponse.arrayBuffer()).byteLength
assert.equal(rangeBytes, 256, `Range stream returned ${rangeBytes} bytes instead of 256`)

const transcodeResponse = await fetch(subsonicURL('stream', adminProof, {
  id: song.id,
  format: 'mp3',
  maxBitRate: 64,
}), { signal: AbortSignal.timeout(60_000) })
assert.equal(transcodeResponse.status, 200, `transcoded stream returned HTTP ${transcodeResponse.status}`)
const transcodeReader = transcodeResponse.body.getReader()
const transcodeChunk = await transcodeReader.read()
assert.equal(transcodeChunk.done, false, 'transcoded stream returned an empty body')
assert.ok(transcodeChunk.value.byteLength > 0, 'transcoded stream returned an empty first chunk')
await transcodeReader.cancel()

const adminExchange = await exchange(adminProof)
assert.equal(adminExchange.user.username, admin.username)
assert.equal(adminExchange.user.adminRole, true, 'room creator is not a Navidrome administrator')
assert.ok(adminExchange.user.musicFolderIDs.includes(folderID), 'admin exchange omitted the scanned library')

const memberExchange = await exchange(memberProof)
assert.equal(memberExchange.user.username, member.username)
assert.equal(memberExchange.user.adminRole, false, 'acceptance member unexpectedly has admin access')
assert.ok(memberExchange.user.musicFolderIDs.includes(folderID), 'member cannot access the room music folder')

const adminToken = adminExchange.sessionToken
const memberToken = memberExchange.sessionToken
const rooms = (await gateway('/rooms', { token: adminToken })).payload.rooms
let room = rooms.find((candidate) => candidate.name === roomName)
if (!room) {
  room = (await gateway('/rooms', {
    method: 'POST',
    token: adminToken,
    expected: [201],
    body: { name: roomName, musicFolderIDs: [folderID], queueLimit: 20, playbackMode: 'fifo' },
  })).payload
} else if (room.status === 'closed') {
  room = (await gateway(`/rooms/${room.roomID}/reopen`, { method: 'POST', token: adminToken, body: {} })).payload
}

const invite = (await gateway(`/rooms/${room.roomID}/invites`, {
  method: 'POST',
  token: adminToken,
  expected: [201],
  body: { label: '服务器完整链路验收邀请', maxUses: 20, singleUse: false },
})).payload
assert.ok(invite.shareURL.includes('#invite='), 'share URL does not keep the invitation in its fragment')

await gateway('/invites/redeem', {
  method: 'POST',
  token: memberToken,
  body: { roomID: room.roomID, invite: invite.invite },
})
await gateway(`/rooms/${room.roomID}/join`, { method: 'POST', token: memberToken, body: {} })

const initialSnapshot = (await gateway(`/rooms/${room.roomID}/snapshot`, { token: memberToken })).payload
assert.ok(Array.isArray(initialSnapshot.members), 'snapshot.members is not an array')
assert.ok(Array.isArray(initialSnapshot.queue), 'snapshot.queue is not an array')
assert.ok(Array.isArray(initialSnapshot.history), 'snapshot.history is not an array')

await gateway(`/rooms/${room.roomID}/queue/tracks`, {
  method: 'POST',
  token: memberToken,
  expected: [201],
  body: { track: { id: song.id, musicFolderID: folderID } },
})

const websocketClients = []
for (const token of [memberToken, adminToken, memberToken]) {
  const ticket = (await gateway(`/rooms/${room.roomID}/ws-ticket`, {
    method: 'POST',
    token,
    body: {},
  })).payload
  const client = connectWebSocket(ticket.webSocketURL)
  await client.opened
  const snapshot = await client.next('snapshot')
  assert.equal(snapshot.roomID, room.roomID)
  assert.ok(Array.isArray(snapshot.payload.queue), 'WebSocket snapshot queue is not an array')
  assert.ok(Array.isArray(snapshot.payload.history), 'WebSocket snapshot history is not an array')
  websocketClients.push(client)
}

const managerSnapshot = (await gateway(`/rooms/${room.roomID}/snapshot`, { token: adminToken })).payload
const beforeRevision = managerSnapshot.playback.revision
const play = (await gateway(`/rooms/${room.roomID}/playback`, {
  method: 'POST',
  token: adminToken,
  body: { command: 'play', expectedRevision: beforeRevision },
})).payload
assert.equal(play.revision, beforeRevision + 1)
assert.equal(play.status, 'playing')
assert.equal(play.currentTrack.id, song.id)
const playbackEvents = await Promise.all(websocketClients.map((client) => client.next('playback')))
for (const playbackEvent of playbackEvents) {
  assert.equal(playbackEvent.revision, play.revision, 'WebSocket client did not receive the authoritative playback revision')
  assert.equal(playbackEvent.payload.currentTrack.id, song.id)
  assert.equal(playbackEvent.payload.status, 'playing')
}

await new Promise((resolve) => setTimeout(resolve, 1_200))
const progressSnapshots = await Promise.all([
  gateway(`/rooms/${room.roomID}/snapshot`, { token: adminToken }),
  gateway(`/rooms/${room.roomID}/snapshot`, { token: memberToken }),
  gateway(`/rooms/${room.roomID}/snapshot`, { token: memberToken }),
])
const positions = progressSnapshots.map(({ payload }) => payload.playback.positionSeconds)
assert.ok(Math.min(...positions) >= play.positionSeconds + 0.8, `authoritative position did not advance: ${positions}`)
const progressDriftSeconds = Math.max(...positions) - Math.min(...positions)
assert.ok(progressDriftSeconds < 0.5, `three clients observed excessive position drift: ${positions}`)

const seekPosition = Math.min(12, Math.max(1, Number(song.duration || 180) - 1))
const seek = (await gateway(`/rooms/${room.roomID}/playback`, {
  method: 'POST',
  token: adminToken,
  body: { command: 'seek', expectedRevision: play.revision, positionSeconds: seekPosition },
})).payload
assert.equal(seek.revision, play.revision + 1)
const seekEvents = await Promise.all(websocketClients.map((client) => client.next('playback')))
for (const seekEvent of seekEvents) {
  assert.equal(seekEvent.revision, seek.revision, 'WebSocket client did not receive the seek revision')
  assert.ok(Math.abs(seekEvent.payload.positionSeconds - seekPosition) < 0.25, 'seek event carried the wrong position')
}

websocketClients[2].socket.close()
const reconnectTicket = (await gateway(`/rooms/${room.roomID}/ws-ticket`, {
  method: 'POST',
  token: memberToken,
  body: {},
})).payload
const reconnectedClient = connectWebSocket(reconnectTicket.webSocketURL)
await reconnectedClient.opened
const reconnectSnapshot = await reconnectedClient.next('snapshot')
assert.equal(reconnectSnapshot.payload.playback.revision, seek.revision)
assert.ok(reconnectSnapshot.payload.playback.positionSeconds >= seekPosition, 'reconnect snapshot moved playback backwards')
assert.ok(reconnectSnapshot.payload.playback.positionSeconds < seekPosition + 3, 'reconnect snapshot advanced too far')
const reconnectPosition = reconnectSnapshot.payload.playback.positionSeconds
websocketClients[2] = reconnectedClient

const conflict = await gateway(`/rooms/${room.roomID}/playback`, {
  method: 'POST',
  token: adminToken,
  expected: [409],
  body: { command: 'pause', expectedRevision: beforeRevision },
})
assert.equal(conflict.payload.error?.code, 'revision_conflict')

const memberControl = await gateway(`/rooms/${room.roomID}/playback`, {
  method: 'POST',
  token: memberToken,
  expected: [403],
  body: { command: 'pause', expectedRevision: seek.revision },
})
assert.equal(memberControl.payload.error?.code, 'room_manager_required')

const paused = (await gateway(`/rooms/${room.roomID}/playback`, {
  method: 'POST',
  token: adminToken,
  body: { command: 'pause', expectedRevision: seek.revision },
})).payload
assert.equal(paused.status, 'paused')
for (const client of websocketClients) client.socket.close()

const planned = await gateway(`/rooms/${room.roomID}/chat`, {
  token: memberToken,
  expected: [501],
})
assert.equal(planned.payload.error?.code, 'feature_not_implemented')
assert.equal(planned.payload.error?.details?.featureKey, 'chat')

const noMediaProxy = await fetch(`${baseURL}${gatewayPrefix}/api/v1/stream`, {
  signal: AbortSignal.timeout(10_000),
})
assert.equal(noMediaProxy.status, 404, 'gateway unexpectedly exposes a media stream endpoint')

const landingURL = new URL(invite.shareURL)
landingURL.hash = ''
const landing = await fetch(landingURL, { signal: AbortSignal.timeout(10_000) })
assert.equal(landing.status, 200, `invitation landing page returned HTTP ${landing.status}`)
assert.match(await landing.text(), /Navidrome Music Room/)

const finalSnapshot = (await gateway(`/rooms/${room.roomID}/snapshot`, { token: memberToken })).payload
assert.equal(finalSnapshot.playback.status, 'paused')
assert.ok(finalSnapshot.members.some((candidate) => candidate.username === member.username && candidate.active))
assert.ok(finalSnapshot.history.some((entry) => entry.track.id === song.id))

process.stdout.write(`${JSON.stringify({
  passed: true,
  navidrome: {
    musicFolderID: folderID,
    searchQuery,
    song: { id: song.id, title: song.title, artist: song.artist, album: song.album },
    coverBytes,
    structuredLyrics: structuredLyrics.length,
    rangeBytes,
    transcodeFirstChunkBytes: transcodeChunk.value.byteLength,
  },
  room: {
    roomID: room.roomID,
    name: room.name,
    shareURL: invite.shareURL,
    deepLink: invite.deepLink,
    inviteRemainingUses: invite.maxUses - 1,
    member: member.username,
    synchronizedClients: websocketClients.length,
    progressDriftSeconds,
    reconnectPosition,
    playbackRevision: finalSnapshot.playback.revision,
    historyEntries: finalSnapshot.history.length,
  },
  checks: {
    openSubsonic: true,
    invitationRedeem: true,
    memberJoin: true,
    queue: true,
    websocketSnapshot: true,
    websocketPlayback: true,
    synchronizedProgress: true,
    seekBroadcast: true,
    reconnectSnapshot: true,
    revisionConflict: true,
    memberPlaybackDenied: true,
    featureLock: true,
    gatewayDoesNotProxyAudio: true,
    invitationLanding: true,
  },
}, null, 2)}\n`)
