package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/arvind/whoop-go/whoop"
	"github.com/arvind/whoop-stats/internal/config"
	"github.com/arvind/whoop-stats/internal/crypto"
	"github.com/arvind/whoop-stats/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	cfg    *config.Config
	db     *db.Queries
	logger *slog.Logger

	clientCache   sync.Map
	clientMutexes sync.Map
}

func NewManager(cfg *config.Config, queries *db.Queries, logger *slog.Logger) *Manager {
	return &Manager{
		cfg:    cfg,
		db:     queries,
		logger: logger,
	}
}

type tokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// GetClient returns an authenticated whoop.Client for the given whoop_user_id.
// It handles token decryption, refreshing if expired (for now we assume if we are calling this, we just refresh to be safe or use a valid one),
// and updates the DB if refreshed. In a real app we'd check expiry, but WHOOP tokens expire in 1 hour, so we can just refresh it on startup/poll.
func (m *Manager) GetClient(ctx context.Context, whoopUserID string) (*whoop.Client, error) {
	// 1. Check cache first
	if cached, ok := m.clientCache.Load(whoopUserID); ok {
		return cached.(*whoop.Client), nil
	}

	// 2. Lock per user to prevent concurrent DB hits / refreshes
	mutexI, _ := m.clientMutexes.LoadOrStore(whoopUserID, &sync.Mutex{})
	mu := mutexI.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	// Double-check cache after locking
	if cached, ok := m.clientCache.Load(whoopUserID); ok {
		return cached.(*whoop.Client), nil
	}

	// Try to get user from DB
	user, err := m.db.GetUserByWhoopID(ctx, whoopUserID)
	if err != nil {
		m.logger.Warn("User not found in DB, attempting offline load", "whoop_user_id", whoopUserID)
		// If not found in DB, try to load from .whoop_token.json as a fallback (offline mode)
		user, err = m.loadFromOfflineJSON(ctx, whoopUserID)
		if err != nil {
			return nil, fmt.Errorf("user not found and offline json load failed: %w", err)
		}
		m.logger.Info("Successfully loaded user tokens from offline json")
	}

	// Decrypt refresh token
	encryptionKey := []byte(m.cfg.EncryptionKey)
	refreshToken, err := crypto.Decrypt(user.EncryptedRefreshToken, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt refresh token: %w", err)
	}

	// Always refresh token for polling session to ensure we have a fresh 1-hour access token
	newTok, err := m.refreshToken(string(refreshToken))
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	m.logger.Debug("Successfully refreshed token from WHOOP API", "whoop_user_id", whoopUserID)

	// Encrypt new tokens
	encAccess, err := crypto.Encrypt([]byte(newTok.AccessToken), encryptionKey)
	if err != nil {
		return nil, err
	}
	encRefresh, err := crypto.Encrypt([]byte(newTok.RefreshToken), encryptionKey)
	if err != nil {
		return nil, err
	}

	// Update DB
	_, err = m.db.UpsertUser(ctx, db.UpsertUserParams{
		WhoopUserID:           whoopUserID,
		EncryptedAccessToken:  encAccess,
		EncryptedRefreshToken: encRefresh,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update user tokens: %w", err)
	}

	// Return configured client
	client := whoop.NewClient(whoop.WithToken(newTok.AccessToken))

	m.clientCache.Store(whoopUserID, client)
	time.AfterFunc(55*time.Minute, func() {
		m.clientCache.Delete(whoopUserID)
	})

	return client, nil
}
func (m *Manager) GetInternalUserID(ctx context.Context, whoopUserID string) (pgtype.UUID, error) {
	user, err := m.db.GetUserByWhoopID(ctx, whoopUserID)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return user.ID, nil
}

func (m *Manager) loadFromOfflineJSON(ctx context.Context, whoopUserID string) (db.User, error) {
	// For offline single-tenant, we check if .whoop_token.json exists
	f, err := os.ReadFile(".whoop_token.json")
	if err != nil {
		return db.User{}, err
	}
	var tok tokenData
	if err := json.Unmarshal(f, &tok); err != nil {
		return db.User{}, err
	}

	if tok.RefreshToken == "" {
		return db.User{}, errors.New("no refresh token in offline json")
	}

	encryptionKey := []byte(m.cfg.EncryptionKey)
	encAccess, _ := crypto.Encrypt([]byte(tok.AccessToken), encryptionKey)
	encRefresh, _ := crypto.Encrypt([]byte(tok.RefreshToken), encryptionKey)

	// Upsert to DB
	return m.db.UpsertUser(ctx, db.UpsertUserParams{
		WhoopUserID:           whoopUserID, // we assume the JSON belongs to this whoop user
		EncryptedAccessToken:  encAccess,
		EncryptedRefreshToken: encRefresh,
	})
}

func (m *Manager) refreshToken(refreshTok string) (*tokenData, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshTok)
	data.Set("client_id", m.cfg.WhoopClientID)
	data.Set("client_secret", m.cfg.WhoopClientSecret)
	data.Set("scope", "offline")

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

	return &result, nil
}
