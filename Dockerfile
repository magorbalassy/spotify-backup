# Multi-stage build for spotify-backup

# --- Build stage ---
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY spotify-backup.go ./
# Build without BuildKit cache mounts for compatibility with legacy builder
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -ldflags="-s -w" -o /out/spotify-backup ./spotify-backup.go

# --- Runtime stage ---
FROM alpine:3.20
# CA certs for HTTPS requests
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 app
USER app
WORKDIR /data

# Defaults: write outputs under /data/out
ENV OUT_DIR=/data/out
# Document the typical OAuth callback port; can be changed via SPOTIFY_REDIRECT_URI
EXPOSE 8888

COPY --from=build /out/spotify-backup /usr/local/bin/spotify-backup

# .token will be read/written relative to /data (bind-mount this from host)
ENTRYPOINT ["/usr/local/bin/spotify-backup"]
