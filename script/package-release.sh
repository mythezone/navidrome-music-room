#!/bin/sh
set -eu

arch=${1:?usage: package-release.sh amd64|arm64 VERSION}
version=${2:?usage: package-release.sh amd64|arm64 VERSION}
case "$arch" in
  amd64|arm64) ;;
  *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac

root_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
dist_dir=${DIST_DIR:-$root_dir/dist}
test -f "$dist_dir/navidrome-music-room.ndp" || { echo "build the plugin first" >&2; exit 1; }

stage_dir=$(mktemp -d)
trap 'rm -rf "$stage_dir"' EXIT INT TERM

cd "$root_dir/gateway"
CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -buildvcs=false -trimpath -ldflags='-s -w' -o "$stage_dir/music-room-gateway" ./cmd/music-room-gateway
CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -buildvcs=false -trimpath -ldflags='-s -w' -o "$stage_dir/music-room-launcher" ./cmd/music-room-launcher
cp "$dist_dir/navidrome-music-room.ndp" "$stage_dir/navidrome-music-room.ndp"
sed "s/0.1.0-dev/$version/" "$root_dir/release.json" > "$stage_dir/release.json"
chmod 0755 "$stage_dir/music-room-gateway" "$stage_dir/music-room-launcher"
chmod 0644 "$stage_dir/navidrome-music-room.ndp" "$stage_dir/release.json"
tar -C "$stage_dir" -czf "$dist_dir/navidrome-music-room-linux-$arch.tar.gz" \
  music-room-gateway music-room-launcher navidrome-music-room.ndp release.json
chmod 0644 "$dist_dir/navidrome-music-room-linux-$arch.tar.gz"
