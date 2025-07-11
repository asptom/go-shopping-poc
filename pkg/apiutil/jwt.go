package apiutil

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	ContextKeyUser contextKey = "user"
)

// JWTMiddleware validates JWT tokens in the Authorization header.
func JWTMiddleware(secretOrPublicKey []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "missing or invalid Authorization header", http.StatusUnauthorized)
				return
			}
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// For KeyCloak, you may need to use RS256 and provide the public key
				return secretOrPublicKey, nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// Optionally, pass claims/user info in context
			ctx := context.WithValue(r.Context(), ContextKeyUser, token.Claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserClaims extracts JWT claims from the request context.
func GetUserClaims(r *http.Request) (jwt.Claims, error) {
	claims, ok := r.Context().Value(ContextKeyUser).(jwt.Claims)
	if !ok {
		return nil, errors.New("no user claims in context")
	}
	return claims, nil
}
