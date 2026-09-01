#!/bin/sh
set -eu

navidrome_tag=${1:-0.63.2}
root_dir=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
gateway_image=${MUSIC_ROOM_E2E_IMAGE:-}
gateway_binary=${MUSIC_ROOM_E2E_GATEWAY_BINARY:-$root_dir/dist/music-room-gateway-e2e}
pairing_token=0123456789abcdef0123456789abcdef
username=nmr-admin
password=nmr-e2e-password
salt=0123456789abcdef

for command_name in docker curl jq md5sum; do
  command -v "$command_name" >/dev/null 2>&1 || {
    echo "$command_name is required" >&2
    exit 1
  }
done
if [ -z "$gateway_image" ] && [ ! -x "$gateway_binary" ]; then
  echo "build $gateway_binary or set MUSIC_ROOM_E2E_IMAGE" >&2
  exit 1
fi
if [ "${MUSIC_ROOM_E2E_PULL_NAVIDROME:-true}" = true ]; then
  docker pull "deluan/navidrome:$navidrome_tag" >/dev/null
fi

case_id=$(printf '%s-%s' "$navidrome_tag" "$$" | tr -c 'A-Za-z0-9_.-' '-')
network_name=nmr-e2e-network-$case_id
navidrome_name=nmr-e2e-navidrome-$case_id
gateway_name=nmr-e2e-gateway-$case_id
work_dir=$(mktemp -d)

cleanup() {
  docker rm -f "$gateway_name" "$navidrome_name" >/dev/null 2>&1 || true
  docker network rm "$network_name" >/dev/null 2>&1 || true
  rm -rf "$work_dir"
}
trap cleanup EXIT INT TERM

mkdir -p "$work_dir/music"
cp "$root_dir/script/testdata/nmr-e2e-tone.mp3" "$work_dir/music/nmr-e2e-tone.mp3"
cp "$root_dir/script/testdata/nmr-e2e-tone.lrc" "$work_dir/music/nmr-e2e-tone.lrc"
chmod -R a+rX "$work_dir/music"

docker network create "$network_name" >/dev/null
docker run --detach --name "$navidrome_name" \
  --network "$network_name" --network-alias navidrome \
  --publish 127.0.0.1::4533 \
  --tmpfs /data:rw,mode=0700 \
  --volume "$work_dir/music:/music:ro" \
  --env ND_LOGLEVEL=warn \
  --env ND_SCANSCHEDULE='@every 30s' \
  "deluan/navidrome:$navidrome_tag" >/dev/null

if [ -n "$gateway_image" ]; then
  docker run --detach --name "$gateway_name" \
    --network "$network_name" \
    --publish 127.0.0.1::4534 \
    --tmpfs /data:rw,exec,uid=65532,gid=65532,mode=0700 \
    --tmpfs /plugins:rw,uid=65532,gid=65532,mode=0700 \
    --env MUSIC_ROOM_DATA_DIR=/data \
    --env MUSIC_ROOM_PLUGIN_INSTALL_DIR=/plugins \
    --env MUSIC_ROOM_NAVIDROME_INTERNAL_URL=http://navidrome:4533 \
    --env MUSIC_ROOM_NAVIDROME_PUBLIC_URL=https://music.e2e.invalid \
    --env MUSIC_ROOM_PUBLIC_URL=https://rooms.e2e.invalid \
    --env MUSIC_ROOM_PLUGIN_PAIRING_TOKEN="$pairing_token" \
    "$gateway_image" >/dev/null
else
  docker run --detach --name "$gateway_name" \
    --network "$network_name" \
    --user 65532:65532 \
    --publish 127.0.0.1::4534 \
    --tmpfs /data:rw,uid=65532,gid=65532,mode=0700 \
    --volume "$gateway_binary:/music-room-gateway:ro" \
    --env MUSIC_ROOM_DATA_DIR=/data \
    --env MUSIC_ROOM_NAVIDROME_INTERNAL_URL=http://navidrome:4533 \
    --env MUSIC_ROOM_NAVIDROME_PUBLIC_URL=https://music.e2e.invalid \
    --env MUSIC_ROOM_PUBLIC_URL=https://rooms.e2e.invalid \
    --env MUSIC_ROOM_PLUGIN_PAIRING_TOKEN="$pairing_token" \
    --entrypoint /music-room-gateway \
    debian:bookworm-slim >/dev/null
