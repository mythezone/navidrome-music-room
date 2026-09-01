# Pinned Sigstore trusted root

`trusted_root.json` is the Sigstore public-good instance trusted root bundled
with each Music Room release so update verification does not need a writable
home directory or network access to Sigstore services.

- TUF target metadata version: 14
- Length: 6787 bytes
- SHA-256: `6494e21ea73fa7ee769f85f57d5a3e6a08725eae1e38c755fc3517c9e6bc0b66`
- Refreshed: 2026-09-01

It was obtained by Cosign v3.1.2 from
`https://tuf-repo-cdn.sigstore.dev` using Cosign's embedded initial TUF root.
The target length and digest above match the signed `targets.json` metadata.
After refreshing this file, verify an actual project release with Cosign using
`--network none`, `--trusted-root`, and the release workflow's exact identity
before committing the change.
