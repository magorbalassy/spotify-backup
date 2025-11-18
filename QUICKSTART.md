# Quick Start Guide

## Start the Backend Server

```bash
# Build the application
go build -o spotify-backup

# Start in web server mode
WEB_MODE=true PORT=8080 ./spotify-backup
```

The server will start on `http://localhost:8080`

## Test with cURL

### 1. Check initial status
```bash
curl http://localhost:8080/api/status
```

Expected response:
```json
{
  "hasToken": false,
  "hasClientId": false,
  "needsSetup": true,
  "message": "Please provide Spotify client ID and secret to begin"
}
```

### 2. Setup credentials and get auth URL
```bash
curl -X POST http://localhost:8080/api/auth/setup \
  -H "Content-Type: application/json" \
  -d '{
    "clientId": "YOUR_SPOTIFY_CLIENT_ID",
    "clientSecret": "YOUR_SPOTIFY_CLIENT_SECRET"
  }'
```

Expected response:
```json
{
  "success": true,
  "message": "Client credentials saved. Please authorize the application",
  "authUrl": "https://accounts.spotify.com/authorize?client_id=..."
}
```

### 3. Open the auth URL in browser
Copy the `authUrl` from step 2 and open it in your browser. After authorizing, you'll be redirected to the callback URL.

### 4. Check status again
```bash
curl http://localhost:8080/api/status
```

Expected response after successful auth:
```json
{
  "hasToken": true,
  "hasClientId": true,
  "needsSetup": false,
  "message": "Authentication complete. Ready to backup playlists"
}
```

## Getting Spotify Credentials

1. Go to [Spotify Developer Dashboard](https://developer.spotify.com/dashboard)
2. Log in with your Spotify account
3. Click "Create App"
4. Fill in app name and description
5. Add redirect URI: `http://127.0.0.1:8888/callback`
6. Copy your Client ID and Client Secret

## Environment Variables

- `WEB_MODE`: Set to `true` to run as web server (default: CLI mode)
- `PORT`: Server port (default: 8080)
- `SPOTIFY_REDIRECT_URI`: OAuth callback URL (default: http://127.0.0.1:8888/callback)
- `SPOTIFY_CLIENT_ID`: Pre-configure client ID (optional)
- `SPOTIFY_CLIENT_SECRET`: Pre-configure client secret (optional)

## Running in Docker

If you have Docker:

```bash
docker build -t spotify-backup .
docker run -p 8080:8080 -e WEB_MODE=true spotify-backup
```

## CLI Mode (Original Functionality)

To run in CLI mode without the web server:

```bash
# Set credentials via environment variables
export SPOTIFY_CLIENT_ID="your-client-id"
export SPOTIFY_CLIENT_SECRET="your-client-secret"

# Run the backup
./spotify-backup
```

This will prompt for interactive OAuth in the terminal and then backup all playlists.
