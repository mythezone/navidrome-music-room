VERSION ?= 0.1.0-dev
DIST ?= ../dist

.PHONY: build package clean

build:
	mkdir -p $(DIST)/plugin
	GOFLAGS="$(GOFLAGS) -buildvcs=false" tinygo build -target wasip1 -buildmode=c-shared -ldflags="-X main.version=$(VERSION)" -o $(DIST)/plugin/plugin.wasm .

package: build
	cp manifest.json $(DIST)/plugin/manifest.json
	cd $(DIST)/plugin && zip -q -FS ../navidrome-music-room.ndp manifest.json plugin.wasm

clean:
	rm -rf $(DIST)/plugin $(DIST)/navidrome-music-room.ndp
