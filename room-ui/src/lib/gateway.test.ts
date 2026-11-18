import { afterEach, expect, it, vi } from 'vitest'
import { GatewayClient } from './gateway'
import { gatewaySession, roomSnapshot } from '../test/fixtures'
afterEach(() => vi.restoreAllMocks())
it('uses the v1.2 gateway routes, encoded IDs and exact mutation payloads', async () => {
  const request = vi.spyOn(GatewayClient.prototype, 'request').mockResolvedValue(gatewaySession())
  const client = new GatewayClient('/music-room')
  await client.patchRoom('r', { allowMemberSkip: true })
  expect(request).toHaveBeenLastCalledWith('/rooms/r', { method: 'PATCH', body: { allowMemberSkip: true } })
  await client.playback('r', 'previous', 7)
  expect(request).toHaveBeenLastCalledWith('/rooms/r/playback', { method: 'POST', body: { command: 'previous', expectedRevision: 7 } })
  const track = roomSnapshot().playback.currentTrack!
  await client.addTrack('r', track, true)
  expect(request).toHaveBeenLastCalledWith('/rooms/r/queue/tracks', { method: 'POST', body: { track, playNext: true } })
  await client.topQueue('r', 'a/b'); expect(request).toHaveBeenLastCalledWith('/rooms/r/queue/a%2Fb/top', { method: 'POST', body: {} })
  await client.shuffleQueue('r'); expect(request).toHaveBeenLastCalledWith('/rooms/r/queue/shuffle', { method: 'POST', body: {} })
  await client.requeueHistory('r', 'h'); expect(request).toHaveBeenLastCalledWith('/rooms/r/history/h/requeue', { method: 'POST', body: {} })
  await client.chatMessages('r', 25, 'a/b'); expect(request).toHaveBeenLastCalledWith('/rooms/r/chat?limit=25&before=a%2Fb')
  await client.sendMessage('r', { body: 'Hi', replyToID: 'm', references: [{ kind: 'track', id: 's', title: 'Song' }] })
  expect(request).toHaveBeenLastCalledWith('/rooms/r/chat', { method: 'POST', body: { body: 'Hi', replyToID: 'm', references: [{ kind: 'track', id: 's', title: 'Song' }] } })
  await client.fulfillWish('r', 'w', { responseNote: 'Added', resources: [{ title: 'Album', url: 'https://music.test/app/#/album/a' }] })
  expect(request).toHaveBeenLastCalledWith('/rooms/r/wishes/w/fulfill', { method: 'POST', body: { responseNote: 'Added', resources: [{ title: 'Album', url: 'https://music.test/app/#/album/a' }] } })
  await client.selectBadge('r', ''); expect(request).toHaveBeenLastCalledWith('/rooms/r/achievements/selected', { method: 'PUT', body: { badgeID: '' } })
})
