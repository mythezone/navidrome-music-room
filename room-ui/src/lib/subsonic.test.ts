import { afterEach, describe, expect, it, vi } from 'vitest'

import { SubsonicClient } from './subsonic'

describe('SubsonicClient catalog browsing', () => {
  afterEach(() => vi.restoreAllMocks())

  it('loads songs, newest albums and artists from Navidrome in one catalog refresh', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = new URL(String(input))
      let payload: Record<string, unknown> = {}
      if (url.pathname.endsWith('/getRandomSongs.view')) {
        payload = { randomSongs: { song: [{ id: 'song-1', title: 'Song One' }] } }
      } else if (url.pathname.endsWith('/getAlbumList2.view')) {
        payload = { albumList2: { album: [{ id: 'album-1', name: 'Album One' }] } }
      } else if (url.pathname.endsWith('/getArtists.view')) {
        payload = { artists: { index: [{ name: 'A', artist: [{ id: 'artist-1', name: 'Artist One' }] }] } }
      }
      return {
        ok: true,
        status: 200,
        json: async () => ({ 'subsonic-response': { status: 'ok', ...payload } }),
      } as Response
    })

    const client = new SubsonicClient('https://music.example.test', {
      username: 'listener', salt: 'salt', token: 'token',
    })
    const catalog = await client.browse(7)

    expect(catalog.songs.map((song) => song.id)).toEqual(['song-1'])
    expect(catalog.albums.map((album) => album.id)).toEqual(['album-1'])
    expect(catalog.artists.map((artist) => artist.id)).toEqual(['artist-1'])

    const calledURLs = fetchMock.mock.calls.map(([input]) => new URL(String(input)))
    const songsURL = calledURLs.find((url) => url.pathname.endsWith('/getRandomSongs.view'))
    const albumsURL = calledURLs.find((url) => url.pathname.endsWith('/getAlbumList2.view'))
    expect(songsURL?.searchParams.get('musicFolderId')).toBe('7')
    expect(songsURL?.searchParams.get('size')).toBe('80')
    expect(albumsURL?.searchParams.get('musicFolderId')).toBe('7')
    expect(albumsURL?.searchParams.get('type')).toBe('newest')
  })
})
