package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var (
	envAccessToken     = "SPOTIFY_ACCESS_TOKEN"  // optional: direct token
	envRefreshToken    = "SPOTIFY_REFRESH_TOKEN" // optional: use with client id/secret
	envClientID        = "SPOTIFY_CLIENT_ID"
	envClientSecret    = "SPOTIFY_CLIENT_SECRET"
	envRedirectURI     = "SPOTIFY_REDIRECT_URI"
	envOutDir          = "OUT_DIR"
	defaultOutDir      = "./backup"
	defaultRedirectURI = "http://127.0.0.1:8888/callback"
	tokenFile          = ".token"
	userAgent          = "spotify-backup/1.0"
	sanitizePattern    = regexp.MustCompile(`[^\w\-. ]+`)
	httpClient         = &http.Client{Timeout: 30 * time.Second}
)

// Global state for web server mode
var (
	appState = &AppState{
		clientID:     os.Getenv(envClientID),
		clientSecret: os.Getenv(envClientSecret),
	}
)

type AppState struct {
	accessToken  string
	refreshToken string
	clientID     string
	clientSecret string
	redirectURI  string
}

type playlistPage struct {
	Items []playlistItem `json:"items"`
	Next  string         `json:"next"`
}

type playlistItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Owner       struct {
		DisplayName string `json:"display_name"`
		ID          string `json:"id"`
	} `json:"owner"`
	Tracks struct {
		Total int `json:"total"`
	} `json:"tracks"`
	Images []struct {
		URL string `json:"url"`
	} `json:"images"`
}

type tracksPage struct {
	Items []trackItem `json:"items"`
	Next  string      `json:"next"`
}

type trackItem struct {
	AddedAt string `json:"added_at"`
	Track   track  `json:"track"`
}

type track struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	DurationMs   int               `json:"duration_ms"`
	Popularity   int               `json:"popularity"`
	Explicit     bool              `json:"explicit"`
	PreviewURL   string            `json:"preview_url"`
	ExternalURLs map[string]string `json:"external_urls"`
	Artists      []struct {
		Name string `json:"name"`
	} `json:"artists"`
	Album struct {
		Name string `json:"name"`
	} `json:"album"`
}

type savedPlaylist struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Owner       string      `json:"owner"`
	Image       string      `json:"image,omitempty"`
	TracksTotal int         `json:"tracks_total"`
	Tracks      []trackItem `json:"tracks"`
	SourceURL   string      `json:"source_url"`
}

func main() {
	// Check if web server mode is enabled
	webMode := os.Getenv("WEB_MODE")
	if webMode == "true" || webMode == "1" {
		startWebServer()
		return
	}

	// Original CLI mode
	runCLIMode()
}

