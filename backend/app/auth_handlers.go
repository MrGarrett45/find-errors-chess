// Package app provides public health and authenticated identity endpoints.
package app

import (
	"example/my-go-api/auth"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health is a public health check endpoint.
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// Me returns the current authenticated user identity.
func Me(c *gin.Context) {
	claims, ok := auth.ClaimsFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing auth context"})
		return
	}

	response := gin.H{
		"sub":   claims.Subject,
		"iss":   claims.Issuer,
		"aud":   claims.Audience,
		"scope": claims.Scope,
	}

	if !claims.ExpiresAt.IsZero() {
		response["exp"] = claims.ExpiresAt.Unix()
	}

	c.JSON(http.StatusOK, response)
}
