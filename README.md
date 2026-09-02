<p align="center">
  <img src="docs/assets/logo.png" width="148" alt="Navidrome Music Room logo" />
</p>

<h1 align="center">Navidrome Music Room</h1>

<p align="center"><strong>Turn your Navidrome library into a private, synchronized listening room.</strong></p>

<p align="center">
  <a href="https://github.com/mythezone/navidrome-music-room/releases/tag/v1.1.0-dev"><img alt="Version" src="https://img.shields.io/badge/version-v1.1.0--dev-ff6b57"></a>
  <a href="https://github.com/mythezone/navidrome-music-room/releases"><img alt="GitHub release" src="https://img.shields.io/github/v/release/mythezone/navidrome-music-room?include_prereleases&sort=semver"></a>
  <a href="https://github.com/mythezone/navidrome-music-room/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/mythezone/navidrome-music-room/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://github.com/mythezone/navidrome-music-room/actions/workflows/codeql.yml"><img alt="CodeQL" src="https://github.com/mythezone/navidrome-music-room/actions/workflows/codeql.yml/badge.svg"></a>
  <a href="LICENSE"><img alt="GPL-3.0" src="https://img.shields.io/badge/license-GPL--3.0-blue"></a>
  <a href="https://www.navidrome.org/"><img alt="Navidrome 0.63.2+" src="https://img.shields.io/badge/Navidrome-0.63.2%2B-00a4dc"></a>
  <a href="https://opensubsonic.netlify.app/"><img alt="OpenSubsonic" src="https://img.shields.io/badge/API-OpenSubsonic-6e56cf"></a>
  <img alt="Docker Compose" src="https://img.shields.io/badge/install-Docker%20Compose-2496ed">
  <img alt="Linux amd64 and arm64" src="https://img.shields.io/badge/Linux-amd64%20%7C%20arm64-fcc624">
  <a href="CONTRIBUTING.md"><img alt="PRs welcome" src="https://img.shields.io/badge/PRs-welcome-brightgreen"></a>
</p>

<p align="center"><a href="README.zh-CN.md">简体中文</a> · <a href="https://github.com/mythezone/navidrome-music-room/releases">Download</a> · <a href="#quick-install">Install</a> · <a href="#use-the-room">Use</a></p>

![Navidrome Music Room across desktop and mobile](docs/assets/hero.png)

An administrator creates a room from Navidrome's plugin page, shares a link or QR code, and invited Navidrome users listen together in a desktop or mobile browser. Everyone uses their own Navidrome account and library permissions. Music is streamed directly from Navidrome while the room keeps play, pause, seek, next track, queue and reconnect state synchronized.

Current prerelease: **v1.1.0-dev**. It adds first-class **Songs / Albums / Artists** browsing and whole-album requests to the Web room.

## Where it is useful

- Listen to an album with friends in different places while everyone stays on the same second.
- Let family members use one self-hosted library without sharing an administrator password.
- Run a private party or team room where members can request tracks and the owner controls playback.
- Share a room from Navidrome now, then open the same invitation in the upcoming MusicMate app.

## What works

| Experience | v1.1.0-dev |
|---|---|
| Create, edit, close, reopen and delete rooms | ✅ |
| Share HTTPS invite, MusicMate deep link and local QR code | ✅ |
| Invite redemption, persistent membership and member removal | ✅ |
| Browser playback on desktop and mobile | ✅ |
| Synchronized play, pause, seek, next track and reconnect | ✅ |
| Browse and request songs, albums and artists | ✅ |
| Request a whole album | ✅ |
| Search, favourites, playlists, cover art and lyrics | ✅ |
| Direct Navidrome streaming with each user's own access | ✅ |
| Chat, richer room activity and community statistics | Open-source roadmap |

The management console is opened from the stock Navidrome plugin **Website** link:

![Room management opened from Navidrome](docs/assets/admin-ui-live.png)

It creates invitation links and QR codes locally in the browser:

![Share link and QR dialog](docs/assets/share-dialog-live.png)

The same invitation opens the full Web room:

![Desktop Web listening room](docs/assets/web-room-live.png)

Songs, albums and artists now use Navidrome's own OpenSubsonic library APIs:

![Songs, albums and artists in the request desk](docs/assets/web-room-catalog-live.png)

<img src="docs/assets/web-room-mobile.png" width="390" alt="Mobile Web listening room" />

## Downloads

