# API contracts and compatibility fixtures

`openapi.yaml` is the normative REST contract. JSON files in `fixtures/` are golden payloads used by gateway and MusicMate compatibility tests.

The event names intentionally preserve the existing FAIO room vocabulary: `snapshot`, `playback`, `queue`, `history`, `presence`, `room_updated`, `chat_message`, `chat_updated`, `wish_updated`, `wish_deleted`, `update_available`, and `error`. Navidrome payloads replace FAIO file URLs with `NavidromeTrackRef`; clients must obtain media from Navidrome using their own credentials.

Contract changes follow semantic versioning. Removing a field or changing its meaning requires a new `/api/v2` namespace.

The fixtures cover the room address, authoritative snapshot, WebSocket events,
revision conflicts, chat pages, wishes, update notices, current capabilities, and
paired-version readiness payloads.

The v1.2 community endpoints require active membership. Chat references use
`{kind, id, title, artist?}`; wish resources use `{title, url}` with HTTP(S) URLs.
Empty collections are always arrays. Recalled and deleted chat messages remain
as tombstones, and reply previews reflect their current visibility.

Statistics and rankings use UTC calendar periods (weeks start Monday).
`MediaStat` entries contain `{id, name, artist, count, seconds}`; member ranking
entries contain `{username, displayName, count}`. Listening credit requires 30
seconds or half the track duration and is awarded to online members plus the
contributor, once per finished history entry. Requested and listening rankings
for bounded periods use activity timestamps recorded by v1.2.

`updateNotice` in snapshots and `update_available` WebSocket events are visible
only to Navidrome administrators. Channel changes schedule an asynchronous
check; periodic startup checks are controlled by `MUSIC_ROOM_AUTO_UPDATE_CHECK`
(default true) and `MUSIC_ROOM_UPDATE_CHECK_INTERVAL` (default 6h). The optional
`MUSIC_ROOM_UPDATE_WEBHOOK_URL` receives each newly detected version once.
