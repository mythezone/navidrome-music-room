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
- Continue testing against stock Navidrome without a maintained fork.

## Later

- Promote the console to a real Navidrome sidebar/player panel only after an official host extension point exists.
- Add chat, richer room activity, accessibility improvements, and community statistics through public contributions.
- Add native song and album context actions if Navidrome publishes a supported resource-action extension point.
- Multi-node coordination only if real deployments justify Redis/consensus complexity.
