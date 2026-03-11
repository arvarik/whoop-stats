// Package auth manages WHOOP OAuth2 token lifecycle including caching, refresh,
// encryption at rest, and offline fallback for homelab deployments.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/arvarik/whoop-go/whoop"
	"github.com/arvind/whoop-stats/internal/config"
	"github.com/arvind/whoop-stats/internal/crypto"
	"github.com/arvind/whoop-stats/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// whoopTokenEndpoint is the WHOOP OAuth2 token exchange endpoint.
const whoopTokenEndpoint = "https://api.prod.whoop.com/oauth/oauth2/token"

// defaultRedirectURI must match the redirect URI used during initial OAuth authorization.
// WHOOP requires it on refresh_token requests as well.
const defaultRedirectURI = "http://localhost:8081/callback"

// Manager handles WHOOP API authentication, including OAuth2 token refresh,
// AES-256-GCM encrypted token storage, and per-user client caching with TTL.
type Manager struct {
	cfg    *config.Config
	db     *db.Queries
	logger *slog.Logger

	// httpClient is used for all WHOOP API token requests with a configured timeout.
	httpClient *http.Client

	// clientCache stores authenticated whoop.Client instances keyed by whoop_user_id.
	// Entries are evicted 55 minutes after creation (WHOOP tokens expire in 1 hour).
	clientCache   sync.Map
	clientMutexes sync.Map
}

// NewManager creates a new auth Manager with the given configuration.
func NewManager(cfg *config.Config, queries *db.Queries, logger *slog.Logger) *Manager {
	return &Manager{
		cfg:    cfg,
		db:     queries,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// tokenData represents an OAuth2 token response from the WHOOP API.
type tokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// GetClient returns an authenticated whoop.Client for the given whoop_user_id.
// It uses a cached client if available, otherwise loads tokens from the database
// (or offline JSON fallback), refreshes them via the WHOOP API, encrypts and
// stores the new tokens, and caches the client for subsequent calls.
func (m *Manager) GetClient(ctx context.Context, whoopUserID string) (*whoop.Client, error) {
	// Check cache first (fast path, no lock)
	if cached, ok := m.clientCache.Load(whoopUserID); ok {
		return cached.(*whoop.Client), nil
	}

	// Per-user lock to prevent concurrent DB hits and token refreshes
	mutexI, _ := m.clientMutexes.LoadOrStore(whoopUserID, &sync.Mutex{})
	mu := mutexI.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	// Double-check cache after acquiring lock
	if cached, ok := m.clientCache.Load(whoopUserID); ok {
		return cached.(*whoop.Client), nil
	}

	// Load user from DB, falling back to offline JSON for first-run scenarios
	user, err := m.db.GetUserByWhoopID(ctx, whoopUserID)
	if err != nil {
		m.logger.Warn("User not found in DB, attempting offline JSON fallback", "whoop_user_id", whoopUserID)
		user, err = m.loadFromOfflineJSON(ctx, whoopUserID)
		if err != nil {
			return nil, fmt.Errorf("user not found and offline JSON load failed: %w", err)
		}
		m.logger.Info("Loaded user tokens from offline JSON")
	}

	// Decrypt refresh token
	encryptionKey := []byte(m.cfg.EncryptionKey)
	refreshToken, err := crypto.Decrypt(user.EncryptedRefreshToken, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypting refresh token: %w", err)
	}

	// Always refresh to get a fresh 1-hour access token
	newTok, err := m.refreshToken(ctx, string(refreshToken))
	if err != nil {
		return nil, fmt.Errorf("refreshing token: %w", err)
	}

	m.logger.Debug("Refreshed WHOOP API token", "whoop_user_id", whoopUserID)

	// Encrypt new tokens for storage
	encAccess, err := crypto.Encrypt([]byte(newTok.AccessToken), encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("encrypting access token: %w", err)
	}
	encRefresh, err := crypto.Encrypt([]byte(newTok.RefreshToken), encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("encrypting refresh token: %w", err)
	}

	// Persist encrypted tokens
	if _, err = m.db.UpsertUser(ctx, db.UpsertUserParams{
		WhoopUserID:           whoopUserID,
		EncryptedAccessToken:  encAccess,
		EncryptedRefreshToken: encRefresh,
	}); err != nil {
		return nil, fmt.Errorf("persisting refreshed tokens: %w", err)
	}

	// Build and cache the client with a 55-minute TTL (tokens expire in 1 hour)
	client := whoop.NewClient(whoop.WithToken(newTok.AccessToken))
	m.clientCache.Store(whoopUserID, client)
	time.AfterFunc(55*time.Minute, func() {
		m.clientCache.Delete(whoopUserID)
	})

	return client, nil
}

// GetInternalUserID maps a WHOOP user ID to the internal database UUID.
func (m *Manager) GetInternalUserID(ctx context.Context, whoopUserID string) (pgtype.UUID, error) {
	user, err := m.db.GetUserByWhoopID(ctx, whoopUserID)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("looking up user %q: %w", whoopUserID, err)
	}
	return user.ID, nil
}

// loadFromOfflineJSON reads tokens from .whoop_token.json for first-run scenarios
// where the user has run cmd/auth but hasn't yet been inserted into the DB.
func (m *Manager) loadFromOfflineJSON(ctx context.Context, whoopUserID string) (db.User, error) {
	f, err := os.ReadFile(".whoop_token.json")
	if err != nil {
		return db.User{}, fmt.Errorf("reading .whoop_token.json: %w", err)
	}

	var tok tokenData
	if err := json.Unmarshal(f, &tok); err != nil {
		return db.User{}, fmt.Errorf("parsing .whoop_token.json: %w", err)
	}

	if tok.RefreshToken == "" {
		return db.User{}, errors.New("no refresh_token in .whoop_token.json")
	}

	encryptionKey := []byte(m.cfg.EncryptionKey)
	encAccess, err := crypto.Encrypt([]byte(tok.AccessToken), encryptionKey)
	if err != nil {
		return db.User{}, fmt.Errorf("encrypting offline access token: %w", err)
	}
	encRefresh, err := crypto.Encrypt([]byte(tok.RefreshToken), encryptionKey)
	if err != nil {
		return db.User{}, fmt.Errorf("encrypting offline refresh token: %w", err)
	}

	return m.db.UpsertUser(ctx, db.UpsertUserParams{
		WhoopUserID:           whoopUserID,
		EncryptedAccessToken:  encAccess,
		EncryptedRefreshToken: encRefresh,
	})
}

// refreshToken exchanges a refresh token for new access and refresh tokens.
func (m *Manager) refreshToken(ctx context.Context, refreshTok string) (*tokenData, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshTok)
	data.Set("client_id", m.cfg.WhoopClientID)
	data.Set("client_secret", m.cfg.WhoopClientSecret)
	data.Set("redirect_uri", defaultRedirectURI)
	data.Set("scope", "offline")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, whoopTokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading token response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WHOOP token endpoint returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result tokenData
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}

	if result.AccessToken == "" {
		return nil, fmt.Errorf("empty access_token in WHOOP response (HTTP %d)", resp.StatusCode)
	}

	return &result, nil
}
