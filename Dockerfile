# syntax=docker/dockerfile:1.12
ARG GO_IMAGE=golang:1.25-bookworm
ARG COSIGN_IMAGE=ghcr.io/sigstore/cosign/cosign@sha256:d91bc4e7e95e8d2f549c747a72dc174f90579e410a1695f57f686674f84ce849

FROM ${GO_IMAGE} AS gateway-build
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG GOPROXY=https://proxy.golang.org,direct
WORKDIR /src/gateway
COPY gateway/go.mod gateway/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod GOPROXY=${GOPROXY} go mod download
COPY gateway/ ./
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -buildvcs=false -trimpath -ldflags='-s -w' -o /out/music-room-gateway ./cmd/music-room-gateway && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -buildvcs=false -trimpath -ldflags='-s -w' -o /out/music-room-launcher ./cmd/music-room-launcher

FROM --platform=$BUILDPLATFORM ${GO_IMAGE} AS plugin-build
ARG VERSION=0.1.0-dev
ARG BUILDARCH
ARG TINYGO_VERSION=0.41.1
ARG GOPROXY=https://proxy.golang.org,direct
RUN case "${BUILDARCH}" in \
      amd64) tinygo_sha=e156d1d93a376eef639a4143d13be07e8c463fb6cf2d7d447698ed4474d23e91 ;; \
      arm64) tinygo_sha=789733bc3b5bace0bd1835a267b3ea267804a7ef1cfe69bc522c295f5226d624 ;; \
      *) echo "Unsupported TinyGo build architecture: ${BUILDARCH}" >&2; exit 1 ;; \
    esac && \
    curl --http1.1 -fsSL --retry 5 --retry-all-errors -o /tmp/tinygo.tar.gz \
      "https://github.com/tinygo-org/tinygo/releases/download/v${TINYGO_VERSION}/tinygo${TINYGO_VERSION}.linux-${BUILDARCH}.tar.gz" && \
    echo "${tinygo_sha}  /tmp/tinygo.tar.gz" | sha256sum -c - && \
    mkdir -p /opt/tinygo && tar -xzf /tmp/tinygo.tar.gz -C /opt/tinygo --strip-components=1 && \
    rm /tmp/tinygo.tar.gz
ENV TINYGOROOT=/opt/tinygo
ENV PATH=/opt/tinygo/bin:$PATH
WORKDIR /src/plugin
COPY plugin/go.mod plugin/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod GOPROXY=${GOPROXY} go mod download
COPY plugin/ ./
RUN --mount=type=cache,target=/go/pkg/mod \
    GOFLAGS=-buildvcs=false tinygo build -target wasip1 -buildmode=c-shared -ldflags="-X main.version=${VERSION}" -o /out/plugin.wasm .

FROM debian:bookworm-slim AS ndp-build
ARG VERSION=0.1.0-dev
RUN apt-get update && apt-get install -y --no-install-recommends zip ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /out/plugin
COPY plugin/manifest.json ./manifest.json
COPY release.json /out/release.json
COPY --from=plugin-build /out/plugin.wasm ./plugin.wasm
RUN sed -i "s/\"version\": \"0.1.0-dev\"/\"version\": \"${VERSION}\"/" manifest.json && \
    sed -i "s/0.1.0-dev/${VERSION}/" /out/release.json && \
    zip -q -9 /out/navidrome-music-room.ndp manifest.json plugin.wasm

FROM ${COSIGN_IMAGE} AS cosign

FROM scratch AS artifacts
COPY --from=gateway-build /out/music-room-gateway /music-room-gateway
COPY --from=gateway-build /out/music-room-launcher /music-room-launcher
COPY --from=cosign /ko-app/cosign /cosign
COPY deploy/sigstore/trusted_root.json /sigstore-trusted-root.json
COPY --from=ndp-build /out/navidrome-music-room.ndp /navidrome-music-room.ndp
COPY --from=ndp-build /out/release.json /release.json

FROM debian:bookworm-slim
ARG VERSION=0.1.0-dev
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata && rm -rf /var/lib/apt/lists/*
COPY --from=gateway-build /out/music-room-launcher /usr/local/bin/music-room-launcher
COPY --from=gateway-build /out/music-room-gateway /opt/music-room/release/music-room-gateway
COPY --from=cosign /ko-app/cosign /opt/music-room/release/cosign
COPY deploy/sigstore/trusted_root.json /opt/music-room/release/sigstore-trusted-root.json
COPY --from=ndp-build /out/navidrome-music-room.ndp /opt/music-room/release/navidrome-music-room.ndp
COPY --from=ndp-build /out/release.json /opt/music-room/release/release.json
RUN chmod 0755 /usr/local/bin/music-room-launcher /opt/music-room/release/music-room-gateway /opt/music-room/release/cosign && \
    chmod 0644 /opt/music-room/release/sigstore-trusted-root.json /opt/music-room/release/navidrome-music-room.ndp /opt/music-room/release/release.json
ENV MUSIC_ROOM_COSIGN_BINARY=/opt/music-room/release/cosign
ENV MUSIC_ROOM_SIGSTORE_TRUSTED_ROOT=/opt/music-room/release/sigstore-trusted-root.json
USER 65532:65532
EXPOSE 4534
HEALTHCHECK --interval=15s --timeout=3s --start-period=15s --retries=4 CMD ["/opt/music-room/release/music-room-gateway", "--healthcheck"]
ENTRYPOINT ["/usr/local/bin/music-room-launcher"]
