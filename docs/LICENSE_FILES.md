# Offline License verification

The community gateway contains only an entitlement boundary and an offline verifier. It does not contain implementations of chat, stickers, VIP, statistics, rankings, or achievements. A valid License therefore proves claims but does not make an absent feature executable.

## Location and key

The Navidrome plugin setting `license_file` is a path relative to the gateway `room-data` directory. An empty setting resolves to `secrets/license.json`. Absolute paths, traversal, symlinks escaping `room-data`, non-regular files, and files larger than 64 KiB are rejected.

Set `MUSIC_ROOM_LICENSE_PUBLIC_KEY` on the gateway to the issuer's base64-encoded Ed25519 public key (or a PEM PKIX public key). This is public verification material, not a secret. Verification performs no network request.

## Envelope

```json
{
  "format": "musicmate-license/v1",
  "payload": "BASE64URL_WITHOUT_PADDING_OF_CLAIMS_JSON",
  "signature": "BASE64URL_WITHOUT_PADDING_OF_ED25519_SIGNATURE"
}
```

The Ed25519 signature covers the exact decoded `payload` bytes. Claims use RFC 3339 timestamps:

```json
{
  "licenseID": "license_...",
  "subject": "customer-or-instance-reference",
  "issuedAt": "2026-09-01T00:00:00Z",
  "notBefore": "2026-09-01T00:00:00Z",
  "expiresAt": "2027-09-01T00:00:00Z",
  "features": ["chat", "statistics"]
}
```

Unknown or duplicate feature keys are rejected. `/api/v1/entitlements` reports `not_installed`, `verification_unavailable`, `invalid`, `not_yet_valid`, `expired`, or `valid`, plus non-sensitive claim metadata. Effective commercial feature flags remain `false` until a separately distributed, version-compatible extension is present.
