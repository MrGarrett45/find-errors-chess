// Package auth tests JWT middleware behavior against a mock JWKS.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestMiddlewareMissingToken(t *testing.T) {
	t.Setenv("AUTH_DISABLED", "false")
	verifier, _ := newTestVerifier(t)

	router := gin.New()
	router.Use(Middleware(verifier, MiddlewareConfig{}))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}
}

func TestMiddlewareMalformedHeader(t *testing.T) {
	t.Setenv("AUTH_DISABLED", "false")
	verifier, _ := newTestVerifier(t)

	router := gin.New()
	router.Use(Middleware(verifier, MiddlewareConfig{}))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Token abc")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}
}

func TestMiddlewareInvalidToken(t *testing.T) {
	t.Setenv("AUTH_DISABLED", "false")
	verifier, _ := newTestVerifier(t)

	badKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	tokenString := signToken(t, badKey, "test-key", verifier.issuer, verifier.audience)

	router := gin.New()
	router.Use(Middleware(verifier, MiddlewareConfig{}))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}

}

func TestMiddlewareValidToken(t *testing.T) {
	t.Setenv("AUTH_DISABLED", "false")
	verifier, key := newTestVerifier(t)
	tokenString := signToken(t, key, "test-key", verifier.issuer, verifier.audience)

	router := gin.New()
	router.Use(Middleware(verifier, MiddlewareConfig{}))
	router.GET("/protected", func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c.Request.Context())
		if !ok || claims.Subject == "" {
			c.Status(http.StatusUnauthorized)
			return
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func newTestVerifier(t *testing.T) (*Verifier, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	jwks := newJWKS(key, "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	t.Cleanup(server.Close)

	issuer := "https://example.auth0.com/"
	audience := "https://api.example"
	verifier, err := NewVerifier(issuer, audience, server.URL)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}
	return verifier, key
}

func signToken(t *testing.T, key *rsa.PrivateKey, kid, issuer, audience string) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   issuer,
		"aud":   audience,
		"sub":   "user-123",
		"scope": "read:me",
		"exp":   now.Add(10 * time.Minute).Unix(),
		"iat":   now.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	tokenString, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return tokenString
}

type jwksPayload struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func newJWKS(key *rsa.PrivateKey, kid string) jwksPayload {
	n := base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes())
	return jwksPayload{
		Keys: []jwk{
			{
				Kty: "RSA",
				Kid: kid,
				Use: "sig",
				Alg: "RS256",
				N:   n,
				E:   e,
			},
		},
	}
}

func TestHasScopes(t *testing.T) {
	if hasScopes("read:me write:me", []string{"read:me"}) != true {
		t.Fatalf("expected required scope")
	}
	if hasScopes("read:me", []string{"write:me"}) {
		t.Fatalf("expected missing scope")
	}
}

func TestExtractBearerToken(t *testing.T) {
	token, ok := extractBearerToken("Bearer abc")
	if !ok || token != "abc" {
		t.Fatalf("expected token")
	}
	if _, ok := extractBearerToken("Bearer"); ok {
		t.Fatalf("expected invalid header")
	}
	if _, ok := extractBearerToken("Token abc"); ok {
		t.Fatalf("expected invalid scheme")
	}
	if _, ok := extractBearerToken(""); ok {
		t.Fatalf("expected empty header to be invalid")
	}
}

func TestClaimsFromContext(t *testing.T) {
	claims := &Claims{Subject: "user-1"}
	ctx := WithClaims(context.Background(), claims)
	got, ok := ClaimsFromContext(ctx)
	if !ok || got.Subject != "user-1" {
		t.Fatalf("expected claims from context")
	}
}
