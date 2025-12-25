// Package auth provides request context helpers for verified Auth0 claims.
package auth

import (
	"context"
	"time"
)

type ctxKey int

const claimsKey ctxKey = iota

// Claims contains the verified Auth0 token details we care about.
type Claims struct {
	Subject   string
	Issuer    string
	Audience  []string
	ExpiresAt time.Time
	Scope     string
	Raw       map[string]any
}

// WithClaims stores auth claims in a context.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ClaimsFromContext returns claims from a context.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}