func runCLIMode() {
	outDir := os.Getenv(envOutDir)
	if outDir == "" {
		outDir = defaultOutDir
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fail("create outdir:", err)
	}

	accessToken := os.Getenv(envAccessToken)
	refreshToken := os.Getenv(envRefreshToken)
	clientID := os.Getenv(envClientID)
	clientSecret := os.Getenv(envClientSecret)
	redirectURI := os.Getenv(envRedirectURI)
	if redirectURI == "" {
		redirectURI = defaultRedirectURI
	}

	// Try to load refresh token from file if not in env
	if refreshToken == "" {
		if saved, err := loadRefreshToken(); err == nil && saved != "" {
			refreshToken = saved
			fmt.Println("Loaded refresh token from", tokenFile)
		}
	}

	// If no tokens but have client credentials, do interactive auth
	if accessToken == "" && refreshToken == "" && clientID != "" && clientSecret != "" {
		fmt.Println("No tokens found. Starting interactive OAuth flow...")
		tok, refTok, err := doInteractiveAuth(clientID, clientSecret, redirectURI)
		if err != nil {
			fail("interactive auth failed:", err)
		}
		accessToken = tok
		refreshToken = refTok

		// Save refresh token to file
		if err := saveRefreshToken(refreshToken); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to save refresh token: %v\n", err)
		} else {
			fmt.Println("✓ Refresh token saved to", tokenFile)
		}
	} else if accessToken == "" && refreshToken != "" && clientID != "" && clientSecret != "" {
		tok, err := refreshAccessToken(clientID, clientSecret, refreshToken)
		if err != nil {
			fail("refresh token:", err)
		}
		accessToken = tok
		fmt.Println("Got access token from refresh token")
	}

	if accessToken == "" {
		fail("no SPOTIFY_ACCESS_TOKEN and no refresh token+client credentials provided")
	}

	playlists, err := fetchAllPlaylists(accessToken)
	if err != nil {
		fail("fetch playlists:", err)
	}
	fmt.Printf("Found %d playlists\n", len(playlists))

	imagesDir := filepath.Join(outDir, "images")
	_ = os.MkdirAll(imagesDir, 0o755)
	plistDir := filepath.Join(outDir, "playlists")
	_ = os.MkdirAll(plistDir, 0o755)

	index := make([]map[string]string, 0, len(playlists))

	for i, p := range playlists {
		fmt.Printf("[%d/%d] downloading playlist %q (%s)\n", i+1, len(playlists), p.Name, p.ID)
		tracks, err := fetchAllPlaylistTracks(accessToken, p.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to fetch tracks for %s: %v\n", p.ID, err)
			continue
		}
		sp := savedPlaylist{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Owner:       p.Owner.DisplayName,
			TracksTotal: p.Tracks.Total,
			Tracks:      tracks,
			SourceURL:   fmt.Sprintf("https://open.spotify.com/playlist/%s", p.ID),
		}
		if len(p.Images) > 0 && p.Images[0].URL != "" {
			sp.Image = p.Images[0].URL
		}

		fileName := safeFilename(fmt.Sprintf("%s-%s.json", p.Name, p.ID))
		outPath := filepath.Join(plistDir, fileName)
		if err := writeJSONFile(outPath, sp); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to write playlist file %s: %v\n", outPath, err)
			continue
		}

		// optional: download playlist image
		if sp.Image != "" {
			imgExt := ".jpg"
			u, _ := url.Parse(sp.Image)
			if u != nil {
				if ext := filepath.Ext(u.Path); ext != "" && len(ext) <= 5 {
					imgExt = ext
				}
			}
			imgName := safeFilename(fmt.Sprintf("playlist-%s%s", p.ID, imgExt))
			imgPath := filepath.Join(imagesDir, imgName)
			if err := downloadFile(sp.Image, imgPath); err == nil {
				index = append(index, map[string]string{
					"id":        p.ID,
					"name":      p.Name,
					"file":      filepath.Join("playlists", fileName),
					"imageFile": filepath.Join("images", imgName),
				})
			} else {
				index = append(index, map[string]string{
					"id":   p.ID,
					"name": p.Name,
					"file": filepath.Join("playlists", fileName),
				})
			}
		} else {
			index = append(index, map[string]string{
				"id":   p.ID,
				"name": p.Name,
				"file": filepath.Join("playlists", fileName),
			})
		}
	}

	// write top-level index
	indexPath := filepath.Join(outDir, "playlists-index.json")
	if err := writeJSONFile(indexPath, index); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write index: %v\n", err)
	}
	fmt.Println("Backup completed. Output dir:", outDir)
}

// refreshAccessToken exchanges a refresh token for a new access token.
func refreshAccessToken(clientID, clientSecret, refreshToken string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)

	req, _ := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token refresh failed: %s - %s", resp.Status, string(b))
	}
	var out struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.AccessToken == "" {
		return "", errors.New("no access_token received")
	}
	return out.AccessToken, nil
}

func fetchAllPlaylists(accessToken string) ([]playlistItem, error) {
	var all []playlistItem
	url := "https://api.spotify.com/v1/me/playlists?limit=50"
	for url != "" {
		var page playlistPage
		if err := apiGetJSON(accessToken, url, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Items...)
		url = page.Next
	}
	return all, nil
}

func fetchAllPlaylistTracks(accessToken, playlistID string) ([]trackItem, error) {
	var all []trackItem
	url := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks?limit=100", playlistID)
	for url != "" {
		var page tracksPage
		if err := apiGetJSON(accessToken, url, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Items...)
		url = page.Next
	}
	return all, nil
}

