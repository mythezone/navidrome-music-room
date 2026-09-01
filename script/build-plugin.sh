#!/bin/sh
set -eu

version=${1:-0.1.0-dev}
root_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
dist_dir=${DIST_DIR:-$root_dir/dist}

command -v tinygo >/dev/null 2>&1 || { echo "tinygo is required" >&2; exit 1; }
command -v zip >/dev/null 2>&1 || { echo "zip is required" >&2; exit 1; }

mkdir -p "$dist_dir/plugin"
cd "$root_dir/plugin"
GOFLAGS="${GOFLAGS:+$GOFLAGS }-buildvcs=false" tinygo build -target wasip1 -buildmode=c-shared -ldflags="-X main.version=$version" -o "$dist_dir/plugin/plugin.wasm" .
sed "s/\"version\": \"0.1.0-dev\"/\"version\": \"$version\"/" manifest.json > "$dist_dir/plugin/manifest.json"
cd "$dist_dir/plugin"
zip -q -9 -FS "$dist_dir/navidrome-music-room.ndp" manifest.json plugin.wasm
chmod 0644 "$dist_dir/navidrome-music-room.ndp"
