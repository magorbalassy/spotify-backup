# Spotify Backup Web API

## Running the Web Server

To start the application in web server mode:

```bash
WEB_MODE=true PORT=8080 ./spotify-backup
```

Environment variables:
- `WEB_MODE`: Set to `true` or `1` to enable web server mode
- `PORT`: Server port (default: 8080)
- `SPOTIFY_REDIRECT_URI`: OAuth redirect URI (default: http://127.0.0.1:8888/callback)

## API Endpoints

### 1. Check Status

**GET** `/api/status`

Returns the current authentication status.

**Response:**
```json
{
  "hasToken": false,
  "hasClientId": false,
  "needsSetup": true,
  "message": "Please provide Spotify client ID and secret to begin"
}
```

**States:**
- `needsSetup: true, hasClientId: false` - Client credentials not configured yet
- `needsSetup: false, hasClientId: true, hasToken: false` - Ready for Spotify OAuth
- `hasToken: true` - Authenticated and ready to use

### 2. Setup Client Credentials

**POST** `/api/auth/setup`

Configure Spotify client ID and secret, and receive the authorization URL.

**Request:**
```json
{
  "clientId": "your-spotify-client-id",
  "clientSecret": "your-spotify-client-secret"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Client credentials saved. Please authorize the application",
  "authUrl": "https://accounts.spotify.com/authorize?client_id=..."
}
```

### 3. Start Authorization

**POST** `/api/auth/start`

Get the Spotify authorization URL (alternative to `/auth/setup` if credentials already configured).

**Response:**
```json
{
  "authUrl": "https://accounts.spotify.com/authorize?client_id=...",
  "message": "Please visit the auth URL to authorize the application"
}
```

### 4. OAuth Callback

**GET** `/api/auth/callback?code=...`

Receives the OAuth callback from Spotify. This endpoint is called by Spotify after user authorization.

**Response:**
Returns an HTML page indicating success or failure.

## Authentication Flow for Angular UI

### Scenario 1: No Token, No Client ID

1. UI calls `GET /api/status`
2. Response: `needsSetup: true`
3. UI displays form for Client ID and Secret
4. User submits credentials
5. UI calls `POST /api/auth/setup` with credentials
6. Backend returns `authUrl`
7. UI opens `authUrl` in new window/tab
8. User authorizes on Spotify
9. Spotify redirects to `/api/auth/callback`
10. Backend stores tokens and shows success page
11. User closes auth window
12. UI polls `GET /api/status` until `hasToken: true`

### Scenario 2: Has Client ID, No Token

1. UI calls `GET /api/status`
2. Response: `hasClientId: true, hasToken: false`
3. UI displays "Authenticate with Spotify" button
4. User clicks button
5. UI calls `POST /api/auth/start`
6. Backend returns `authUrl`
7. Continue from step 7 above

### Scenario 3: Has Token

1. UI calls `GET /api/status`
2. Response: `hasToken: true`
3. UI shows main application interface

## CORS Configuration

The API is configured to accept requests from:
- `http://localhost:4200` (Angular dev server)
- `http://localhost:3000` (alternative dev server)

Allowed methods: GET, POST, PUT, DELETE, OPTIONS

## Notes

- Refresh tokens are persisted to `.token` file
- The callback redirect URI must match what's configured in your Spotify app settings
- Access tokens are kept in memory only
- Client credentials are stored in memory and lost on restart
