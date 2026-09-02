# MusicMate integration

The HTTPS invitation is also the browser client URL in v1.0. MusicMate should
continue claiming the `musicmate://` deep link, while normal HTTPS navigation
must remain available as a complete fallback rather than forcing an app install.

## Provider boundary

Existing FAIO rooms remain unchanged. Add a provider discriminator and preserve old Codable defaults:

```swift
enum RoomProvider: String, Codable {
    case faio
    case navidrome
}

struct RoomAddress: Codable {
    let provider: RoomProvider
    let gatewayBaseURL: URL
    let navidromeBaseURL: URL?
    let roomID: String
    let invite: String?
}
```

`FAIORoomProvider` keeps the existing cookie/session and FAIO media URLs. `NavidromeRoomProvider` exchanges OpenSubsonic proof for a bearer session, calls `/api/v1`, and resolves media through Navidrome.

## Credential handling

Store the Navidrome password only in the platform credential store. For each exchange:

1. generate a cryptographically random hexadecimal salt;
2. compute lowercase hexadecimal MD5 over UTF-8 `password + salt`;
3. POST username, salt, and token to `/api/v1/auth/exchange`;
4. keep the returned room session in memory and exchange again before its 15-minute expiry.

Never persist the proof, append it to a room URL, or include it in analytics/crash metadata.

## Direct media URLs

Use the user's own OpenSubsonic auth parameters with the `currentTrack.id`:

- `/rest/stream.view?id=...`
- `/rest/getCoverArt.view?id=...`
- `/rest/getLyricsBySongId.view?id=...`

Do not use a URL broadcast by another client. `NavidromeTrackRef` is the shared durable identity.

## Links

The parser accepts:

- `https://gateway.example/join/{roomID}#invite={secret}`
- `https://music.example/music-room/join/{roomID}#invite={secret}` (same-origin Compose deployment)
- `musicmate://join?server=...&gateway=...&room=...&invite=...`
- existing FAIO `/listen/{roomID}` and `musicmate://room/{roomID}?server=...` links

Parse the HTTPS fragment locally and clear it from any displayed/logged URL. Generate QR images locally; do not send invitation URLs to a third-party QR service.

When `gatewayBaseURL` contains a path prefix such as `/music-room`, preserve it
when appending `/api/v1`; do not rebuild the endpoint from only scheme and host.

## WebSocket

POST `/ws-ticket`, then connect once to the returned URL. A rejected or reused ticket requires a new request. On reconnect, fetch a new snapshot and calculate playing position from `positionSeconds`, `anchorServerTime`, and the newly observed local/server clock delta.

The first successful queue request promotes the selected track to
`playback.currentTrack` with status `paused` and increments `revision`. Managers
must use the primary play button to send the shared `play`/`pause` command;
ordinary members' primary button only changes local listening. Every device then
advances the server-authoritative position locally and corrects drift on playback
events, seeks, track changes, and reconnect snapshots. Audio continues to stream
directly from Navidrome with that device's own OpenSubsonic credentials.

## Navidrome plugin activation adapter

An `.ndp` file replacement disables the plugin. For Navidrome v0.63.2 the administrator flow uses a short-lived UI JWT and these internal endpoints:

```text
POST /api/plugin/rescan
PUT  /api/plugin/navidrome-music-room   {"enabled":true}
```

`NavidromePluginAdminAdapter` sends `X-ND-Authorization: Bearer <UI JWT>` and never stores the JWT. `NavidromeUpdateCoordinator` requires explicit administrator confirmation, waits for the target gateway version, invokes the adapter, and confirms success only when `/readyz` reports a paired heartbeat from the matching plugin version.