The [v1.1.0-dev release](https://github.com/mythezone/navidrome-music-room/releases/tag/v1.1.0-dev) contains everything needed to install without compiling:

| File | Use |
|---|---|
| [`navidrome-music-room-compose-1.1.0-dev.tar.gz`](https://github.com/mythezone/navidrome-music-room/releases/download/v1.1.0-dev/navidrome-music-room-compose-1.1.0-dev.tar.gz) | Recommended ready-to-run Navidrome + gateway Compose kit |
| [`navidrome-music-room.ndp`](https://github.com/mythezone/navidrome-music-room/releases/download/v1.1.0-dev/navidrome-music-room.ndp) | Navidrome plugin package for an existing installation |
| [`navidrome-music-room-linux-amd64.tar.gz`](https://github.com/mythezone/navidrome-music-room/releases/download/v1.1.0-dev/navidrome-music-room-linux-amd64.tar.gz) | Gateway and launcher for Linux x86-64 |
| [`navidrome-music-room-linux-arm64.tar.gz`](https://github.com/mythezone/navidrome-music-room/releases/download/v1.1.0-dev/navidrome-music-room-linux-arm64.tar.gz) | Gateway and launcher for Linux ARM64 |

Checksums, SPDX SBOMs, provenance and Sigstore bundles are attached to the same release.

## Quick install

Requirements: Linux amd64/arm64, Docker Engine, Docker Compose v2 and writable Navidrome data, music and plugin directories.

```bash
curl -LO https://github.com/mythezone/navidrome-music-room/releases/download/v1.1.0-dev/navidrome-music-room-compose-1.1.0-dev.tar.gz
tar -xzf navidrome-music-room-compose-1.1.0-dev.tar.gz
cd navidrome-music-room-1.1.0-dev
cp .env.example .env
```

Edit `.env`. For a LAN test on `192.168.1.20:1970`:

```dotenv
PUID=1000
PGID=1000
NAVIDROME_BIND_ADDRESS=0.0.0.0
NAVIDROME_PORT=1970
NAVIDROME_PUBLIC_URL=http://192.168.1.20:1970
MUSIC_ROOM_PUBLIC_URL=http://192.168.1.20:1970/music-room
MUSIC_ROOM_ALLOWED_ORIGINS=http://192.168.1.20:1970
MUSIC_LIBRARY_PATH=/srv/music
NAVIDROME_DATA_PATH=/srv/navidrome/data
NAVIDROME_PLUGINS_PATH=/srv/navidrome/plugins
MUSIC_ROOM_PLUGIN_PAIRING_TOKEN=replace-with-at-least-32-random-characters
```

Use the real IP or hostname seen by every listener. LAN HTTP is suitable for local testing; use one HTTPS origin before exposing the service to the internet.

Start it:

```bash
./install.sh "$PWD"
docker compose ps
```

The installer creates a random pairing token when needed. The gateway launcher copies `navidrome-music-room.ndp` into the shared plugin directory automatically. Open `http://192.168.1.20:1970`, create the first Navidrome administrator, and let Navidrome finish scanning the music directory.

### Use an existing Navidrome library

Back up Navidrome first, stop its old container, and point the release kit at the same host directories:

```dotenv
NAVIDROME_DATA_PATH=/path/to/existing/navidrome-data
MUSIC_LIBRARY_PATH=/path/to/existing/music
NAVIDROME_PLUGINS_PATH=/path/to/existing/navidrome-plugins
```

The Compose file mounts those paths as `/data`, `/music` and `/plugins`. It does not import or rewrite the Navidrome database. Room data is created separately below:

```text
/path/to/existing/navidrome-plugins/navidrome-music-room/room-data/
```

For a customized Compose stack, copy the `music-room-gateway` service and the `/music-room/` reverse-proxy rule from the release kit, then make these Navidrome settings available:

```yaml
environment:
  ND_PLUGINS_ENABLED: "true"
  ND_PLUGINS_FOLDER: /plugins
  ND_PLUGINS_AUTORELOAD: "true"
volumes:
  - /path/to/existing/navidrome-plugins:/plugins
```

The `.ndp` package does not listen for browser or WebSocket traffic by itself, so the local gateway is required. This is why the Compose kit is the recommended download.

## Configure the plugin in Navidrome

1. Open **Settings → Plugins → Navidrome Music Room**.
2. Set **Navidrome internal URL** to `http://navidrome:4533` for the bundled Compose network.
3. Set **Navidrome public URL** to the address listeners use, such as `https://music.example.com`.
4. Set **Gateway internal URL** to `http://music-room-gateway:4534`.
5. Set **Gateway public URL** to `https://music.example.com/music-room`.
6. Paste the pairing token printed by `install.sh` or stored in `.env`.
7. Select the Navidrome users allowed to use listening rooms, save, and enable the plugin.
8. Wait up to 30 seconds, then click the plugin's **Website** link to open the room console.

The gateway and Navidrome should share one public origin (`/` and `/music-room/`). This lets browsers authenticate and stream without cross-origin credential exceptions.

## Use the room

### Create and share

1. Sign in to Navidrome as an administrator.
2. Open the Music Room plugin and click **Website**.
3. Create a room, choose the allowed music folders and save it.
4. Create an invitation and choose **Copy link**, **QR code**, or **MusicMate link**.
5. The recipient opens the link, signs in with their own Navidrome account and redeems the invitation.

### Request music

Open **Request music** inside the room and select one of the three library views:

- **Songs** loads a usable selection from Navidrome and requests a track immediately.
- **Albums** shows the newest albums. Open an album to pick a song, or use **Request whole album**.
- **Artists** uses Navidrome's artist index, then opens that artist's albums and songs.
- **Search** returns matching songs, albums and artists and keeps the same three tabs.

These views call the standard OpenSubsonic library methods (`getRandomSongs`, `getAlbumList2`, `getArtists`, `getArtist`, `getAlbum` and `search3`). The room only submits Navidrome IDs to the shared queue; it does not duplicate or scrape Navidrome's database.

### Listen together

Each browser must click **Start listening** once because of browser autoplay rules. After that, the room owner or a Navidrome administrator controls shared play, pause, seek and next-track state. All clients calculate position from the gateway's authoritative timestamp, repair drift, preload the next track and recover the latest snapshot after refresh or network interruption.

## Can it add buttons to native song and album pages?

Not with the official `.ndp` API available in Navidrome v0.63.2. The supported plugin manifest can expose configuration and a Website link, but it has no song/album context-action, custom route, sidebar or player-control extension point. Adding a native **Request to room** button today would require a Navidrome frontend patch or an injected browser script, both of which would make upgrades fragile.

This project stays compatible with stock Navidrome: the embedded room page reuses the same OpenSubsonic queries and presents Songs, Albums and Artists itself. If Navidrome adds an official resource-action API, native buttons can be added without changing the room protocol. See the [official plugin documentation](https://www.navidrome.org/docs/usage/features/plugins/).

## Accounts, permissions and privacy

- There are no anonymous rooms. Every listener must be an authorized Navidrome user.
- Only a Navidrome administrator can create a room. The owner and administrators manage shared playback and members.
- Invite tokens are random and only their SHA-256 digests are stored.
- Music-folder access is checked when joining and requesting music. A room never borrows its owner's permissions.
- Passwords are sent only to the same-origin Navidrome login endpoint. The gateway receives a short-lived OpenSubsonic proof, not the raw password.
- Audio, artwork and lyrics go directly from Navidrome to each listener; the room gateway never proxies media bytes.
- No telemetry is sent by default.

Read [SECURITY.md](SECURITY.md) before an internet-facing deployment.

## Data, backup and removal

All room-owned files live outside Navidrome's database:

```text
${Plugins.Folder}/navidrome-music-room/room-data/
├── rooms.sqlite3
├── secrets/
├── backups/
├── releases/
└── logs/
```

Back up that directory together with Navidrome. Removing the container or `.ndp` intentionally keeps room data; stop the gateway and delete this exact directory only when you want a permanent removal.

## Update and troubleshooting

Use **Check for updates → Upgrade now** in the administrator console. The signed updater downloads the matching release bundle, and the launcher backs up room data, switches the gateway and `.ndp` together, checks health, and restores the previous release if activation fails. For a manual/offline upgrade, follow [the update guide](docs/UPDATES.md) instead of replacing only the `.ndp`.

- **Plugin is missing:** confirm `ND_PLUGINS_ENABLED=true`, the `/plugins` mount, and that `navidrome-music-room.ndp` exists there; then rescan plugins.
- **Plugin lease expired:** verify the internal gateway URL, pairing token, enabled state and authorized-user selection; wait one heartbeat interval.
- **No songs or albums:** update to v1.1.0-dev, grant the room's music-folder access, and reopen **Request music**.
- **Browser is silent:** click **Start listening** once and inspect the direct Navidrome `stream` request.
- **WebSocket disconnects:** confirm the reverse proxy forwards `Upgrade` and `Connection` for `/music-room/`.

More details: [compatibility](docs/COMPATIBILITY.md), [updates](docs/UPDATES.md), [MusicMate integration](docs/MUSICMATE_INTEGRATION.md), and the [API contract](contracts/openapi.yaml).

## MusicMate app preview

MusicMate is the upcoming native companion for iOS and Android. It will scan the same room QR code, keep Navidrome credentials in the system credential store, use Navidrome as its music source and join the exact same synchronized queue as Web listeners. The Web room remains fully usable without the app.

![MusicMate app preview](docs/assets/musicmate-demo.gif)

## Work with us

The project is open source and welcomes code, translations, testing, UI ideas and deployment reports.

- Bugs and feature proposals: [GitHub Issues](https://github.com/mythezone/navidrome-music-room/issues)
- Pull requests: read [CONTRIBUTING.md](CONTRIBUTING.md) and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- Product, integration or community collaboration: [mythezone@gmail.com](mailto:mythezone@gmail.com)
- Security reports: use GitHub's private vulnerability reporting form described in [SECURITY.md](SECURITY.md)

The source is available under GPL-3.0-only. See [LICENSE](LICENSE) and [NOTICE](NOTICE).

## Buy me a coffee

If Music Room is useful to you, a coffee helps keep testing devices, releases and documentation moving. The project remains open source.

<table>
  <tr><th>Alipay</th><th>WeChat Pay</th></tr>
  <tr>
    <td><img src="docs/assets/donate-alipay.jpg" width="300" alt="Alipay donation QR code" /></td>
    <td><img src="docs/assets/donate-wechat.jpg" width="300" alt="WeChat Pay donation QR code" /></td>
  </tr>
</table>
