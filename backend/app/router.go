// Package app wires shared HTTP routes for both local and Lambda execution.
package app

import (
	"time"

	"example/my-go-api/auth"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// NewRouter builds the shared HTTP router for both local and Lambda execution.
func NewRouter() (*gin.Engine, error) {
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		MaxAge:       12 * time.Hour,
	}))

	router.GET("/health", Health)

	verifier, err := auth.NewVerifierFromEnv()
	if err != nil && !auth.AuthDisabled() {
		return nil, err
	}

	protected := router.Group("/")
	protected.Use(auth.Middleware(verifier, auth.MiddlewareConfig{
		OnAuthenticated: func(c *gin.Context, claims *auth.Claims) error {
			return UpsertUserFromClaims(c.Request.Context(), claims)
		},
	}))
	protected.GET("/me", Me)
	protected.GET("/chessgames/:username", GetChessGames)
	protected.GET("/errors/:username", GetErrorPositions)
	protected.GET("/jobs/:jobid", GetJobStatus)

	return router, nil
}
