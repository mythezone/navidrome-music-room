# Contributing

Thanks for helping make self-hosted listening rooms safer and easier to use.

## Before coding

Open an issue for protocol changes, new permissions, database tables, updater behavior, or licensed-feature boundaries. Small bug fixes and documentation corrections can go directly to a pull request.

Do not add Navidrome filesystem access, media proxying, anonymous sessions, arbitrary upstream URLs, telemetry enabled by default, or a second audio player without an accepted design proposal.

## Development checks

Requirements are Docker and Docker Compose v2. The Make targets pin the required Go/Navidrome toolchains.

```bash
make test
make race
make vet
make validate-openapi
make validate-compose
make validate-plugin
```

Tests should cover both success and denial paths. Permission changes require an end-to-end scenario with at least one administrator and one regular user. Contract changes must update `contracts/openapi.yaml`, golden fixtures, and MusicMate decoders together.

## Pull requests

- Keep changes focused and explain user-visible behavior.
- Add migration and rollback tests for schema changes.
- Never commit real usernames, URLs, pairing tokens, OpenSubsonic proofs, licenses, or database files.
- Use semantic commit messages when practical (`feat:`, `fix:`, `docs:`, `security:`).
- Confirm that media bodies do not appear in gateway tests or captures.

By contributing, you agree that your contribution is licensed under GPL-3.0-only.