func apiGetJSON(accessToken, urlStr string, out interface{}) error {
	req, _ := http.NewRequest("GET", urlStr, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("unauthorized - access token expired or invalid")
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("spotify api error %s: %s", resp.Status, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func downloadFile(urlStr, dest string) error {
	req, _ := http.NewRequest("GET", urlStr, nil)
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed download %s: %s", urlStr, resp.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func writeJSONFile(path string, v interface{}) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		f.Close()
		return err
	}
	f.Close()
	return os.Rename(tmp, path)
}

func safeFilename(name string) string {
	name = strings.TrimSpace(name)
	name = sanitizePattern.ReplaceAllString(name, "_")
	if len(name) > 200 {
		name = name[:200]
	}
	return name
}

func fail(v ...interface{}) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(1)
}

// loadRefreshToken reads the refresh token from the local token file if present.
func loadRefreshToken() (string, error) {
	b, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// saveRefreshToken persists the refresh token to the local token file.
func saveRefreshToken(tok string) error {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return errors.New("empty refresh token")
	}
	return os.WriteFile(tokenFile, []byte(tok+"\n"), 0o600)
}

// doInteractiveAuth implements the authorization code flow with local server
func doInteractiveAuth(clientID, clientSecret, redirectURI string) (accessToken, refreshToken string, err error) {
	// Parse port from redirect URI
	u, _ := url.Parse(redirectURI)
	port := u.Port()
	if port == "" {
		port = "8888"
	}

	scopes := "playlist-read-private playlist-read-collaborative user-library-read"
	authURL := fmt.Sprintf(
		"https://accounts.spotify.com/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=%s",
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(scopes),
	)

	// Channel to receive the authorization code
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Start local HTTP server
	srv := &http.Server{Addr: ":" + port}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- errors.New("no code in callback")
			fmt.Fprintf(w, "<html><body><h1>Error: No authorization code received</h1></body></html>")
			return
		}
		codeChan <- code
		fmt.Fprintf(w, "<html><body><h1>✓ Authorization successful!</h1><p>You can close this window and return to the terminal.</p></body></html>")
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	fmt.Println("Starting Spotify authorization...")
	fmt.Println("Opening browser for authentication...")

	// Open browser automatically
	if err := openBrowser(authURL); err != nil {
		fmt.Println("\nCouldn't open browser automatically. Please open this URL manually:")
		fmt.Printf("\n   %s\n\n", authURL)
	}

	fmt.Println("Waiting for authorization...")

	var code string
	select {
	case code = <-codeChan:
		fmt.Println("✓ Authorization successful")
	case err := <-errChan:
		return "", "", err
	case <-time.After(5 * time.Minute):
		return "", "", errors.New("timeout waiting for authorization")
	}

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)

	// Exchange code for tokens
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)

	req, _ := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("token exchange failed: %s - %s", resp.Status, string(b))
	}

	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", err
	}

	if out.AccessToken == "" || out.RefreshToken == "" {
		return "", "", errors.New("no tokens received")
	}

	return out.AccessToken, out.RefreshToken, nil
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

// Web server functionality

func startWebServer() {
	mime.AddExtensionType(".js", "text/javascript")
	mime.AddExtensionType(".mjs", "text/javascript")
	mime.AddExtensionType(".css", "text/css")
	mime.AddExtensionType(".wasm", "application/wasm")

	// Load any existing tokens
	if rt, err := loadRefreshToken(); err == nil && rt != "" {
		appState.refreshToken = rt
		fmt.Println("Loaded refresh token from", tokenFile)
	}

	redirectURI := os.Getenv(envRedirectURI)
	if redirectURI == "" {
		redirectURI = defaultRedirectURI
	}
	appState.redirectURI = redirectURI

	gin.SetMode(gin.ReleaseMode)
	r := setupWebServer()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting web server on port %s...\n", port)
	if err := r.Run(":" + port); err != nil {
		fail("failed to start web server:", err)
	}
}

func setupWebServer() *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4200", "http://localhost:8080"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	api := r.Group("/api")
	{
		api.GET("/status", handleStatus)
		api.POST("/auth/setup", handleAuthSetup)
		api.GET("/auth/callback", handleAuthCallback)
		api.POST("/auth/start", handleAuthStart)
	}

	// Serve favicon (if present)
	r.StaticFile("/favicon.ico", "./public/favicon.ico")

	// Explicit root route to serve the SPA entry
	r.GET("/", func(c *gin.Context) {
		c.File("./public/index.html")
	})

	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		clean := path.Clean(p)
		if clean == "/" {
			c.File("./public/index.html")
			return
		}

		local := filepath.Join("public", strings.TrimPrefix(clean, "/"))
		if info, err := os.Stat(local); err == nil && !info.IsDir() {
			c.File(local)
			return
		}

		c.File("./public/index.html")
	})

	return r
}

