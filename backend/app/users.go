// Package app provides user persistence helpers for authenticated requests.
package app

import (
	"context"
	"database/sql"
	"strings"

	"example/my-go-api/auth"
)

// UpsertUserFromClaims creates a user row if it does not already exist.
func UpsertUserFromClaims(ctx context.Context, claims *auth.Claims) error {
	if db == nil {
		return nil
	}
	if claims == nil || claims.Subject == "" {
		return nil
	}

	email := readStringClaim(claims.Raw, "email")
	name := readStringClaim(claims.Raw, "name")

	const q = `
		INSERT INTO users (auth0_sub, email, name, last_login)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (auth0_sub) DO NOTHING;
	`

	_, err := db.ExecContext(ctx, q, claims.Subject, nullIfEmpty(email), nullIfEmpty(name))
	return err
}

func readStringClaim(raw map[string]any, key string) string {
	if raw == nil {
		return ""
	}
	val, ok := raw[key]
	if !ok {
		return ""
	}
	if s, ok := val.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func nullIfEmpty(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
