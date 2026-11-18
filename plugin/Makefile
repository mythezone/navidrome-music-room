VERSION ?= 1.2.1
DIST ?= ../dist

.PHONY: build package clean

build:
	mkdir -p $(DIST)/plugin
	@if command -v go >/dev/null 2>&1; then \
		go_root=$$(go env GOROOT) || exit $$?; \
		export GOROOT="$$go_root" PATH="$$go_root/bin:$$PATH"; \
	fi; \
	GOFLAGS="$(GOFLAGS) -buildvcs=false" tinygo build -target wasip1 -buildmode=c-shared -ldflags="-X main.version=$(VERSION)" -o $(DIST)/plugin/plugin.wasm .
	@LC_ALL=C grep -aFq -- "$(VERSION)" $(DIST)/plugin/plugin.wasm || (echo "TinyGo did not inject plugin version $(VERSION)" >&2; exit 1)

package: build
	sed -E 's/"version": "[^"]+"/"version": "$(VERSION)"/' manifest.json > $(DIST)/plugin/manifest.json
	cd $(DIST)/plugin && zip -q -FS ../navidrome-music-room.ndp manifest.json plugin.wasm

clean:
	rm -rf $(DIST)/plugin $(DIST)/navidrome-music-room.ndp
