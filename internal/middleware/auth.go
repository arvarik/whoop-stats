// Package middleware provides HTTP middleware for the whoop-stats API server.
// It includes authentication (JWT), request logging, and IP-based rate limiting.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

// UserIDKey is the context key for the internal user UUID.
const UserIDKey contextKey = "user_id"

// WhoopUserIDKey is the context key for the WHOOP-specific user identifier.
const WhoopUserIDKey contextKey = "whoop_user_id"

// Auth returns middleware that validates JWT Bearer tokens from the Authorization header.
// It enforces HS256 signing to prevent algorithm confusion attacks and extracts
// the whoop_user_id claim into the request context for downstream handlers.
func Auth(secret []byte) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenStr := parts[1]

			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				// Enforce HMAC signing method to prevent algorithm confusion attacks.
				// Without this check, an attacker could use "alg: none" or RSA-based
				// algorithms to forge valid tokens.
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return secret, nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "invalid token claims", http.StatusUnauthorized)
				return
			}

			whoopUserID, ok := claims["whoop_user_id"].(string)
			if !ok {
				http.Error(w, "missing whoop_user_id in token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), WhoopUserIDKey, whoopUserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
