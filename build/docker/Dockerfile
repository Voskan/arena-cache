# syntax=docker/dockerfile:1

###############################################################################
# arena-cache‑inspect – minimal static container (multi‑arch)                 #
###############################################################################
# The image is built by GitHub Actions (release.yml) for linux/amd64 & arm64.  #
# It uses a two‑stage build:                                                   #
#   1. Builder   – Go 1.24 on Alpine, CGO disabled, static musl binary.        #
#   2. Scratch   – copy binary + CA certs → ~9 MB image.                       #
###############################################################################

ARG GO_VERSION=1.24.0
ARG ALPINE_VERSION=3.19

############################## 1. Builder stage ################################
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

# Install build tools (musl-dev for static linking, ca‑certs for HTTPS at test)
RUN apk add --no-cache git build-base ca-certificates

WORKDIR /src

# Copy full repo (context = project root)
COPY . .

# Build static binary
ARG VERSION=dev
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS=linux \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
        -o /out/arena-cache-inspect ./cmd/arena-cache-inspect

############################## 2. Runtime stage ################################
FROM scratch AS final

# Copy binary and TLS root certificates (for HTTPS targets)
COPY --from=builder /out/arena-cache-inspect /usr/local/bin/arena-cache-inspect
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

LABEL org.opencontainers.image.title="arena-cache-inspect" \
      org.opencontainers.image.description="CLI for inspecting arena-cache instances" \
      org.opencontainers.image.url="https://github.com/${GITHUB_REPOSITORY}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.licenses="MIT"

ENTRYPOINT ["/usr/local/bin/arena-cache-inspect"]
CMD ["-h"]

# Image size ≈ 9 MB (static binary + certs)
