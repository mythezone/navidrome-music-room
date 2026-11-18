export interface NavidromeProof {
  username: string
  salt: string
  token: string
}

export interface NavidromeUser {
  username: string
  displayName: string
  adminRole: boolean
  musicFolderIDs: number[]
}

export interface GatewaySession {
  sessionToken: string
  expiresAt: string
  user: NavidromeUser
  navidromeBaseURL: string
  gatewayBaseURL: string
  pluginLeaseExpiresAt: string
}

export interface RoomCapabilities {
  chat: boolean
  stickers: boolean
  statistics: boolean
  rankings: boolean
  achievements: boolean
  uploads?: boolean
  onlineSources?: boolean
  invites: boolean
  queue: boolean
  history: boolean
}

export interface Room {
  roomID: string
  name: string
  ownerUsername: string
  ownerDisplayName: string
  status: 'open' | 'closed'
  queueLimit: number
  playbackMode: 'fifo' | 'fair_random'
  musicFolderIDs: number[]
  preloadNextTrack: boolean
  allowMemberSkip: boolean
  capabilities: RoomCapabilities
  onlineCount: number
  createdAt: string
  updatedAt: string
}

export interface Member {
  roomID?: string
  username: string
  displayName: string
  role: 'owner' | 'manager' | 'member'
  active: boolean
  online: boolean
  joinedAt: string
  lastSeenAt: string
}

export interface TrackRef {
  id: string
  musicFolderID: number
  albumID?: string
  artistID?: string
  title: string
  artist: string
  album: string
  durationSeconds: number
  coverArtID?: string
}

export interface QueueEntry {
  queueID: string
  roomID?: string
  position: number
  track: TrackRef
  contributorUsername: string
  contributorDisplayName: string
  createdAt: string
}

export interface PlaybackState {
  revision: number
  serverTime: string
  status: 'stopped' | 'playing' | 'paused'
  pausedForEmpty: boolean
  positionSeconds: number
  anchorServerTime?: string
  currentTrack?: TrackRef
  nextTrack?: TrackRef
  contributorUsername?: string
  contributorDisplayName?: string
}

export interface HistoryEntry {
  historyID: string
  roomID?: string
  track: TrackRef
  contributorUsername: string
  contributorDisplayName: string
  startedAt: string
  finishedAt?: string
  playedSeconds: number
}

export interface RoomSnapshot {
  room: Room
  self: Member
  members: Member[]
  queue: QueueEntry[]
  playback: PlaybackState
  history: HistoryEntry[]
  generatedAt: string
  permissions?: RoomPermissions
  updateNotice?: UpdateNotice
}

export interface GatewayEvent<T = unknown> {
  type: 'snapshot' | 'playback' | 'queue' | 'history' | 'presence' | 'room_updated' | 'pong' | 'error' | 'chat_message' | 'chat_updated' | 'wish_updated' | 'wish_deleted' | 'update_available'
  roomID: string
  revision?: number
  serverTime: string
  payload: T
}

export interface QueueMutation {
  entry: QueueEntry
  queue: QueueEntry[]
  playback?: PlaybackState
}

export interface SubsonicSong {
  id: string
  title: string
  artist?: string
  album?: string
  albumId?: string
  artistId?: string
  musicFolderId?: number
  coverArt?: string
  duration?: number
  starred?: string
  track?: number
  year?: number
  suffix?: string
  contentType?: string
}

export interface SubsonicAlbum {
  id: string
  name: string
  artist?: string
  artistId?: string
  coverArt?: string
  songCount?: number
  duration?: number
  year?: number
  starred?: string
  song?: SubsonicSong[]
}

export interface SubsonicArtist {
  id: string
  name: string
  coverArt?: string
  albumCount?: number
  starred?: string
}

export interface SubsonicPlaylist {
  id: string
  name: string
  songCount?: number
  duration?: number
  owner?: string
  public?: boolean
  coverArt?: string
  entry?: SubsonicSong[]
}

