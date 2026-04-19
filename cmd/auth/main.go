package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const tokenFile = ".whoop_token.json"

// tokenData represents the stored token session.
type tokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	clientID := os.Getenv("WHOOP_CLIENT_ID")
	clientSecret := os.Getenv("WHOOP_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		slog.Error("WHOOP_CLIENT_ID and WHOOP_CLIENT_SECRET environment variables are required")
		slog.Info("Usage:\n  export WHOOP_CLIENT_ID=your_id\n  export WHOOP_CLIENT_SECRET=your_secret\n  go run cmd/auth/main.go")
		os.Exit(1)
	}

	// Try to load and refresh an existing token first.
	if tok, err := loadToken(); err == nil && tok.RefreshToken != "" {
		slog.Info("Found existing token session. Attempting refresh...")
		newTok, err := refreshToken(clientID, clientSecret, tok.RefreshToken)
		if err == nil {
			saveToken(newTok)
			printToken(newTok)
			return
		}
		slog.Warn("Refresh failed, starting new authorization flow", "error", err)
	}

	// No valid session — run the full OAuth authorization code flow.
	runAuthFlow(clientID, clientSecret)
}

func runAuthFlow(clientID, clientSecret string) {
	redirectURI := os.Getenv("WHOOP_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "http://localhost:8081/callback"
	}

	u, err := url.Parse(redirectURI)
	if err != nil {
		slog.Error("Error parsing WHOOP_REDIRECT_URI", "error", err)
		os.Exit(1)
	}

	port := u.Port()
	if port == "" {
		port = "80"
		if u.Scheme == "https" {
			port = "443"
		}
	}

	scopes := []string{
		"offline",
		"read:recovery",
		"read:cycles",
		"read:workout",
		"read:sleep",
		"read:profile",
		"read:body_measurement",
	}

	authURL := fmt.Sprintf("https://api.prod.whoop.com/oauth/oauth2/auth?client_id=%s&response_type=code&redirect_uri=%s&scope=%s&state=whoop-stats-state",
		clientID,
		url.QueryEscape(redirectURI),
		url.QueryEscape(strings.Join(scopes, " ")),
	)

	slog.Info("=== WHOOP OAuth 2.0 Token Generator ===")
	slog.Info("IMPORTANT: Ensure you have added the following Redirect URI to your WHOOP App settings in the Developer Dashboard", "redirect_uri", redirectURI)
	slog.Info("Open this URL in your browser to authorize", "auth_url", authURL)
	slog.Info("Waiting for authorization callback", "port", port)

	server := &http.Server{Addr: ":" + port}

	http.HandleFunc(u.Path, func(w http.ResponseWriter, r *http.Request) {
		// Check for OAuth error response from WHOOP.
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			desc := r.URL.Query().Get("error_description")
			hint := r.URL.Query().Get("error_hint")
			slog.Error("OAuth error", "error", errParam, "description", desc, "hint", hint)
			msg := fmt.Sprintf("OAuth error: %s\nDescription: %s\nHint: %s", errParam, desc, hint)
			http.Error(w, msg, http.StatusBadRequest)
			go func() {
				time.Sleep(1 * time.Second)
				if err := server.Shutdown(context.Background()); err != nil {
					slog.Error("Server shutdown error", "error", err)
				}
			}()
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Failed to get auth code from request", http.StatusBadRequest)
			return
		}

		slog.Info("Received auth code! Exchanging for access token...")

		data := url.Values{}
		data.Set("grant_type", "authorization_code")
		data.Set("code", code)
		data.Set("client_id", clientID)
		data.Set("client_secret", clientSecret)
		data.Set("redirect_uri", redirectURI)

		tok, err := exchangeToken(data)
		if err != nil {
			http.Error(w, fmt.Sprintf("Token exchange error: %v", err), http.StatusInternalServerError)
			return
		}

		saveToken(tok)
		printToken(tok)

		_, _ = fmt.Fprintf(w, "Success! You can close this window and check your terminal.")

		go func() {
			time.Sleep(1 * time.Second)
			if err := server.Shutdown(context.Background()); err != nil {
				slog.Error("Server shutdown error", "error", err)
			}
		}()
	})

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

func exchangeToken(data url.Values) (*tokenData, error) {
	req, err := http.NewRequest(http.MethodPost, "https://api.prod.whoop.com/oauth/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result tokenData
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if result.AccessToken == "" {
		return nil, fmt.Errorf("no access token in response (HTTP %d)", resp.StatusCode)
	}

	result.ExpiresAt = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return &result, nil
}

func refreshToken(clientID, clientSecret, refreshTok string) (*tokenData, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshTok)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", "http://localhost:8081/callback")
	data.Set("scope", "offline")

	return exchangeToken(data)
}

func loadToken() (*tokenData, error) {
	f, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}
	var tok tokenData
	if err := json.Unmarshal(f, &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

func saveToken(tok *tokenData) {
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		slog.Warn("Could not save token", "error", err)
		return
	}
	if err := os.WriteFile(tokenFile, data, 0600); err != nil {
		slog.Warn("Could not write token file", "file", tokenFile, "error", err)
	}
}

func printToken(tok *tokenData) {
	slog.Info("=== SUCCESS ===")
	slog.Info("Token generated successfully")
	if tok.RefreshToken != "" {
		slog.Info("Initial tokens saved", "file", tokenFile)
		slog.Info("Upload this file to your NAS / Production server root")
	}
}
