# Multi-stage build for spotify-backup

# --- Angular build ---
FROM node:20-alpine AS ui
WORKDIR /ui
COPY spotify-backup-ui/package*.json ./
RUN npm ci
COPY spotify-backup-ui/ ./
RUN npm run build

# --- Go build ---
FROM golang:1.25-alpine AS go-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY spotify-backup.go ./
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -ldflags="-s -w" -o /out/spotify-backup ./spotify-backup.go

# --- Runtime ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app
USER app
WORKDIR /app
ENV WEB_MODE=1 PORT=8080 OUT_DIR=/data/out
EXPOSE 8080 8888
COPY --from=go-build /out/spotify-backup /usr/local/bin/spotify-backup
# Angular dist output path may differ; adjust if needed:
COPY --from=ui /ui/dist/spotify-backup-ui/browser ./public
ENTRYPOINT ["/usr/local/bin/spotify-backup"]
