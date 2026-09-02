import type {
  CatalogResults,
  LyricLine,
  NavidromeProof,
  SubsonicAlbum,
  SubsonicArtist,
  SubsonicPlaylist,
  SubsonicSong,
  TrackRef,
} from '../types'
import { parseLRC } from './sync'

interface SubsonicEnvelope {
  'subsonic-response'?: Record<string, any>
}

export class SubsonicError extends Error {
  constructor(message: string, public readonly code = 'subsonic_failed') {
    super(message)
  }
}

function list<T>(value: T[] | undefined | null): T[] {
  return Array.isArray(value) ? value : []
}

export class SubsonicClient {
  constructor(
    private readonly baseURL: string,
    private readonly proof: NavidromeProof,
  ) {}

  private url(method: string, parameters: Record<string, string | number | undefined> = {}): URL {
    const endpoint = new URL(`${this.baseURL.replace(/\/$/, '')}/rest/${method}.view`)
    const query = new URLSearchParams({
      u: this.proof.username,
      s: this.proof.salt,
      t: this.proof.token,
      v: '1.16.1',
      c: 'NavidromeMusicRoomWeb',
      f: 'json',
    })
    for (const [key, value] of Object.entries(parameters)) {
      if (value !== undefined && value !== '') query.set(key, String(value))
    }
    endpoint.search = query.toString()
    return endpoint
  }

  private async request(method: string, parameters: Record<string, string | number | undefined> = {}): Promise<Record<string, any>> {
    const response = await fetch(this.url(method, parameters), {
      headers: { Accept: 'application/json' },
      credentials: 'same-origin',
    })
    const body = (await response.json().catch(() => ({}))) as SubsonicEnvelope
    const envelope = body['subsonic-response'] || {}
    if (!response.ok || envelope.status !== 'ok') {
      throw new SubsonicError(envelope.error?.message || `Navidrome 请求失败 (${response.status})`)
    }
    return envelope
  }

  assertSameOrigin(origin = window.location.origin): void {
    if (new URL(this.baseURL, origin).origin !== origin) {
      throw new SubsonicError('Web 听歌房要求 Navidrome 与房间网关使用同一个公开域名', 'same_origin_required')
    }
  }

  async folders(): Promise<Array<{ id: number; name: string }>> {
    const result = await this.request('getMusicFolders')
    return list<Record<string, unknown>>(result.musicFolders?.musicFolder).map((folder) => ({
      id: Number(folder.id),
      name: String(folder.name || `音乐库 ${folder.id}`),
    }))
  }

  async browse(folderID: number): Promise<CatalogResults> {
    const [songsResponse, albumsResponse, artistsResponse] = await Promise.all([
      this.request('getRandomSongs', { size: 80, musicFolderId: folderID }),
      this.request('getAlbumList2', { type: 'newest', size: 60, musicFolderId: folderID }),
      this.request('getArtists', { musicFolderId: folderID }),
    ])
    const artists: SubsonicArtist[] = []
    for (const group of list<Record<string, any>>(artistsResponse.artists?.index)) {
      artists.push(...list<SubsonicArtist>(group.artist))
    }
    return {
      songs: list<SubsonicSong>(songsResponse.randomSongs?.song),
      albums: list<SubsonicAlbum>(albumsResponse.albumList2?.album),
      artists: artists.slice(0, 80),
    }
  }

  async search(query: string, folderID: number): Promise<CatalogResults> {
    const result = await this.request('search3', {
      query,
      songCount: 80,
      albumCount: 40,
      artistCount: 40,
      musicFolderId: folderID,
    })
    const search = result.searchResult3 || {}
    return {
      songs: list<SubsonicSong>(search.song),
      albums: list<SubsonicAlbum>(search.album),
      artists: list<SubsonicArtist>(search.artist),
    }
  }

  async album(albumID: string): Promise<SubsonicAlbum> {
    const result = await this.request('getAlbum', { id: albumID })
    return result.album as SubsonicAlbum
  }

  async artistAlbums(artistID: string): Promise<SubsonicAlbum[]> {
    const result = await this.request('getArtist', { id: artistID })
    return list<SubsonicAlbum>(result.artist?.album)
  }

  async starred(): Promise<CatalogResults> {
    const result = await this.request('getStarred2')
    const starred = result.starred2 || {}
    return {
      songs: list<SubsonicSong>(starred.song),
      albums: list<SubsonicAlbum>(starred.album),
      artists: list<SubsonicArtist>(starred.artist),
    }
  }

  async playlists(): Promise<SubsonicPlaylist[]> {
    const result = await this.request('getPlaylists')
    return list<SubsonicPlaylist>(result.playlists?.playlist)
  }

  async playlist(playlistID: string): Promise<SubsonicPlaylist> {
    const result = await this.request('getPlaylist', { id: playlistID })
    return result.playlist as SubsonicPlaylist
  }

  async setStarred(kind: 'song' | 'album' | 'artist', id: string, starred: boolean): Promise<void> {
    const key = kind === 'song' ? 'id' : kind === 'album' ? 'albumId' : 'artistId'
    await this.request(starred ? 'star' : 'unstar', { [key]: id })
  }

  streamURL(songID: string): string {
    return this.url('stream', { id: songID, format: 'raw' }).toString()
  }

  coverURL(coverID: string | undefined, size = 720): string {
    return coverID ? this.url('getCoverArt', { id: coverID, size }).toString() : ''
  }

  async lyrics(song: SubsonicSong): Promise<LyricLine[]> {
    try {
      const result = await this.request('getLyricsBySongId', { id: song.id })
      const entries = list<Record<string, any>>(result.lyricsList?.structuredLyrics)
      const preferred = entries.find((entry) => entry.synced) || entries[0]
      const lines = list<Record<string, unknown>>(preferred?.line).map((line) => ({
        time: Math.max(0, Number(line.start || 0) / 1000),
        text: String(line.value || '').trim(),
      })).filter((line) => line.text)
      if (lines.length) return lines
    } catch {
      // Older servers can omit getLyricsBySongId; fall through to legacy lyrics.
    }
    try {
      const result = await this.request('getLyrics', { artist: song.artist, title: song.title })
      return parseLRC(String(result.lyrics?.value || result.lyrics || ''))
    } catch {
      return []
    }
  }

  toTrack(song: SubsonicSong, musicFolderID: number): TrackRef {
    return {
      id: song.id,
      musicFolderID,
      albumID: song.albumId,
      title: song.title || '未知歌曲',
      artist: song.artist || '未知歌手',
      album: song.album || '未知专辑',
      durationSeconds: Number(song.duration || 0),
      coverArtID: song.coverArt,
    }
  }
}
