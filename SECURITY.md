# Security policy

## Supported versions

Until v1.0, only the newest prerelease receives security fixes. After v1.0, the latest minor release and the previous minor release will receive critical fixes.

## Reporting a vulnerability

Do not open a public issue for authentication bypasses, invitation leaks, SSRF, path traversal, update-signature failures, ACL bypasses, or credential exposure.

Use GitHub's **Security → Report a vulnerability** private reporting form for this repository. Include:

- affected version/commit and deployment type;
- reproduction steps with secrets removed;
- expected and observed result;
- impact and any known exploitation;
- a proposed fix, if available.

We aim to acknowledge a report within 3 business days and provide an initial severity assessment within 7 business days. Timelines may change for complex coordinated disclosure. Reporters may request credit or anonymity.

## Scope

In scope: gateway API, `.ndp` bridge, launcher/updater, official Compose/systemd files, and the MusicMate Navidrome provider contract.

Out of scope: vulnerabilities in an unmodified Navidrome installation, brute force that ignores published rate limits, unsupported multi-gateway deployments, social engineering, and attacks requiring prior root access to the host.

See [docs/SECURITY_MODEL.md](docs/SECURITY_MODEL.md) for trust boundaries and known limitations.
