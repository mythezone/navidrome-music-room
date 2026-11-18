# Changelog

All notable changes follow [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and Semantic Versioning.

## [Unreleased]

## [1.2.0] - 2026-11-18

### Added

- Seven-language internationalization (`en`, `zh-CN`, `zh-TW`, `ja`, `es`, `fr`,
  `de`) in the Web Listening Room and Administrator Console, with automatic
  Navidrome/browser locale detection, a persistent `nmr:locale` switcher, and
  localized error codes.
- Collaborative track skipping through per-room `allowMemberSkip`, enabled by
  default and configurable in the Admin Console and in-room settings. Contributors
  can skip their own current track even when member skipping is disabled; managers
  can also use the `previous` track command.
- Real-time Group Chat with `@member` and `@song` suggestions, playable track/album
  references, localized quick track review presets, replies, five-minute sender
  recall, manager deletion, unread tab badges, and two built-in localized
  sticker/reaction packs.
- A Room Information hub with Listening Statistics, period leaderboards
  (`day`/`week`/`month`/`year`/`all`) for Top DJs, Listeners, Chatters, Artists,
  Albums, and Tracks, and a 24-hour Listening Activity Clock. Member profiles
  include ten progression levels and unlockable achievements/badges. Song Wishes
  support a `pending`/`fulfilled` workflow with library resource attachments.
- FAIO-style Music Room UI/UX improvements: a resizable desktop split layout,
  click-artwork immersive fullscreen view with auto-scrolling expandable LRC
  lyrics, and a five-band Web Audio equalizer, bypassed on mobile touch devices
  to preserve native background audio. Queue controls include drag-and-drop
  reordering, Pin-to-Top, Shuffle Queue, and Random Fill; history supports
  pagination and one-click re-queue, and Favorites has Songs, Albums, Artists,
  and Playlists tabs.
- In-browser One-Click Signed Upgrade, Rollback, and redacted Diagnostics UI at
  `/music-room/admin/`, coordinating same-origin Navidrome `/api/plugin/rescan`
  and plugin re-enable automatically. Scheduled GitHub Release detection uses
  `MUSIC_ROOM_AUTO_UPDATE_CHECK` and `MUSIC_ROOM_UPDATE_CHECK_INTERVAL`, with
  admin-only WebSocket update badges, optional `MUSIC_ROOM_UPDATE_WEBHOOK_URL`
  push, and `.ndp` heartbeat update logging.

### Changed

- Updated `/api/v1/capabilities` and the OpenAPI contract to `1.2.0`, enabling
  `chat`, `stickers`, `statistics`, `rankings`, and `achievements` under GPL-3.0.
- Advanced SQLite to migration v5 with `rooms.allow_member_skip`, `chat_messages`,
  `room_wishes`, and `member_stats`.
- Frontend audit gates now check runtime dependencies with `--omit=dev`;
  the unpatched `GHSA-fc5q-39xc-vjpf` advisory remains in build/test dependencies.

### Fixed

- Fixed the unauthenticated Admin Console blank screen with the missing `LockIcon`
  import and a React `ErrorBoundary` ([#8](https://github.com/mythezone/navidrome-music-room/issues/8)).
  Added a prefix-preserving `308` redirect from `GET /admin` to `admin/`.
- Replaced updater string inequality with SemVer comparison so older versions
  and build-metadata-only changes are not offered as upgrades. Hardened version
  stamping in `build-plugin.sh`, `package-release.sh`, and `Dockerfile` to replace
  arbitrary existing version strings and normalize a leading `v`.
- Refreshed frontend lockfiles and regenerated both embedded production bundles.

## [1.1.0-dev] - 2026-09-02

### Added

- Added separate Songs, Albums, and Artists views to the browser request desk.
- Added whole-album requests and album/artist drill-down using Navidrome's standard
  OpenSubsonic library endpoints.
- Added a release-ready Docker Compose kit alongside the standalone `.ndp` and
  Linux amd64/arm64 gateway bundles.
- Added the new project logo, hero image, expanded badges, usage-first English and
  Chinese guides, MusicMate preview, collaboration details, and donation QR codes.

### Changed

- Reframed all unfinished features as an open-source roadmap and removed the
  experimental feature-claim configuration surface.
- Planned feature endpoints now return `501 feature_not_implemented`.

### Fixed

- Populate the initial song list with `getRandomSongs` instead of returning an
  empty array.
- Populate the initial album list with `getAlbumList2(type=newest)` instead of
  `recent`, which is empty until a user has listening history.

## [1.0.0] - 2026-09-02

### Added

- Added an embedded React 17/Material UI v4 administrator console for room CRUD,
  member management, revocable invitations, share links, and browser-local QR codes.
- Added a same-origin Compose edge: Navidrome remains at `/` and room HTTP/WebSocket
  traffic is available below `/music-room/` without exposing port 4534.
- Exposed the console through the stock Navidrome plugin detail page's Website field,
  without maintaining a Navidrome fork.
- Added an embedded Vue 3 Web listening room opened directly by HTTPS invitation
  links, with desktop and mobile layouts, Navidrome login, persistent membership,
  catalog search, favourites, playlists, queue, history, lyrics, and roadmap panels.
- Added browser playback backed by each listener's direct OpenSubsonic stream,
  the authoritative room clock, revisioned seek/play/pause/next events, drift repair,
  one-use WebSocket tickets, visibility recovery, and refresh reconnection.
- Added opt-in live Playwright acceptance coverage for two isolated browser clients,
  mobile overflow, refresh recovery, music-folder ACL rejection, and real Range audio.

### Fixed

- Promote the first requested song to a paused authoritative current track,
  broadcast the new playback revision, and repair pre-fix rooms that contain a
  pending queue without a current track when the gateway restarts.
- Make MusicMate's primary play button send shared play/pause commands for room
  owners and Navidrome administrators while preserving local-only listening
  controls for regular members.
- Start playback history only when the primed song actually begins playing.
- Serialize empty room, member, invite, queue, and history collections as `[]`
  instead of `null`, fixing MusicMate's `response_incompatible` decode error.
- Accept the six-character hexadecimal OpenSubsonic salt generated by Navidrome's
  own web login so the console can reuse the active session.
- Allow Navidrome administrators to inspect members of rooms they manage even when
  they have not previously joined those rooms.

## [0.1.0-beta.3] - 2026-09-01

### Fixed

- Ensure TinyGo release builds embed the requested plugin version so gateway heartbeats and one-click update confirmation no longer report `dev`.
- Fail plugin builds when version injection is missing.
- Enable Navidrome's plugin file watcher so a gateway-installed or updated `.ndp` is discovered without restarting Navidrome.

### Changed

- Let Compose track the official Navidrome `latest` image by default while keeping the image, data directory, and plugin directory configurable for pinned or isolated deployments.

## [0.1.0-beta.2] - 2026-09-01

### Fixed

- Made signature verification self-contained for non-root containers and systemd installs by bundling architecture-matched Cosign v3 and a TUF-authenticated Sigstore trusted root.
- Removed Cosign's deprecated `--offline` cache path and verified release bundles through explicit `--trusted-root` with no network or writable home directory.

Beta.1 installations require one manual image or signed-bundle update to
beta.2; automatic updates are self-contained from beta.2 onward.

## [0.1.0-beta.1] - 2026-09-01

### Added

- Go companion gateway with independent SQLite persistence.
- Navidrome v0.63.2 `.ndp` user authorization and lease heartbeat bridge.
- OpenSubsonic proof exchange, persistent invitation membership, room management, queue, authoritative playback, history, presence, and one-use WebSocket tickets.
- Consistent responses for feature surfaces not implemented by the preview.
- Signed GitHub Release staging, stable launcher activation, health rollback, Compose/systemd deployment, and OpenAPI/golden contracts.
- MusicMate Navidrome link/provider compatibility types.

[Unreleased]: https://github.com/mythezone/navidrome-music-room/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/mythezone/navidrome-music-room/compare/v1.1.0-dev...v1.2.0
[1.1.0-dev]: https://github.com/mythezone/navidrome-music-room/compare/v1.0.0...v1.1.0-dev
[1.0.0]: https://github.com/mythezone/navidrome-music-room/compare/v0.1.0-beta.3...v1.0.0
[0.1.0-beta.3]: https://github.com/mythezone/navidrome-music-room/compare/v0.1.0-beta.2...v0.1.0-beta.3
[0.1.0-beta.2]: https://github.com/mythezone/navidrome-music-room/compare/v0.1.0-beta.1...v0.1.0-beta.2
[0.1.0-beta.1]: https://github.com/mythezone/navidrome-music-room/releases/tag/v0.1.0-beta.1
