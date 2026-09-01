# API contracts and compatibility fixtures

`openapi.yaml` is the normative REST contract. JSON files in `fixtures/` are golden payloads used by gateway and MusicMate compatibility tests.

The event names intentionally preserve the existing FAIO room vocabulary: `snapshot`, `playback`, `queue`, `history`, `presence`, `room_updated`, and `error`. Navidrome payloads replace FAIO file URLs with `NavidromeTrackRef`; clients must obtain media from Navidrome using their own credentials.

Contract changes follow semantic versioning. Removing a field or changing its meaning requires a new `/api/v2` namespace.

The fixtures cover the room address, authoritative snapshot, WebSocket events,
revision conflict, uniform feature lock, offline entitlements, and paired-version
readiness payloads.
