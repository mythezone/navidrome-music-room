# Open-source roadmap

## v0.1 developer preview

- Freeze REST/WebSocket contracts and FAIO compatibility fixtures.
- Build and validate the `.ndp` bridge against Navidrome v0.63.2.
- Complete gateway permission, persistence, queue/playback, invitation, and update tests.
- Land MusicMate provider/link/credential abstractions without regressing FAIO rooms.

## v0.5 beta

- Real Navidrome minimum/latest integration matrix.
- Three-client synchronization, reconnect, transcode, Range, and ACL test harness.
- Embedded React/MUI room management console opened from the stock plugin Website field.
- Web room CRUD, member/invite management, share links, and local QR generation.
- Redacted opt-in diagnostic bundle.
- Prerelease packages for Linux amd64/arm64 and Compose install under ten minutes.

## v1.0 stable

- Signed artifacts, SBOM, provenance, rollback matrix, and restore drill.
- Embedded browser listening room with responsive desktop/mobile layouts,
  invitation login, direct OpenSubsonic media, queue, history, favourites, and lyrics.
- Real dual-browser synchronization, refresh recovery, Range-stream, and
  music-folder ACL isolation acceptance tests.
- Complete installation/upgrade/troubleshooting documentation.
- FAIO regression suite green; no FAIO data import.
- Independent security and GPL compliance review.

## v1.1 preview

- Songs, Albums, and Artists browsing in the Web request desk.
- Whole-album requests using the listener's own music-folder permissions.
- Usage-first release documentation, branding, and ready-to-run Compose bundle.
- Adopt `docs/assets/musicmate-app-icon-source.png` as the canonical icon source
  for the next MusicMate iOS and Android release.
- Continue testing against stock Navidrome without a maintained fork.

## v1.2

- [x] Seven-language Web room and Admin Console with Navidrome/browser detection,
  a persistent switcher, and localized errors.
- [x] Default-on member skipping, contributor self-skip, per-room settings in both
  consoles, and manager previous-track control.
- [x] Real-time chat, member/song mentions, library references, quick reviews,
  replies, recall/moderation, unread badges, and two localized sticker packs.
- [x] Room Information with period statistics and leaderboards, a 24-hour activity
  clock, ten member levels, achievements/badges, and pending/fulfilled song wishes.
- [x] Resizable desktop layout, immersive artwork/lyrics mode, five-band EQ with
  native mobile audio, queue reorder/pin/shuffle/random fill, paginated history
  re-queue, and Songs/Albums/Artists/Playlists favorites.
- [x] Web signed upgrade, rollback, diagnostics, and automatic same-origin
  Navidrome plugin rescan/re-enable coordination.
- [x] Scheduled SemVer update detection, admin-only WebSocket notices, optional
  webhook push, and plugin heartbeat logging.
- [x] v1.2.0 contracts and SQLite migration v5 for collaborative/community data.

## Later

- Promote the console to a real Navidrome sidebar/player panel only after an official host extension point exists.
- Continue accessibility and richer room-activity improvements through public contributions.
- Add native song and album context actions if Navidrome publishes a supported resource-action extension point.
- Multi-node coordination only if real deployments justify Redis/consensus complexity.
