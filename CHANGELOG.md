# Changelog

All notable changes follow [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and Semantic Versioning.

## [Unreleased]

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
- Consistent `402 feature_locked` commercial boundaries.
- Signed GitHub Release staging, stable launcher activation, health rollback, Compose/systemd deployment, and OpenAPI/golden contracts.
- MusicMate Navidrome link/provider compatibility types.

[Unreleased]: https://github.com/mythezone/navidrome-music-room/compare/v0.1.0-beta.2...HEAD
[0.1.0-beta.2]: https://github.com/mythezone/navidrome-music-room/compare/v0.1.0-beta.1...v0.1.0-beta.2
[0.1.0-beta.1]: https://github.com/mythezone/navidrome-music-room/releases/tag/v0.1.0-beta.1
