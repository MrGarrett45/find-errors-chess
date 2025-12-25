// Package auth provides Gin middleware for enforcing Auth0 JWT auth.
package auth

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// MiddlewareConfig controls auth enforcement behavior.
type MiddlewareConfig struct {
	RequireScopes []string
	PublicPaths   map[string]bool
	DisableAuth   bool
}

// Middleware enforces bearer token auth and injects claims into the request context.
func Middleware(verifier *Verifier, cfg MiddlewareConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.DisableAuth || AuthDisabled() {
			claims := &Claims{
				Subject: "local-dev",
				Issuer:  "local",
				Raw:     map[string]any{"sub": "local-dev"},
			}
			ctx := WithClaims(c.Request.Context(), claims)
			c.Request = c.Request.WithContext(ctx)
			c.Next()
			return
		}

		if cfg.PublicPaths != nil && cfg.PublicPaths[c.FullPath()] {
			c.Next()
			return
		}

		if verifier == nil {
			respondUnauthorized(c, "auth verifier not configured")
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Printf("auth failure: missing Authorization header path=%s", c.Request.URL.Path)
			respondUnauthorized(c, "missing authorization header")
			return
		}

		token, ok := extractBearerToken(authHeader)
		if !ok {
			log.Printf("auth failure: malformed Authorization header path=%s", c.Request.URL.Path)
			respondUnauthorized(c, "invalid authorization header")
			return
		}

		claims, err := verifier.Verify(token)
		if err != nil {
			log.Printf("auth failure: token invalid path=%s err=%v", c.Request.URL.Path, err)
			respondUnauthorized(c, "invalid token")
			return
		}

		if len(cfg.RequireScopes) > 0 && !hasScopes(claims.Scope, cfg.RequireScopes) {
			log.Printf("auth failure: missing scopes path=%s", c.Request.URL.Path)
			respondUnauthorized(c, "insufficient scope")
			return
		}

		ctx := WithClaims(c.Request.Context(), claims)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func extractBearerToken(header string) (string, bool) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return "", false
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}

func hasScopes(scopeClaim string, required []string) bool {
	if scopeClaim == "" {
		return false
	}
	available := map[string]struct{}{}
	for _, s := range strings.Fields(scopeClaim) {
		available[s] = struct{}{}
	}
	for _, scope := range required {
		if _, ok := available[scope]; !ok {
			return false
		}
	}
	return true
}

func respondUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": message,
	})
}
