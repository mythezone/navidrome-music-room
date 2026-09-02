#!/bin/sh
set -eu

version_arg=${1:?usage: package-compose-release.sh VERSION [PLUGIN_PATH] [OUTPUT_DIR]}
version=${version_arg#v}
case "$version_arg" in
  v*) image_tag=$version_arg ;;
  *) image_tag=v$version ;;
esac

root_dir=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
plugin_path=${2:-$root_dir/dist/navidrome-music-room.ndp}
output_dir=${3:-$root_dir/dist}
test -f "$plugin_path" || { echo "plugin not found: $plugin_path" >&2; exit 1; }

stage_dir=$(mktemp -d)
trap 'rm -rf "$stage_dir"' EXIT INT TERM
bundle_dir=$stage_dir/navidrome-music-room-$version
mkdir -p "$bundle_dir/deploy" "$bundle_dir/docs/assets"

sed \
  -e "s/__MUSIC_ROOM_IMAGE_TAG__/$image_tag/g" \
  -e "s/__MUSIC_ROOM_VERSION__/$version/g" \
  "$root_dir/deploy/compose/compose.release.yaml" > "$bundle_dir/compose.yaml"
sed \
  -e "s|^MUSIC_ROOM_IMAGE=.*|MUSIC_ROOM_IMAGE=ghcr.io/mythezone/navidrome-music-room:$image_tag|" \
  "$root_dir/.env.example" > "$bundle_dir/.env.example"

cp "$root_dir/deploy/edge-nginx.conf" "$bundle_dir/deploy/edge-nginx.conf"
cp "$root_dir/deploy/compose/install.sh" "$bundle_dir/install.sh"
cp "$root_dir/README.md" "$root_dir/README.zh-CN.md" "$root_dir/LICENSE" "$root_dir/NOTICE" "$bundle_dir/"
cp "$plugin_path" "$bundle_dir/navidrome-music-room.ndp"
cp "$root_dir/docs/assets/"* "$bundle_dir/docs/assets/"
chmod 0755 "$bundle_dir/install.sh"
chmod 0644 "$bundle_dir/compose.yaml" "$bundle_dir/.env.example" "$bundle_dir/navidrome-music-room.ndp"

mkdir -p "$output_dir"
archive=$output_dir/navidrome-music-room-compose-$version.tar.gz
tar -C "$stage_dir" -czf "$archive" "navidrome-music-room-$version"
chmod 0644 "$archive"
echo "$archive"
