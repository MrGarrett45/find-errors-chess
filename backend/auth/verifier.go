// Package auth verifies Auth0 JWTs via JWKS and validates issuer/audience.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultLeeway = 30 * time.Second
)

// Verifier validates Auth0 JWT access tokens against a JWKS endpoint.
type Verifier struct {
	issuer   string
	audience string
	keyfunc  keyfunc.Keyfunc
	parser   *jwt.Parser
}

// NewVerifierFromEnv initializes a verifier from AUTH0_ISSUER and AUTH0_AUDIENCE.
func NewVerifierFromEnv() (*Verifier, error) {
	issuer := strings.TrimSpace(os.Getenv("AUTH0_ISSUER"))
	audience := strings.TrimSpace(os.Getenv("AUTH0_AUDIENCE"))
	if issuer == "" || audience == "" {
		return nil, errors.New("AUTH0_ISSUER and AUTH0_AUDIENCE must be set")
	}
	return NewVerifier(issuer, audience, "")
}

// NewVerifier builds a verifier with an optional JWKS URL override.
func NewVerifier(issuer, audience, jwksURL string) (*Verifier, error) {
	normalizedIssuer := normalizeIssuer(issuer)
	if normalizedIssuer == "" {
		return nil, errors.New("issuer must be set")
	}
	if audience == "" {
		return nil, errors.New("audience must be set")
	}
	if jwksURL == "" {
		jwksURL = normalizedIssuer + ".well-known/jwks.json"
	}

	keyProvider, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("failed to init JWKS keyfunc: %w", err)
	}

	parser := jwt.NewParser(
		jwt.WithIssuer(normalizedIssuer),
		jwt.WithAudience(audience),
		jwt.WithLeeway(defaultLeeway),
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name, jwt.SigningMethodRS512.Name, jwt.SigningMethodRS384.Name}),
	)

	return &Verifier{
		issuer:   normalizedIssuer,
		audience: audience,
		keyfunc:  keyProvider,
		parser:   parser,
	}, nil
}

// Verify parses and validates a JWT, returning extracted claims.
func (v *Verifier) Verify(tokenString string) (*Claims, error) {
	token, err := v.parser.Parse(tokenString, v.keyfunc.Keyfunc)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	claims := &Claims{
		Subject:   readString(mapClaims, "sub"),
		Issuer:    readString(mapClaims, "iss"),
		Audience:  readAudience(mapClaims["aud"]),
		ExpiresAt: readExpiry(mapClaims["exp"]),
		Scope:     readString(mapClaims, "scope"),
		Raw:       mapClaims,
	}
	if claims.Subject == "" {
		return nil, errors.New("token missing sub")
	}
	return claims, nil
}

func normalizeIssuer(issuer string) string {
	issuer = strings.TrimSpace(issuer)
	if issuer == "" {
		return ""
	}
	if !strings.HasSuffix(issuer, "/") {
		issuer += "/"
	}
	return issuer
}

func readString(claims jwt.MapClaims, key string) string {
	val, _ := claims[key]
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

func readAudience(raw any) []string {
	switch v := raw.(type) {
	case string:
		return []string{v}
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return v
	default:
		return nil
	}
}

func readExpiry(raw any) time.Time {
	switch v := raw.(type) {
	case float64:
		return time.Unix(int64(v), 0)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return time.Unix(i, 0)
		}
	case int64:
		return time.Unix(v, 0)
	}
	return time.Time{}
}

// AuthDisabled reports whether auth should be skipped for local development.
func AuthDisabled() bool {
	if strings.EqualFold(os.Getenv("AUTH_DISABLED"), "true") {
		if strings.EqualFold(os.Getenv("ENV"), "local") || os.Getenv("AWS_LAMBDA_FUNCTION_NAME") == "" {
			log.Print("auth disabled via AUTH_DISABLED for local development")
			return true
		}
	}
	return false
}
