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
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	clientID   = "203669531432-3i00dasn3vondekdoiqgcoo4caki2lcl.apps.googleusercontent.com"
	folderName = "kvit"
	authURL    = "https://accounts.google.com/o/oauth2/v2/auth"
	tokenURL   = "https://oauth2.googleapis.com/token"
)

var oauthConfig = &oauth2.Config{
	ClientID: clientID,
	Scopes:   []string{drive.DriveFileScope},
	Endpoint: oauth2.Endpoint{
		AuthURL:  authURL,
		TokenURL: tokenURL,
	},
}

// PKCE helpers
func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func tokenPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "kvit-token.json"
	}
	dir := filepath.Join(home, ".config", "kvit")
	os.MkdirAll(dir, 0700)
	return filepath.Join(dir, "token.json")
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

	// Build auth URL with PKCE parameters
	authRequestURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&access_type=offline&prompt=consent&code_challenge=%s&code_challenge_method=S256",
		authURL,
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURL),
		url.QueryEscape(drive.DriveFileScope),
		url.QueryEscape(challenge),
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

// getOrCreateFolder finds or creates the "kvit" folder in Drive root
func getOrCreateFolder(srv *drive.Service) (string, error) {
	// Search for existing folder
	q := fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder' and trashed=false", folderName)
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
		q := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false", filename, folderID)
		list, err := srv.Files.List().Q(q).Fields("files(id)").Do()
		if err != nil {
			f.Close()
			return fmt.Errorf("failed to search for %s: %w", filename, err)
		}

		if len(list.Files) > 0 {
			// Update existing
			_, err = srv.Files.Update(list.Files[0].Id, &drive.File{}).Media(f).Do()
		} else {
			// Create new
			driveFile := &drive.File{
				Name:    filename,
				Parents: []string{folderID},
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
		q := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false", filename, folderID)
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
