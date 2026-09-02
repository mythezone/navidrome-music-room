VERSION ?= 1.1.0-dev
GO_IMAGE ?= golang:1.25-bookworm
GOPROXY ?= https://proxy.golang.org,direct

.PHONY: test frontend race vet plugin build validate-plugin validate-openapi validate-compose clean

test: frontend
	docker run --rm -e GOPROXY=$(GOPROXY) -v navidrome_music_room_go_cache:/go/pkg/mod -v $(CURDIR)/gateway:/src -w /src $(GO_IMAGE) sh -c 'go test ./...'
	docker run --rm -e GOPROXY=$(GOPROXY) -v navidrome_music_room_go_cache:/go/pkg/mod -v $(CURDIR)/plugin:/src -w /src $(GO_IMAGE) sh -c 'go test ./...'

frontend:
	npm --prefix admin-ui ci
	npm --prefix admin-ui audit --audit-level=moderate
	npm --prefix admin-ui test
	npm --prefix admin-ui run build
	npm --prefix room-ui ci
	npm --prefix room-ui audit --audit-level=high
	npm --prefix room-ui test
	npm --prefix room-ui run build

race:
	docker run --rm -e GOPROXY=$(GOPROXY) -v navidrome_music_room_go_cache:/go/pkg/mod -v $(CURDIR)/gateway:/src -w /src $(GO_IMAGE) sh -c 'go test -race ./...'

vet:
	docker run --rm -e GOPROXY=$(GOPROXY) -v navidrome_music_room_go_cache:/go/pkg/mod -v $(CURDIR)/gateway:/src -w /src $(GO_IMAGE) sh -c 'go vet ./...'

plugin:
	mkdir -p dist
	docker build --target artifacts --build-arg VERSION=$(VERSION) --output type=local,dest=dist .

build: plugin

validate-plugin: plugin
	docker run --rm -v $(CURDIR)/dist:/dist:ro deluan/navidrome:0.63.2 plugin validate /dist/navidrome-music-room.ndp

validate-openapi:
	ruby -e "require 'yaml'; YAML.load_file('contracts/openapi.yaml', aliases: true); puts 'openapi.yaml: YAML OK'"

validate-compose:
	NAVIDROME_PUBLIC_URL=https://music.example.test MUSIC_ROOM_PUBLIC_URL=https://rooms.example.test MUSIC_ROOM_PLUGIN_PAIRING_TOKEN=0123456789abcdef0123456789abcdef docker compose config --quiet

clean:
	rm -rf dist