// API Response types
type StatusResponse struct {
	HasToken    bool   `json:"hasToken"`
	HasClientID bool   `json:"hasClientId"`
	NeedsSetup  bool   `json:"needsSetup"`
	Message     string `json:"message"`
}

type AuthSetupRequest struct {
	ClientID     string `json:"clientId" binding:"required"`
	ClientSecret string `json:"clientSecret" binding:"required"`
}

type AuthSetupResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	AuthURL string `json:"authUrl,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// handleStatus checks if there's a valid token or if setup is needed
func handleStatus(c *gin.Context) {
	hasToken := appState.accessToken != "" || appState.refreshToken != ""
	hasClientID := appState.clientID != ""

	resp := StatusResponse{
		HasToken:    hasToken,
		HasClientID: hasClientID,
		NeedsSetup:  !hasClientID,
	}

	if !hasClientID {
		resp.Message = "Please provide Spotify client ID and secret to begin"
	} else if !hasToken {
		resp.Message = "Client credentials configured. Ready to authenticate with Spotify"
	} else {
		resp.Message = "Authentication complete. Ready to backup playlists"
	}

	c.JSON(http.StatusOK, resp)
}

// handleAuthSetup receives client ID and secret from UI and initiates auth flow
func handleAuthSetup(c *gin.Context) {
	var req AuthSetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request: " + err.Error()})
		return
	}

	// Store credentials
	appState.clientID = req.ClientID
	appState.clientSecret = req.ClientSecret

	// Generate auth URL
	scopes := "playlist-read-private playlist-read-collaborative user-library-read"
	authURL := fmt.Sprintf(
		"https://accounts.spotify.com/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=%s",
		url.QueryEscape(req.ClientID),
		url.QueryEscape(appState.redirectURI),
		url.QueryEscape(scopes),
	)

	c.JSON(http.StatusOK, AuthSetupResponse{
		Success: true,
		Message: "Client credentials saved. Please authorize the application",
		AuthURL: authURL,
	})
}

// handleAuthStart initiates the OAuth flow
func handleAuthStart(c *gin.Context) {
	if appState.clientID == "" || appState.clientSecret == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Client credentials not configured"})
		return
	}

	scopes := "playlist-read-private playlist-read-collaborative user-library-read"
	authURL := fmt.Sprintf(
		"https://accounts.spotify.com/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=%s",
		url.QueryEscape(appState.clientID),
		url.QueryEscape(appState.redirectURI),
		url.QueryEscape(scopes),
	)

	c.JSON(http.StatusOK, gin.H{
		"authUrl": authURL,
		"message": "Please visit the auth URL to authorize the application",
	})
}

// handleAuthCallback receives the OAuth callback with authorization code
func handleAuthCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.HTML(http.StatusBadRequest, "", gin.H{})
		c.Writer.WriteString("<html><body><h1>Error: No authorization code received</h1></body></html>")
		return
	}

	// Exchange code for tokens
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", appState.redirectURI)

	req, _ := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(appState.clientID, appState.clientSecret)
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		c.Writer.WriteString(fmt.Sprintf("<html><body><h1>Error</h1><p>Failed to exchange authorization code: %s</p></body></html>", err.Error()))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		c.Writer.WriteString(fmt.Sprintf("<html><body><h1>Error</h1><p>Token exchange failed: %s - %s</p></body></html>", resp.Status, string(b)))
		return
	}

	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		c.Writer.WriteString(fmt.Sprintf("<html><body><h1>Error</h1><p>Failed to decode token response: %s</p></body></html>", err.Error()))
		return
	}

	if out.AccessToken == "" || out.RefreshToken == "" {
		c.Writer.WriteString("<html><body><h1>Error</h1><p>No tokens received from Spotify</p></body></html>")
		return
	}

	// Store tokens
	appState.accessToken = out.AccessToken
	appState.refreshToken = out.RefreshToken

	// Save refresh token to file
	if err := saveRefreshToken(out.RefreshToken); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to save refresh token: %v\n", err)
	} else {
		fmt.Println("✓ Refresh token saved to", tokenFile)
	}

	// Return success page
	c.Header("Content-Type", "text/html")
	c.Writer.WriteString("<html><body><h1>✓ Authorization successful!</h1><p>You can close this window and return to the application.</p></body></html>")
}
