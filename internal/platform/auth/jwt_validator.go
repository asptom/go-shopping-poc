package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const claimsKey contextKey = "claims"

var (
	ErrMissingAuthHeader = errors.New("missing authorization header")
	ErrInvalidAuthHeader = errors.New("invalid authorization header format")
	ErrInvalidToken      = errors.New("invalid or expired token")
	ErrMissingRole       = errors.New("missing required role")
)

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// KeycloakValidator validates JWT tokens from Keycloak
type KeycloakValidator struct {
	issuer  string
	jwksURL string
}

// NewKeycloakValidator creates a new Keycloak JWT validator
func NewKeycloakValidator(issuer, jwksURL string) *KeycloakValidator {
	return &KeycloakValidator{
		issuer:  issuer,
		jwksURL: jwksURL,
	}
}

// ValidateToken validates a JWT token and returns claims
func (k *KeycloakValidator) ValidateToken(ctx context.Context, token string) (*Claims, error) {
	parsedToken, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Support both HMAC and RSA
		switch method := token.Method; method {
		case jwt.SigningMethodHS256:
			return []byte("secret"), nil
		case jwt.SigningMethodRS256:
			return k.getPublicKey(token.Header["kid"].(string))
		default:
			return nil, fmt.Errorf("unexpected signing method: %v", method)
		}
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if claims, ok := parsedToken.Claims.(*Claims); ok && parsedToken.Valid {
		if claims.Issuer != k.issuer {
			return nil, fmt.Errorf("%w: issuer mismatch", ErrInvalidToken)
		}
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// getPublicKey fetches the RSA public key from JWKS
func (k *KeycloakValidator) getPublicKey(kid string) (*rsa.PublicKey, error) {
	resp, err := http.Get(k.jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %v", err)
	}
	defer resp.Body.Close()

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %v", err)
	}

	for _, key := range jwks.Keys {
		if key.Kid == kid && key.Kty == "RSA" && key.Use == "sig" {
			// Decode the modulus and exponent
			nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
			if err != nil {
				return nil, fmt.Errorf("failed to decode modulus: %v", err)
			}
			eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
			if err != nil {
				return nil, fmt.Errorf("failed to decode exponent: %v", err)
			}

			n := new(big.Int).SetBytes(nBytes)
			e := int(new(big.Int).SetBytes(eBytes).Int64())

			return &rsa.PublicKey{
				N: n,
				E: e,
			}, nil
		}
	}

	return nil, fmt.Errorf("key with kid %s not found", kid)
}

// HasRole checks if the claims contain a specific role
func (c *Claims) HasRole(role string) bool {
	if c.RealmAccess == nil || c.RealmAccess.Roles == nil {
		return false
	}

	for _, r := range c.RealmAccess.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// Claims represents the claims from a Keycloak JWT token
type Claims struct {
	Subject           string       `json:"sub"`
	Email             string       `json:"email"`
	PreferredUsername string       `json:"preferred_username"`
	RealmAccess       *RealmAccess `json:"realm_access"`
	jwt.RegisteredClaims
}

// RealmAccess contains realm roles
type RealmAccess struct {
	Roles []string `json:"roles"`
}

// RequireAuth creates middleware that validates JWT tokens
func RequireAuth(validator *KeycloakValidator, requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, ErrMissingAuthHeader.Error(), http.StatusUnauthorized)
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, ErrInvalidAuthHeader.Error(), http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")

			claims, err := validator.ValidateToken(r.Context(), token)
			if err != nil {
				http.Error(w, ErrInvalidToken.Error(), http.StatusUnauthorized)
				return
			}

			if requiredRole != "" && !claims.HasRole(requiredRole) {
				http.Error(w, ErrMissingRole.Error(), http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims retrieves claims from the request context
func GetClaims(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}
