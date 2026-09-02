#!/bin/sh
set -eu

project_dir=${1:-$(pwd)}
umask 077

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker is required." >&2
  exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "Docker Compose v2 is required." >&2
  exit 1
fi

mkdir -p "$project_dir/data/navidrome" "$project_dir/data/plugins/navidrome-music-room/room-data" "$project_dir/music"
if [ ! -f "$project_dir/.env" ]; then
  cp "$project_dir/.env.example" "$project_dir/.env"
  token=$(openssl rand -hex 32)
  sed -i "s/replace-with-at-least-32-random-characters/$token/" "$project_dir/.env"
  echo "Created .env. Set NAVIDROME_PUBLIC_URL and MUSIC_ROOM_PUBLIC_URL before exposing the services."
fi

puid=${PUID:-$(id -u)}
pgid=${PGID:-$(id -g)}
chown -R "$puid:$pgid" "$project_dir/data"
docker compose --project-directory "$project_dir" up -d --build

echo "Gateway pairing token:"
docker compose --project-directory "$project_dir" exec -T music-room-gateway \
  sh -c 'test -r /plugins/navidrome-music-room/room-data/secrets/plugin-pairing-token && cat /plugins/navidrome-music-room/room-data/secrets/plugin-pairing-token' 2>/dev/null || \
  awk -F= '/^MUSIC_ROOM_PLUGIN_PAIRING_TOKEN=/{print $2}' "$project_dir/.env"
echo "Configure and authorize users for navidrome-music-room.ndp in Navidrome, enable it, then open its Website link to manage rooms."
