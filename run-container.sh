#!/usr/bin/env bash
set -euo pipefail

# Simple launcher for the spotify-backup container
# Usage:
#   ./run-container.sh [PORT]
# Defaults:
#   PORT=8888 (must match the port in SPOTIFY_REDIRECT_URI)
# Required for first run (interactive OAuth):
#   export SPOTIFY_CLIENT_ID=... SPOTIFY_CLIENT_SECRET=...
#   (Optionally override callback)
#   export SPOTIFY_REDIRECT_URI=http://127.0.0.1:8888/callback
# After first run a refresh token will be saved to ./.token and
# you can omit client credentials for subsequent runs.

PORT=${1:-8888}
IMAGE=${IMAGE:-spotify-backup:latest}
NAME=${NAME:-spotify-backup}

# Build the image if not present
if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
  echo "[i] Building image $IMAGE ..."
  docker build -t "$IMAGE" .
fi

# Prepare volumes
mkdir -p ./backup
[ -f ./.token ] || touch ./.token >/dev/null 2>&1 || true

# Assemble docker run args
RUN_ARGS=(
  --rm -it
  --name "$NAME"
  -p "$PORT:$PORT"
  -v "$PWD/.token:/data/.token"
  -v "$PWD/backup:/data/out"
)

# Pass env vars only if set in host environment
if [[ -n "${SPOTIFY_CLIENT_ID:-}" ]]; then RUN_ARGS+=( -e "SPOTIFY_CLIENT_ID=$SPOTIFY_CLIENT_ID" ); fi
if [[ -n "${SPOTIFY_CLIENT_SECRET:-}" ]]; then RUN_ARGS+=( -e "SPOTIFY_CLIENT_SECRET=$SPOTIFY_CLIENT_SECRET" ); fi
if [[ -n "${SPOTIFY_ACCESS_TOKEN:-}" ]]; then RUN_ARGS+=( -e "SPOTIFY_ACCESS_TOKEN=$SPOTIFY_ACCESS_TOKEN" ); fi
if [[ -n "${SPOTIFY_REFRESH_TOKEN:-}" ]]; then RUN_ARGS+=( -e "SPOTIFY_REFRESH_TOKEN=$SPOTIFY_REFRESH_TOKEN" ); fi

# If user did not provide a redirect URI, set one that matches the mapped port
if [[ -z "${SPOTIFY_REDIRECT_URI:-}" ]]; then
  export SPOTIFY_REDIRECT_URI="http://127.0.0.1:${PORT}/callback"
  RUN_ARGS+=( -e "SPOTIFY_REDIRECT_URI=$SPOTIFY_REDIRECT_URI" )
else
  RUN_ARGS+=( -e "SPOTIFY_REDIRECT_URI=$SPOTIFY_REDIRECT_URI" )
fi

# Ensure OUT_DIR points to the mounted output dir inside the container
RUN_ARGS+=( -e "OUT_DIR=/data/out" )

# Info message
cat <<EOF
[i] Running $IMAGE as $NAME
    OAuth callback: $SPOTIFY_REDIRECT_URI
    Mapped port:    $PORT:$PORT
    Token file:     $PWD/.token
    Output dir:     $PWD/backup
EOF

exec docker run "${RUN_ARGS[@]}" "$IMAGE"
