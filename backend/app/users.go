// Package app provides user persistence helpers for authenticated requests.
package app

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"example/my-go-api/app/models"
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
		INSERT INTO users (auth0_sub, email, name, last_login, plan, analyses_used, usage_period_start)
		VALUES ($1, $2, $3, now(), $4, $5, $6)
		ON CONFLICT (auth0_sub) DO NOTHING;
	`

	_, err := db.ExecContext(
		ctx,
		q,
		claims.Subject,
		nullIfEmpty(email),
		nullIfEmpty(name),
		models.PlanFree,
		0,
		weekStartUTC(time.Now()),
	)
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

func getUserByAuth0Sub(ctx context.Context, auth0Sub string) (models.User, error) {
	var user models.User
	err := db.QueryRowContext(ctx, `
		SELECT plan, analyses_used, usage_period_start
		FROM users
		WHERE auth0_sub = $1;
	`, auth0Sub).Scan(&user.Plan, &user.AnalysesUsed, &user.UsagePeriodStart)
	if err != nil {
		return models.User{}, err
	}
	user.Auth0Sub = auth0Sub
	return user, nil
}
