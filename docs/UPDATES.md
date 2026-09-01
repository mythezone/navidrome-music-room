# Signed update and rollback protocol

Production updates never execute source-control commands.

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

MusicMate's `NavidromeUpdateCoordinator` then waits for the target gateway version, obtains a UI JWT from `/auth/login`, and calls the v0.63.2 adapter's rescan/enable endpoints. The JWT never leaves that operation. Upgrade completion means both the discovery/health version and `/readyz.pluginVersion` match the target release and the heartbeat is paired.

Release CI publishes SHA-256 checksums, Sigstore bundles, SBOMs, and GitHub provenance attestations. Stable releases are immutable; corrections use a new patch version.

`v0.1.0-beta.1` predates the self-contained verifier and cannot reliably stage
its successor in a locked-down non-root runtime. Existing beta.1 installations
must install the beta.2 image or signed bundle once; one-click updates are
self-contained from beta.2 onward.
