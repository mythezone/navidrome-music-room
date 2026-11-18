# Signed update and rollback protocol

Production updates never execute source-control commands.

## Web direct upgrade

Sign in as a Navidrome administrator, then open **Settings → Plugins → Navidrome
Music Room → Website → Upgrade now** (`/music-room/admin/`). Use **Check for
updates** first if no release has been detected. Automatic installation and
rollback require the stable launcher used by the supplied Compose and systemd
deployments; the console shows manual instructions for an unmanaged gateway.

The console stages and installs the signed release, waits for the target gateway
version at `/readyz`, then uses the current browser's Navidrome UI JWT to call
same-origin `POST /api/plugin/rescan` and `GET /api/plugin`. If Music Room is
disabled, it sends `PUT /api/plugin/{id}` with `{"enabled":true}`. These calls use
`X-ND-Authorization`, reject redirects, and never send the Navidrome UI JWT to the
gateway. Upgrade completion requires `/readyz` to report the target gateway and
plugin versions with a paired heartbeat. Rollback uses the same rescan/enable and
readiness coordination for the previous version.

If the Navidrome browser login has expired or the plugin cannot be found, the
console reports that step's error and offers **Rescan & enable plugin in
Navidrome** after the login or installation is repaired. The console also exposes
rollback availability and a redacted JSON diagnostics download.

## Scheduled detection and notifications

| Gateway setting | Default | Behavior |
|---|---|---|
| `MUSIC_ROOM_AUTO_UPDATE_CHECK` | `true` | Check after about 25 seconds at startup, then periodically; setting `false` disables the timer. |
| `MUSIC_ROOM_UPDATE_CHECK_INTERVAL` | `6h` | Positive Go duration between scheduled checks, for example `30m` or `12h`. |
| `MUSIC_ROOM_UPDATE_WEBHOOK_URL` | Empty | Optional HTTP(S) endpoint receiving a JSON POST when a newer release is detected. |

Set these in the Compose `.env` or the systemd gateway environment and restart
the gateway to apply them. The plugin's `stable`/`beta` channel selects eligible
GitHub Releases. Changing an established plugin channel also queues a check, even
when the timer is disabled. SemVer ordering determines whether a release is newer;
checking never installs it automatically.

A newly detected version is logged and sent as an `update_available` WebSocket
event only to authorized Navidrome administrators. Room snapshots also include
the available-update notice for administrators, so reconnecting clients recover
their badge. The optional webhook body contains `event`, `repository`,
`currentVersion`, `latestVersion`, `channel`, `releaseURL`, and `publishedAt`.
Notifications are deduplicated by the last notified version in SQLite. Webhook
requests have a ten-second timeout and do not follow redirects; failed deliveries
are logged and are not retried for that already-recorded version.

Plugin-sync responses include `updateAvailable`, `latestVersion`, and
`releaseURL`. The `.ndp` heartbeat logs each newly seen version in Navidrome and
persists its last logged version in KV storage. These notifications contain
release metadata, not listener credentials.

## Verification and activation

1. `check` reads GitHub Releases from the configured fixed repository. The default `stable` channel accepts only the latest non-draft, non-prerelease release; the opt-in `beta` channel accepts the newest non-draft release, including a prerelease.
2. `stage` selects the exact `linux-{GOARCH}` archive, `checksums.txt`, the matching SPDX SBOM and in-toto/SLSA provenance statement, plus a Sigstore bundle for each document.
3. The downloader accepts HTTPS GitHub/GitHubusercontent hosts only, limits sizes, and writes `.part` files with mode `0600`.
4. The gateway uses the Cosign v3 binary and TUF-authenticated Sigstore trusted root from its currently active, previously verified release. This avoids host dependencies, runtime TUF downloads, and writable-HOME requirements.
5. Cosign first verifies `checksums.txt`, then SHA-256 must match its signed entry for the selected archive.
6. Cosign verifies the archive, SBOM, and provenance without network access against the pinned trusted root, GitHub Actions OIDC issuer, and configured release-workflow identity.
7. The updater validates the signed SPDX inventory and requires signed provenance whose subject name and SHA-256 exactly match the selected archive, repository, tag, and `.github/workflows/release.yml` source.
8. Tar extraction rejects absolute paths, traversal, links, devices, and more than 512 MiB expanded data.
9. The bundle must contain `music-room-gateway`, architecture-matched `cosign`, `sigstore-trusted-root.json`, `navidrome-music-room.ndp`, and `release.json`; their post-extraction digests are recorded in the private pending state.
10. `install` writes `pending-update.json` and asks the launcher for a graceful restart (exit code 75).
11. Before activation, the launcher re-hashes all five staged files to reject modification after signature verification.
12. The launcher backs up SQLite, atomically switches release symlinks and the `.ndp`, starts the new gateway with the active verifier paths, and polls `/healthz`.
13. A failed start restores the previous release, plugin package, verifier, trusted root, and matching database backup. Only after health succeeds is the activation atomically promoted to the rollback record.
14. Manual rollback is offered only when the current/previous release links and a verified pre-activation SQLite backup still form one matching set. The launcher restores that database together with the old release and plugin, while retaining a forward-recovery backup if the rolled-back process fails health checks.

The Admin Web UI performs the browser coordination described above. MusicMate's
`NavidromeUpdateCoordinator` obtains a UI JWT from `/auth/login` and calls its
v0.63.2 adapter's rescan/enable endpoints. Both clients confirm the target gateway
and `/readyz.pluginVersion` before declaring the upgrade complete.

Release CI publishes SHA-256 checksums, Sigstore bundles, SBOMs, and GitHub provenance attestations. Stable releases are immutable; corrections use a new patch version.

`v0.1.0-beta.1` predates the self-contained verifier and cannot reliably stage
its successor in a locked-down non-root runtime. Existing beta.1 installations
must install the beta.2 image or signed bundle once; one-click updates are
self-contained from beta.2 onward.
