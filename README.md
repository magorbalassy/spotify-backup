# Spotify backup

Containerized app to backup your Spotify playlists.

This program assumes you already have Spotify credentials or a refresh token. It does not implement the full interactive OAuth flow.
Containerization: create a Dockerfile that sets those env vars or injects them at runtime.
Extend: add retries/backoff, rate-limit handling (429), incremental backups (compare existing files), or export playlists as CSV/CSV+track uris.

# Usage (example):

Build: GOOS=linux GOARCH=amd64 go build -o spotify-backup ./spotify-backup.go
Run with direct token:
OUT_DIR=./backup SPOTIFY_ACCESS_TOKEN="ya29...." ./spotify-backup
Or run with refresh workflow (recommended for containers):
OUT_DIR=./backup SPOTIFY_CLIENT_ID=... SPOTIFY_CLIENT_SECRET=... SPOTIFY_REFRESH_TOKEN=... ./spotify-backup
Notes:

