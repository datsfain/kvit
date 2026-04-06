package drive

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	clientID     = "203669531432-3i00dasn3vondekdoiqgcoo4caki2lcl.apps.googleusercontent.com"
	clientSecret = "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" // Safe for desktop apps per Google OAuth docs
	folderName   = "kvit"
	authURL      = "https://accounts.google.com/o/oauth2/v2/auth"
	tokenURL     = "https://oauth2.googleapis.com/token"
)

var oauthConfig = &oauth2.Config{
	ClientID: clientID,
	Scopes:   []string{drive.DriveScope},
	Endpoint: oauth2.Endpoint{
		AuthURL:  authURL,
		TokenURL: tokenURL,
	},
}

// generateRandomString creates a cryptographically random base64url string
func generateRandomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// PKCE helpers
func generateCodeVerifier() (string, error) {
	return generateRandomString(32)
}

func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	dir := filepath.Join(home, ".config", "kvit")
	os.MkdirAll(dir, 0700)
	return dir
}

func tokenPath() string {
	return filepath.Join(configDir(), "token.json")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

type kvitConfig struct {
	FolderID string `json:"folder_id,omitempty"`
}

func loadConfig() kvitConfig {
	var c kvitConfig
	data, err := os.ReadFile(configPath())
	if err != nil {
		return c
	}
	json.Unmarshal(data, &c)
	return c
}

func saveConfig(c kvitConfig) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0600)
}

// LinkFolder saves a shared folder ID for syncing
func LinkFolder(folderURL string) error {
	// Extract folder ID from URL like https://drive.google.com/drive/folders/1aBcD...
	id := folderURL
	if strings.Contains(folderURL, "drive.google.com") {
		parts := strings.Split(strings.TrimRight(folderURL, "/"), "/")
		id = parts[len(parts)-1]
		// Remove query params
		if idx := strings.Index(id, "?"); idx != -1 {
			id = id[:idx]
		}
	}

	if id == "" {
		return fmt.Errorf("could not extract folder ID from URL")
	}

	// Verify access
	srv, err := getService()
	if err != nil {
		return err
	}
	folder, err := srv.Files.Get(id).Fields("id, name").Do()
	if err != nil {
		return fmt.Errorf("cannot access folder: %w\nMake sure the folder is shared with your Google account", err)
	}

	c := loadConfig()
	c.FolderID = id
	if err := saveConfig(c); err != nil {
		return err
	}

	fmt.Printf("✓ Linked to folder: %s\n", folder.Name)
	return nil
}

// GetLinkedFolder returns the configured folder ID, if any
func GetLinkedFolder() string {
	return loadConfig().FolderID
}

func saveToken(token *oauth2.Token) error {
	path := tokenPath()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}

func loadToken() (*oauth2.Token, error) {
	f, err := os.Open(tokenPath())
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var token oauth2.Token
	err = json.NewDecoder(f).Decode(&token)
	return &token, err
}

// IsAuthenticated checks if a valid token exists
func IsAuthenticated() bool {
	token, err := loadToken()
	if err != nil {
		return false
	}
	return token.Valid() || token.RefreshToken != ""
}

// Auth performs the OAuth2 PKCE flow: opens browser, receives callback, stores token
func Auth() error {
	// Generate PKCE verifier and challenge
	verifier, err := generateCodeVerifier()
	if err != nil {
		return fmt.Errorf("failed to generate PKCE verifier: %w", err)
	}
	challenge := generateCodeChallenge(verifier)

	// Generate random state for CSRF protection
	state, err := generateRandomString(16)
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	// Start local server to receive callback
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)
	oauthConfig.RedirectURL = redirectURL

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("invalid state parameter (possible CSRF)")
			fmt.Fprint(w, "<html><body><h2>Error: invalid state parameter.</h2></body></html>")
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			fmt.Fprint(w, "<html><body><h2>Error: no authorization code received.</h2></body></html>")
			return
		}
		codeCh <- code
		fmt.Fprint(w, "<html><body><h2>✓ Authorized! You can close this tab.</h2></body></html>")
	})

	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Close()

	// Build auth URL with PKCE parameters and state
	authRequestURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&access_type=offline&prompt=consent+select_account&code_challenge=%s&code_challenge_method=S256&state=%s",
		authURL,
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURL),
		url.QueryEscape(drive.DriveScope),
		url.QueryEscape(challenge),
		url.QueryEscape(state),
	)

	fmt.Println("Opening browser for Google sign-in...")
	fmt.Printf("\nIf the browser doesn't open, visit this URL:\n%s\n\n", authRequestURL)

	openBrowser(authRequestURL)

	fmt.Println("Waiting for authorization...")

	// Wait for callback
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	case <-time.After(2 * time.Minute):
		return fmt.Errorf("authorization timed out (2 minutes)")
	}

	// Exchange code for token with PKCE verifier (no client secret)
	token, err := exchangeCodePKCE(code, verifier, redirectURL)
	if err != nil {
		return fmt.Errorf("failed to exchange token: %w", err)
	}

	if err := saveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	return nil
}

// exchangeCodePKCE exchanges an authorization code for a token using PKCE (no client secret)
func exchangeCodePKCE(code, verifier, redirectURL string) (*oauth2.Token, error) {
	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"code_verifier": {verifier},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURL},
	}

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed (%d): %s", resp.StatusCode, string(body))
	}

	var token oauth2.Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

// Logout removes the stored token
func Logout() error {
	path := tokenPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// refreshToken refreshes an expired token using the refresh token (no client secret)
func refreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
	}

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken:  result.AccessToken,
		TokenType:    result.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(result.ExpiresIn) * time.Second),
	}, nil
}

// getService creates an authenticated Drive service
func getService() (*drive.Service, error) {
	token, err := loadToken()
	if err != nil {
		return nil, fmt.Errorf("not authenticated. Run: kvit auth")
	}

	// Refresh if expired
	if !token.Valid() {
		if token.RefreshToken == "" {
			return nil, fmt.Errorf("token expired. Run: kvit auth")
		}
		token, err = refreshToken(token)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w\nRun: kvit auth", err)
		}
		saveToken(token)
	}

	return drive.NewService(context.Background(), option.WithTokenSource(oauth2.StaticTokenSource(token)))
}

// escapeQuery escapes single quotes for Google Drive API queries
func escapeQuery(s string) string {
	return strings.ReplaceAll(s, "'", "\\'")
}

// getOrCreateFolder returns the linked folder ID, or finds/creates the "kvit" folder in Drive root
func getOrCreateFolder(srv *drive.Service) (string, error) {
	// Use linked folder if configured
	if linked := GetLinkedFolder(); linked != "" {
		return linked, nil
	}

	// Search for existing folder
	q := fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder' and trashed=false", escapeQuery(folderName))
	list, err := srv.Files.List().Q(q).Fields("files(id, name)").Do()
	if err != nil {
		return "", fmt.Errorf("failed to search for folder: %w", err)
	}
	if len(list.Files) > 0 {
		return list.Files[0].Id, nil
	}

	// Create folder
	folder := &drive.File{
		Name:     folderName,
		MimeType: "application/vnd.google-apps.folder",
	}
	created, err := srv.Files.Create(folder).Fields("id").Do()
	if err != nil {
		return "", fmt.Errorf("failed to create folder: %w", err)
	}
	return created.Id, nil
}

// OpenFolder opens the kvit Drive folder in the browser
func OpenFolder() error {
	srv, err := getService()
	if err != nil {
		return err
	}

	folderID, err := getOrCreateFolder(srv)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://drive.google.com/drive/folders/%s", folderID)
	fmt.Printf("Opening %s\n", url)
	openBrowser(url)
	return nil
}

// Push uploads local CSV files to Drive
func Push(files []string) error {
	srv, err := getService()
	if err != nil {
		return err
	}

	folderID, err := getOrCreateFolder(srv)
	if err != nil {
		return err
	}

	for _, filename := range files {
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			continue
		}

		f, err := os.Open(filename)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", filename, err)
		}

		// Check if file exists in Drive folder
		q := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false", escapeQuery(filename), escapeQuery(folderID))
		list, err := srv.Files.List().Q(q).Fields("files(id)").Do()
		if err != nil {
			f.Close()
			return fmt.Errorf("failed to search for %s: %w", filename, err)
		}

		if len(list.Files) > 0 {
			// Update existing
			_, err = srv.Files.Update(list.Files[0].Id, &drive.File{MimeType: "text/csv"}).Media(f).Do()
		} else {
			// Create new
			driveFile := &drive.File{
				Name:     filename,
				MimeType: "text/csv",
				Parents:  []string{folderID},
			}
			_, err = srv.Files.Create(driveFile).Media(f).Do()
		}
		f.Close()

		if err != nil {
			return fmt.Errorf("failed to upload %s: %w", filename, err)
		}
		fmt.Printf("  ✓ %s\n", filename)
	}
	return nil
}

// Pull downloads CSV files from Drive to local
func Pull(files []string) error {
	srv, err := getService()
	if err != nil {
		return err
	}

	folderID, err := getOrCreateFolder(srv)
	if err != nil {
		return err
	}

	for _, filename := range files {
		q := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false", escapeQuery(filename), escapeQuery(folderID))
		list, err := srv.Files.List().Q(q).Fields("files(id)").Do()
		if err != nil {
			return fmt.Errorf("failed to search for %s: %w", filename, err)
		}

		if len(list.Files) == 0 {
			fmt.Printf("  - %s (not on Drive, skipping)\n", filename)
			continue
		}

		resp, err := srv.Files.Get(list.Files[0].Id).Download()
		if err != nil {
			return fmt.Errorf("failed to download %s: %w", filename, err)
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", filename, err)
		}

		if err := os.WriteFile(filename, data, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
		fmt.Printf("  ✓ %s\n", filename)
	}
	return nil
}
