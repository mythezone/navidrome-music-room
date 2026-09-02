# Architecture

## Boundaries

The `.ndp` bridge has four Navidrome permissions: `users`, `scheduler`, `http`, and `kvstore`. It receives only explicitly authorized usernames/display names/admin flags, runs a recurring callback, sends to fixed local gateway hosts, and stores a schedule ID plus generation counter. It has no music-folder filesystem permission and no write access.

Navidrome v0.63.2's web application uses React 17, Material UI v4,
react-admin 3, and Vite; plugin configuration is rendered from JSONForms. The
host does not expose a supported sidebar/custom-page or player-control slot.
The plugin therefore publishes the stable `manifest.website` entry
`/music-room/admin/`. A same-origin Compose edge routes that path to a React
17/Material UI v4 console embedded in the gateway binary and routes invitations
below `/music-room/join/{roomID}/` to an embedded Vue 3 room client. No
Navidrome source patch is used.

Both browser surfaces read the `username`, `subsonic-salt`, and
`subsonic-token` values
created by the current Navidrome login and exchanges them for the same
15-minute in-memory gateway session used by MusicMate. It never receives the
user's raw password. The room client owns exactly one audio element and controls
it from the server's revisioned clock. Share QR codes are rendered locally; invite fragments are
not sent to another QR service.

The gateway owns room coordination only. It stores rooms, persistent membership, invitation digests, queue, playback history, security audit records, migration state, and updater state. It does not store Navidrome passwords, OpenSubsonic proofs, permanent stream URLs, or audio bodies.

The Web room and MusicMate own playback. Both use the listener's Navidrome
account for `search3`, `getAlbum`, `getSong`, `getCoverArt`,
`getLyricsBySongId`, and `stream`; MusicMate alone stores a password long-term
in the platform credential store.

## Trust flow

1. The plugin sends its complete allowlist, configured internal/public endpoints, and monotonic generation using a pairing bearer token. The gateway rejects endpoint drift before accepting the lease.
   If the plugin KV state is lost, it consumes the gateway's stale-generation
   response and safely resumes above the stored counter.
2. A new gateway session requires a heartbeat no older than 90 seconds.
3. The gateway forwards an OpenSubsonic salt/token proof only to its configured Navidrome internal URL.
4. `getUser` supplies `adminRole` and accessible folder IDs. The allowlist and Navidrome result must both authorize the user.
5. The opaque gateway session expires after 15 minutes; its OpenSubsonic proof exists only in memory.
6. Every request rechecks the current plugin allowlist. Existing sessions tolerate a stale heartbeat for only 60 additional seconds.

## Playback model

The server stores a base position and optional server-time anchor. A playing client's expected position is:

```text
basePosition + (now - anchorServerTime)
```

Mutations require `expectedRevision`. A stale request receives `409 revision_conflict` with the latest state. The gateway records only `NavidromeTrackRef`; clients independently turn the ID into a media request. When the final WebSocket leaves, a 15-second timer pauses an active room with a new revision.

Queue selection prevents one contributor from monopolizing playback. FIFO chooses the oldest entry from another contributor when possible. Fair-random chooses a contributor first and then one of that contributor's tracks. The personal pending limit is stored per room.

## Persistence

SQLite lives below the plugin folder but does not share Navidrome's schema. WAL, foreign keys, one writer lock for compound mutations, and transactional migrations are enabled. Current tables are:

- `rooms`
- `members`
- `invites`
- `queue`
- `playback_history`
- `security_audit`
- `plugin_state`
- `update_state`
- `schema_migrations`

There are intentionally no chat, statistics, ranking, level, or achievement tables until those open-source roadmap features are implemented.

## Deployment model

The Compose edge is the only published listener. It forwards `/` to Navidrome
and strips `/music-room/` before forwarding HTTP and WebSocket traffic to the
private gateway listener. Consequently Navidrome login storage is same-origin
with the management console, and a single TLS certificate protects both.
The edge access-log format intentionally omits query strings because
OpenSubsonic proofs are query parameters.

v1 supports one gateway process. The launcher supervises that process and owns atomic release switching. Redis, distributed locks, cloud relays, and multi-node consensus are intentionally absent.
