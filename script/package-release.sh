#!/bin/sh
set -eu

arch=${1:?usage: package-release.sh amd64|arm64 VERSION}
version=${2:?usage: package-release.sh amd64|arm64 VERSION}
version=${version#v}
case "$arch" in
  amd64|arm64) ;;
  *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac

root_dir=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
dist_dir=${DIST_DIR:-$root_dir/dist}
test -f "$dist_dir/navidrome-music-room.ndp" || { echo "build the plugin first" >&2; exit 1; }
cosign_binary=${COSIGN_BINARY:?set COSIGN_BINARY to the pinned linux/$arch Cosign executable}
trusted_root=${SIGSTORE_TRUSTED_ROOT:-$root_dir/deploy/sigstore/trusted_root.json}
test -f "$cosign_binary" || { echo "Cosign executable not found: $cosign_binary" >&2; exit 1; }
test -f "$trusted_root" || { echo "Sigstore trusted root not found: $trusted_root" >&2; exit 1; }

stage_dir=$(mktemp -d)
trap 'rm -rf "$stage_dir"' EXIT INT TERM

cd "$root_dir/gateway"
CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -buildvcs=false -trimpath -ldflags='-s -w' -o "$stage_dir/music-room-gateway" ./cmd/music-room-gateway
CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -buildvcs=false -trimpath -ldflags='-s -w' -o "$stage_dir/music-room-launcher" ./cmd/music-room-launcher
cp "$dist_dir/navidrome-music-room.ndp" "$stage_dir/navidrome-music-room.ndp"
cp "$cosign_binary" "$stage_dir/cosign"
cp "$trusted_root" "$stage_dir/sigstore-trusted-root.json"
sed "s/1.1.0-dev/$version/" "$root_dir/release.json" > "$stage_dir/release.json"
chmod 0755 "$stage_dir/music-room-gateway" "$stage_dir/music-room-launcher" "$stage_dir/cosign"
chmod 0644 "$stage_dir/sigstore-trusted-root.json" "$stage_dir/navidrome-music-room.ndp" "$stage_dir/release.json"
tar -C "$stage_dir" -czf "$dist_dir/navidrome-music-room-linux-$arch.tar.gz" \
  music-room-gateway music-room-launcher cosign sigstore-trusted-root.json navidrome-music-room.ndp release.json
chmod 0644 "$dist_dir/navidrome-music-room-linux-$arch.tar.gz"
