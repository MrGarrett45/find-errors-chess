// Package app enforces weekly analysis limits for authenticated users.
package app

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"example/my-go-api/app/models"
)

const FreeWeeklyLimit = 100

type quotaError struct {
	Limit int
	Used  int
}

func (e quotaError) Error() string {
	return "weekly quota exceeded"
}

func weekStartUTC(t time.Time) time.Time {
	t = t.UTC()
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	daysSinceMonday := weekday - 1
	start := t.AddDate(0, 0, -daysSinceMonday)
	return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
}

func enforceWeeklyQuota(ctx context.Context, auth0Sub string, add int) (models.User, error) {
	if db == nil {
		return models.User{}, nil
	}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return models.User{}, err
	}
	defer tx.Rollback()

	user, err := getUserForUpdate(ctx, tx, auth0Sub)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if err := insertDefaultUser(ctx, tx, auth0Sub); err != nil {
				return models.User{}, err
			}
			user, err = getUserForUpdate(ctx, tx, auth0Sub)
		}
		if err != nil {
			return models.User{}, err
		}
	}

	now := time.Now()
	currentWeekStart := weekStartUTC(now)
	resetUsage := user.UsagePeriodStart.Before(currentWeekStart)
	if resetUsage {
		user.AnalysesUsed = 0
		user.UsagePeriodStart = currentWeekStart
	}

	if add < 0 {
		add = 0
	}

	shouldUpdate := resetUsage
	if user.Plan == models.PlanFree {
		if user.AnalysesUsed+add > FreeWeeklyLimit {
			return user, quotaError{Limit: FreeWeeklyLimit, Used: user.AnalysesUsed}
		}
		user.AnalysesUsed += add
		shouldUpdate = true
	}

	if shouldUpdate {
		if err := updateUserUsage(ctx, tx, auth0Sub, user.AnalysesUsed, user.UsagePeriodStart); err != nil {
			return models.User{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return models.User{}, err
	}

	return user, nil
}

func getUserForUpdate(ctx context.Context, tx *sql.Tx, auth0Sub string) (models.User, error) {
	var user models.User
	err := tx.QueryRowContext(ctx, `
		SELECT plan, analyses_used, usage_period_start
		FROM users
		WHERE auth0_sub = $1
		FOR UPDATE;
	`, auth0Sub).Scan(&user.Plan, &user.AnalysesUsed, &user.UsagePeriodStart)
	if err != nil {
		return models.User{}, err
	}
	user.Auth0Sub = auth0Sub
	return user, nil
}

func insertDefaultUser(ctx context.Context, tx *sql.Tx, auth0Sub string) error {
	now := weekStartUTC(time.Now())
	_, err := tx.ExecContext(ctx, `
		INSERT INTO users (auth0_sub, plan, analyses_used, usage_period_start)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (auth0_sub) DO NOTHING;
	`, auth0Sub, models.PlanFree, 0, now)
	return err
}

func updateUserUsage(ctx context.Context, tx *sql.Tx, auth0Sub string, used int, start time.Time) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE users
		SET analyses_used = $1, usage_period_start = $2
		WHERE auth0_sub = $3;
	`, used, start, auth0Sub)
	return err
}