export interface CatalogResults {
  songs: SubsonicSong[]
  albums: SubsonicAlbum[]
  artists: SubsonicArtist[]
}

export interface LyricLine {
  time: number
  text: string
}

export interface GatewayErrorBody {
  error?: {
    code?: string
    message?: string
    details?: unknown
  }
}

export interface RoomPermissions {
  canManage: boolean
  canControl: boolean
  canSkip: boolean
  canReorderQueue: boolean
  canFulfillWish: boolean
}

export type RoomSettings = Pick<Room, 'name' | 'playbackMode' | 'queueLimit' | 'preloadNextTrack' | 'allowMemberSkip'>
export type PlaybackCommand = 'play' | 'pause' | 'seek' | 'next' | 'previous' | 'stop'
export type Period = 'day' | 'week' | 'month' | 'year' | 'all'

export interface UpdateNotice {
  currentVersion: string
  latestVersion: string
  channel: string
  releaseURL: string
}

export interface Sticker {
  stickerID: string
  packID: string
  emoji: string
  code: string
  labels: Record<string, string>
}
export interface StickerPack { packID: string; stickers: Sticker[] }
export interface ChatReference {
  kind: 'track' | 'album' | 'artist'
  id: string
  title: string
  artist?: string
}
export interface ChatPayload {
  body: string
  messageType?: 'chat' | 'review'
  references?: ChatReference[]
  mentions?: string[]
  replyToID?: string
  stickerID?: string
}
export interface ChatReplyPreview {
  messageID: string
  senderUsername: string
  senderDisplayName: string
  body: string
  stickerID: string
  recalledAt?: string
  deletedAt?: string
}
export interface ChatMessage extends ChatPayload {
  messageID: string
  roomID?: string
  senderUsername: string
  senderDisplayName: string
  createdAt: string
  recalledAt?: string
  deletedAt?: string
  replyPreview?: ChatReplyPreview
}
export interface MediaStat { id: string; name: string; artist: string; count: number; seconds: number }
export interface RoomStatistics {
  period: Period
  totalPlayCount: number
  totalPlayedSeconds: number
  totalChatCount: number
  activeMemberCount: number
  topArtists: MediaStat[]
  topAlbums: MediaStat[]
  topTracks: MediaStat[]
  hourlyDistribution: Array<{ hour: number; count: number; seconds: number }>
}
export interface RankedMember { username: string; displayName: string; count: number }
export interface RoomRankings {
  period: Period
  requested: RankedMember[]
  listened: RankedMember[]
  chat: RankedMember[]
  topArtists: MediaStat[]
  topAlbums: MediaStat[]
  topTracks: MediaStat[]
}
export interface Badge {
  badgeID: string
  code: string
  kind: 'requested' | 'listened' | 'chat' | 'artist' | 'album'
  subject: string
  tier: number
  count: number
  target: number
  unlocked: boolean
  unlockedAt?: string
}
export interface MemberProfile {
  username: string
  displayName: string
  role: Member['role']
  online: boolean
  joinedAt: string
  lastSeenAt: string
  stats: { staySeconds: number; requestedCount: number; listenedCount: number; chatCount: number }
  level: { level: number; code: string; effectiveMinutes: number; currentThresholdMinutes: number; nextThresholdMinutes: number; progress: number }
  badges: Badge[]
  selectedBadgeID: string
}
export interface WishResource { title: string; url: string }
export interface RoomWish {
  wishID: string
  roomID: string
  authorUsername: string
  authorDisplayName: string
  kind: 'track' | 'album' | 'artist'
  title: string
  artist: string
  reason: string
  status: 'pending' | 'fulfilled'
  createdAt: string
  updatedAt: string
  fulfilledBy: string
  fulfilledByName: string
  responseNote: string
  resources: WishResource[]
}
export type WishPayload = Pick<RoomWish, 'kind' | 'title' | 'artist' | 'reason'>
export interface WishFulfillment { responseNote: string; resources: WishResource[] }
