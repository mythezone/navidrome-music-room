import type { ChatMessage, GatewaySession, RoomSnapshot } from '../types'

export function roomSnapshot(): RoomSnapshot {
  return {
    room: { roomID: '0123456789abcdef0123456789abcdef', name: 'Night Radio', ownerUsername: 'owner', ownerDisplayName: 'Owner', status: 'open', queueLimit: 5, playbackMode: 'fifo', musicFolderIDs: [7], preloadNextTrack: false, allowMemberSkip: false, capabilities: { chat: true, stickers: true, statistics: true, rankings: true, achievements: true, invites: true, queue: true, history: true }, onlineCount: 2, createdAt: '2026-11-18T12:00:00Z', updatedAt: '2026-11-18T12:00:00Z' },
    self: { username: 'listener', displayName: 'Listener', role: 'member', active: true, online: true, joinedAt: '2026-11-18T12:00:00Z', lastSeenAt: '2026-11-18T12:00:00Z' },
    members: [], queue: [], history: [], generatedAt: new Date().toISOString(),
    playback: { revision: 2, serverTime: new Date().toISOString(), status: 'paused', pausedForEmpty: false, positionSeconds: 10, contributorUsername: 'other', contributorDisplayName: 'Other', currentTrack: { id: 'song-1', albumID: 'album-1', musicFolderID: 7, title: 'Night Song', artist: 'Night Artist', album: 'Night Album', durationSeconds: 180 } },
  }
}
export function gatewaySession(): GatewaySession {
  return { sessionToken: 'test-session', expiresAt: new Date(Date.now() + 3600000).toISOString(), user: { username: 'listener', displayName: 'Listener', adminRole: false, musicFolderIDs: [7] }, navidromeBaseURL: window.location.origin, gatewayBaseURL: window.location.origin + '/music-room', pluginLeaseExpiresAt: new Date(Date.now() + 3600000).toISOString() }
}
export function chatMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
  return { messageID: 'message-1', senderUsername: 'other', senderDisplayName: 'Other', body: 'Hello', createdAt: new Date().toISOString(), mentions: [], references: [], ...overrides }
}
