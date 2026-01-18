// Package models defines user plan and usage tracking fields.
package models

import "time"

type Plan string

const (
	PlanFree Plan = "FREE"
	PlanPro  Plan = "PRO"
)

type User struct {
	Auth0Sub         string    `db:"auth0_sub"`
	Plan             Plan      `db:"plan"`
	AnalysesUsed     int       `db:"analyses_used"`
	UsagePeriodStart time.Time `db:"usage_period_start"`
}
