# Roadmap

## v0.1 developer preview

- Freeze REST/WebSocket contracts and FAIO compatibility fixtures.
- Build and validate the `.ndp` bridge against Navidrome v0.63.2.
- Complete gateway permission, persistence, queue/playback, invitation, and update tests.
- Land MusicMate provider/link/credential abstractions without regressing FAIO rooms.

## v0.5 beta

- Real Navidrome minimum/latest integration matrix.
- Three-client synchronization, reconnect, transcode, Range, and ACL test harness.
- App room creation/share/QR screens and local QR generation.
- Redacted opt-in diagnostic bundle.
- Prerelease packages for Linux amd64/arm64 and Compose install under ten minutes.

## v1.0 stable

- Signed artifacts, SBOM, provenance, rollback matrix, and restore drill.
- Complete installation/upgrade/troubleshooting documentation.
- FAIO regression suite green; no FAIO data import.
- Independent security and GPL/commercial-extension legal review.

## Later

- React/Material UI room panel only if Navidrome provides official UI/player extension points.
- Separately distributed licensed features after product, security, and legal review.
- Multi-node coordination only if real deployments justify Redis/consensus complexity.
