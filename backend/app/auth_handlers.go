// Package app provides public health and authenticated identity endpoints.
package app

import (
	"database/sql"
	"net/http"
	"time"

	"example/my-go-api/app/models"
	"example/my-go-api/auth"

	"github.com/gin-gonic/gin"
)

// Health is a public health check endpoint.
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// Me returns weekly usage info for the authenticated user.
func Me(c *gin.Context) {
	claims, ok := auth.ClaimsFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing auth context"})
		return
	}

	if db == nil {
		c.JSON(http.StatusOK, gin.H{
			"plan":         models.PlanFree,
			"analysesUsed": 0,
			"weeklyLimit":  FreeWeeklyLimit,
			"remaining":    FreeWeeklyLimit,
		})
		return
	}

	user, err := getUserByAuth0Sub(c.Request.Context(), claims.Subject)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = UpsertUserFromClaims(c.Request.Context(), claims)
			user, err = getUserByAuth0Sub(c.Request.Context(), claims.Subject)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load user"})
			return
		}
	}

	currentWeekStart := weekStartUTC(time.Now())
	if user.UsagePeriodStart.Before(currentWeekStart) {
		user.AnalysesUsed = 0
		user.UsagePeriodStart = currentWeekStart
		_, _ = db.ExecContext(
			c.Request.Context(),
			`
				UPDATE users
				SET analyses_used = $1, usage_period_start = $2
				WHERE auth0_sub = $3;
			`,
			user.AnalysesUsed,
			user.UsagePeriodStart,
			claims.Subject,
		)
	}

	var weeklyLimit any = nil
	var remaining any = nil
	if user.Plan == models.PlanFree {
		weeklyLimit = FreeWeeklyLimit
		remainingCount := FreeWeeklyLimit - user.AnalysesUsed
		if remainingCount < 0 {
			remainingCount = 0
		}
		remaining = remainingCount
	}

	c.JSON(http.StatusOK, gin.H{
		"plan":         user.Plan,
		"analysesUsed": user.AnalysesUsed,
		"weeklyLimit":  weeklyLimit,
		"remaining":    remaining,
	})
}
