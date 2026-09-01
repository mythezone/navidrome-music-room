# systemd installation

Use the same signed Linux release bundle published for Docker deployments. The
launcher itself stays at a stable path while its bootstrap gateway, plugin, and
metadata live below `release/`.

```bash
sudo install -d -m 0755 /opt/navidrome-music-room/release
stage=$(mktemp -d)
tar -xzf navidrome-music-room-linux-amd64.tar.gz -C "$stage"
sudo install -m 0755 "$stage/music-room-launcher" /opt/navidrome-music-room/music-room-launcher
sudo install -m 0755 "$stage/music-room-gateway" /opt/navidrome-music-room/release/music-room-gateway
sudo install -m 0644 "$stage/navidrome-music-room.ndp" /opt/navidrome-music-room/release/navidrome-music-room.ndp
sudo install -m 0644 "$stage/release.json" /opt/navidrome-music-room/release/release.json
```

For arm64, use the `linux-arm64` bundle. Verify `checksums.txt` and the Sigstore
bundles from the same GitHub Release before the initial install.

Install and edit the service configuration:

```bash
sudo install -d -m 0750 /etc/navidrome-music-room
sudo install -m 0640 deploy/systemd/gateway.env.example /etc/navidrome-music-room/gateway.env
sudo install -m 0644 deploy/systemd/navidrome-music-room.service /etc/systemd/system/navidrome-music-room.service
sudo install -d -o navidrome -g navidrome -m 0700 /var/lib/navidrome/plugins/navidrome-music-room/room-data
sudo systemctl daemon-reload
sudo systemctl enable --now navidrome-music-room
```

Set both public HTTPS URLs, the internal Navidrome URL, pairing token, plugin
directory, and release repository in `gateway.env` before starting. Copy the
same pairing token into the Navidrome plugin settings, authorize users, and
enable the plugin. The service sandbox grants write access only to the shared
Navidrome plugin directory.

The automatic updater writes versioned releases below `room-data/releases/`;
the stable `/opt/navidrome-music-room/music-room-launcher` supervises atomic
switches and rollback. Uninstalling the unit does not remove `room-data`.