fi

navidrome_address=$(docker port "$navidrome_name" 4533/tcp | head -n 1)
gateway_address=$(docker port "$gateway_name" 4534/tcp | head -n 1)
navidrome_base=http://$navidrome_address
gateway_base=http://$gateway_address

wait_for_url() {
  endpoint=$1
  service_name=$2
  attempt=0
  while [ "$attempt" -lt 120 ]; do
    if curl -fsS --max-time 2 "$endpoint" >/dev/null 2>&1; then
      return 0
    fi
    attempt=$((attempt + 1))
    sleep 1
  done
  echo "$service_name did not become ready" >&2
  docker logs "$navidrome_name" >&2 || true
  docker logs "$gateway_name" >&2 || true
  return 1
}

wait_for_url "$navidrome_base/ping" Navidrome
wait_for_url "$gateway_base/healthz" 'Music Room Gateway'

curl -fsS --max-time 10 \
  --header 'Content-Type: application/json' \
  --data "{\"username\":\"$username\",\"password\":\"$password\"}" \
  "$navidrome_base/auth/createAdmin" >/dev/null

token=$(printf '%s%s' "$password" "$salt" | md5sum | awk '{print $1}')
subsonic() {
  method=$1
  shift
  curl -fsS --max-time 30 --get \
    --data-urlencode "u=$username" \
    --data-urlencode "t=$token" \
    --data-urlencode "s=$salt" \
    --data-urlencode 'v=1.16.1' \
    --data-urlencode 'c=MusicMate-E2E' \
    --data-urlencode 'f=json' \
    "$@" "$navidrome_base/rest/$method.view"
}

folders_json=$(subsonic getMusicFolders)
test "$(printf '%s' "$folders_json" | jq -r '."subsonic-response".status')" = ok
folder_id=$(printf '%s' "$folders_json" | jq -r '."subsonic-response".musicFolders.musicFolder[0].id // empty')
test -n "$folder_id"

song_id=
search_json=
attempt=0
while [ "$attempt" -lt 120 ]; do
  search_json=$(subsonic search3 \
    --data-urlencode 'query=NMR E2E Tone' \
    --data-urlencode 'artistCount=0' \
    --data-urlencode 'albumCount=0' \
    --data-urlencode 'songCount=10' \
    --data-urlencode "musicFolderId=$folder_id")
  song_id=$(printf '%s' "$search_json" | jq -r '."subsonic-response".searchResult3.song[0].id // empty')
  [ -n "$song_id" ] && break
  attempt=$((attempt + 1))
  sleep 1
done
test -n "$song_id"

song_json=$(subsonic getSong --data-urlencode "id=$song_id")
album_id=$(printf '%s' "$song_json" | jq -r '."subsonic-response".song.albumId // empty')
cover_id=$(printf '%s' "$song_json" | jq -r '."subsonic-response".song.coverArt // empty')
test -n "$album_id"
test -n "$cover_id"
album_json=$(subsonic getAlbum --data-urlencode "id=$album_id")
test "$(printf '%s' "$album_json" | jq -r '."subsonic-response".album.id // empty')" = "$album_id"

cover_file=$work_dir/cover.bin
subsonic getCoverArt --data-urlencode "id=$cover_id" >"$cover_file"
test -s "$cover_file"

lyrics_json=$(subsonic getLyricsBySongId --data-urlencode "id=$song_id")
test "$(printf '%s' "$lyrics_json" | jq -r '[(."subsonic-response".lyricsList.structuredLyrics // [])[] | .line[]] | length')" -ge 1

range_file=$work_dir/range.bin
range_code=$(curl -sS --max-time 30 --output "$range_file" --write-out '%{http_code}' \
  --header 'Range: bytes=0-255' --get \
  --data-urlencode "u=$username" --data-urlencode "t=$token" --data-urlencode "s=$salt" \
  --data-urlencode 'v=1.16.1' --data-urlencode 'c=MusicMate-E2E' --data-urlencode "id=$song_id" \
  "$navidrome_base/rest/stream.view")
