package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-key-32-bytes-long!!"

func makeToken(t *testing.T, claims jwt.MapClaims, method jwt.SigningMethod, key interface{}) string {
	t.Helper()
	token := jwt.NewWithClaims(method, claims)
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return signed
}

func TestAuth_ValidToken(t *testing.T) {
	secret := []byte(testSecret)
	tokenStr := makeToken(t, jwt.MapClaims{
		"whoop_user_id": "12345",
		"exp":           time.Now().Add(time.Hour).Unix(),
	}, jwt.SigningMethodHS256, secret)

	handler := Auth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := r.Context().Value(WhoopUserIDKey)
		if uid != "12345" {
			t.Errorf("expected whoop_user_id=12345, got %v", uid)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAuth_MissingHeader(t *testing.T) {
	handler := Auth([]byte(testSecret))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuth_InvalidFormat(t *testing.T) {
	handler := Auth([]byte(testSecret))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz") // Basic auth, not Bearer
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuth_ExpiredToken(t *testing.T) {
	secret := []byte(testSecret)
	tokenStr := makeToken(t, jwt.MapClaims{
		"whoop_user_id": "12345",
		"exp":           time.Now().Add(-time.Hour).Unix(), // Expired 1 hour ago
	}, jwt.SigningMethodHS256, secret)

	handler := Auth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for expired token")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuth_WrongSigningKey(t *testing.T) {
	// Sign with one key, verify with another
	tokenStr := makeToken(t, jwt.MapClaims{
		"whoop_user_id": "12345",
		"exp":           time.Now().Add(time.Hour).Unix(),
	}, jwt.SigningMethodHS256, []byte("wrong-secret-key-32-bytes-long!!"))

	handler := Auth([]byte(testSecret))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for wrong key")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuth_MissingWhoopUserIDClaim(t *testing.T) {
	secret := []byte(testSecret)
	tokenStr := makeToken(t, jwt.MapClaims{
		"sub": "someone",
		"exp": time.Now().Add(time.Hour).Unix(),
	}, jwt.SigningMethodHS256, secret)

	handler := Auth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called without whoop_user_id")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuth_NonHMACAlgorithm(t *testing.T) {
	// We can't easily sign with RS256 without an RSA key, but we can
	// verify the middleware rejects a token with an unexpected "alg" header.
	// Craft a token header that claims "none" algorithm.
	secret := []byte(testSecret)

	// Create a token manually with "none" algorithm attempt
	handler := Auth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for non-HMAC algorithm")
	}))

	// Use an unsigned token (alg: none)
	unsignedToken := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ3aG9vcF91c2VyX2lkIjoiMTIzNDUiLCJleHAiOjk5OTk5OTk5OTl9."

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+unsignedToken)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unsigned token, got %d", rr.Code)
	}
}
