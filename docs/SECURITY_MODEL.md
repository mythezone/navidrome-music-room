# Security model

## Protected assets

- Navidrome credentials and OpenSubsonic proofs
- library ACLs and media availability
- room membership and invitation secrets
- authoritative playback/queue state
- updater and plugin pairing keys
- room database and backups

## Primary threats and controls

| Threat | Control |
|---|---|
| Anonymous or removed user joins | Navidrome proof + current plugin allowlist on every session use |
| Client chooses an internal upstream (SSRF) | Gateway has one configured Navidrome URL; client URLs are ignored |
| Invitation leaks into logs | HTTPS link stores secret after `#`; gateway stores only SHA-256 |
| Long-lived token in WebSocket URL | One-use 60-second ticket bound to room and session |
| Member takes over playback | Server-side manager check; optimistic revision check |
| Room grants broader music access | Join and queue folder checks; every client streams with its own account |
| Audio proxy becomes data exfiltration path | Gateway implements no media endpoint |
| Stale or disabled plugin leaves service open | 90-second lease for new work, 60-second existing-session grace |
| Malicious update | Fixed GitHub repository, HTTPS host allowlist, SHA-256, offline Sigstore verification, safe extraction, health rollback |
| Archive traversal or device file | Clean relative paths; regular files/directories only; extraction size limit |
| Database migration failure | Consistent pre-migration and pre-switch backups; launcher restore |
| Secrets in diagnostics | Structured logs omit URL query and authorization; the admin-only opt-in export contains aggregate health and excludes URLs, names, identifiers, invitation data, credentials, and database rows |

## Deployment requirements

- Terminate TLS at a trusted reverse proxy.
- Keep the internal gateway and Navidrome addresses off the public network.
- Use a random pairing token of at least 32 characters and private file permissions.
- Do not enable `MUSIC_ROOM_TRUST_PROXY` unless requests arrive only through your proxy.
- Configure a strict origin allowlist for browser-based clients.
- Run Navidrome and the gateway as the same non-root UID only when sharing a plugin bind mount.
- Back up `room-data` and test restore before production upgrades.

## Known v1 limitations

- In-memory sessions and presence prevent horizontal gateway scaling.
- The private Navidrome rescan/enable API is version-sensitive.
- The gateway validates `getSong` access, but Navidrome's returned song model does not carry a music-folder ID; the client-provided folder is checked against both the room and `getUser` ACL list.
- A compromised client can publish misleading track metadata only until the gateway normalizes it with `getSong`; it cannot obtain media it cannot access.