test "$range_code" = 206
test -s "$range_file"

transcoded_file=$work_dir/transcoded.mp3
transcode_code=$(curl -sS --max-time 60 --output "$transcoded_file" --write-out '%{http_code}' --get \
  --data-urlencode "u=$username" --data-urlencode "t=$token" --data-urlencode "s=$salt" \
  --data-urlencode 'v=1.16.1' --data-urlencode 'c=MusicMate-E2E' --data-urlencode "id=$song_id" \
  --data-urlencode 'format=mp3' --data-urlencode 'maxBitRate=64' \
  "$navidrome_base/rest/stream.view")
test "$transcode_code" = 200
test -s "$transcoded_file"

sent_at=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
sync_payload=$(jq -cn \
  --arg sentAt "$sent_at" --arg username "$username" \
  '{pluginVersion:"e2e",generation:1,navidromeInternalURL:"http://navidrome:4533",navidromePublicURL:"https://music.e2e.invalid",gatewayPublicURL:"https://rooms.e2e.invalid",sentAt:$sentAt,users:[{username:$username,displayName:"E2E Admin",admin:true}]}')
curl -fsS --max-time 10 \
  --header "Authorization: Bearer $pairing_token" \
  --header 'Content-Type: application/json' \
  --data "$sync_payload" \
  "$gateway_base/internal/v1/plugin-sync" >/dev/null

exchange_payload=$(jq -cn --arg username "$username" --arg salt "$salt" --arg token "$token" '{username:$username,salt:$salt,token:$token}')
exchange_json=$(curl -fsS --max-time 10 \
  --header 'Content-Type: application/json' \
  --data "$exchange_payload" \
  "$gateway_base/api/v1/auth/exchange")
session_token=$(printf '%s' "$exchange_json" | jq -r '.sessionToken // empty')
test -n "$session_token"
test "$(printf '%s' "$exchange_json" | jq -r '.user.adminRole')" = true
test "$(printf '%s' "$exchange_json" | jq -r --argjson folder "$folder_id" '.user.musicFolderIDs | index($folder) != null')" = true

room_file=$work_dir/room.json
room_code=$(curl -sS --max-time 10 --output "$room_file" --write-out '%{http_code}' \
  --header "Authorization: Bearer $session_token" \
  --header 'Content-Type: application/json' \
  --data "{\"name\":\"Navidrome $navidrome_tag E2E\",\"musicFolderIDs\":[$folder_id]}" \
  "$gateway_base/api/v1/rooms")
test "$room_code" = 201
room_id=$(jq -r '.roomID // empty' "$room_file")
test -n "$room_id"

locked_code=$(curl -sS --max-time 10 --output /dev/null --write-out '%{http_code}' \
  --header "Authorization: Bearer $session_token" \
  "$gateway_base/api/v1/rooms/$room_id/chat")
test "$locked_code" = 402

diagnostics_json=$(curl -fsS --max-time 10 \
  --header "Authorization: Bearer $session_token" \
  "$gateway_base/api/v1/admin/diagnostics")
test "$(printf '%s' "$diagnostics_json" | jq -r '.redacted')" = true
if printf '%s' "$diagnostics_json" | grep -Eq 'music\.e2e\.invalid|rooms\.e2e\.invalid|nmr-admin'; then
  echo 'diagnostic export leaked a URL or username' >&2
  exit 1
fi

gateway_media_code=$(curl -sS --max-time 10 --output /dev/null --write-out '%{http_code}' "$gateway_base/api/v1/stream")
test "$gateway_media_code" = 404

navidrome_version=$(printf '%s' "$search_json" | jq -r '."subsonic-response".serverVersion // ."subsonic-response".version // "unknown"')
printf 'Navidrome %s (%s): auth, folders, search, album, cover, lyrics, Range, transcode, room ACL, lock, and redacted diagnostics passed\n' \
  "$navidrome_tag" "$navidrome_version"
