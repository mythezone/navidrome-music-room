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
  uploads: boolean
  onlineSources: boolean
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
  capabilities: RoomCapabilities
  onlineCount: number
  createdAt: string
  updatedAt: string
}

export interface Member {
  roomID?: string
  username: string
  displayName: string
  role: 'owner' | 'member'
  active: boolean
  online: boolean
  joinedAt: string
  lastSeenAt: string
}

export interface TrackRef {
  id: string
  musicFolderID: number
  albumID?: string
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
}

export interface GatewayEvent<T = unknown> {
  type: 'snapshot' | 'playback' | 'queue' | 'history' | 'presence' | 'room_updated' | 'pong' | 'error'
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
