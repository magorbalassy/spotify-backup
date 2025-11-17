# Spotify backup

Containerized app to backup your Spotify playlists.

This program assumes you already have Spotify credentials or a refresh token. It does not implement the full interactive OAuth flow.  
Containerization: create a Dockerfile that sets those env vars or injects them at runtime.  
Extend: add retries/backoff, rate-limit handling (429), incremental backups (compare existing files), or export playlists as CSV/CSV+track uris.  

# Usage (example):

Build:  
`GOOS=linux GOARCH=amd64 go build -o spotify-backup ./spotify-backup.go`  

Build Docker image:
`docker build -t spotify-backup:latest .`

Run with direct token:  
`OUT_DIR=./backup SPOTIFY_ACCESS_TOKEN="ya29...." ./spotify-backup` 

Launch image locally:  
```bash
export SPOTIFY_CLIENT_ID=your_client_id
export SPOTIFY_CLIENT_SECRET=your_client_secret
docker run --rm -it -p 8888:8888 \
  -v "$PWD/.token:/data/.token" \
  -v "$PWD/backup:/data/out" \
  -e SPOTIFY_REDIRECT_URI=http://127.0.0.1:8888/callback \
  spotify-backup:latest
```

