package main

import (
	"context"

	"example/my-go-api/app"

	"github.com/gin-gonic/gin"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
)

var ginLambda *ginadapter.GinLambda

// init runs once per Lambda container (cold start)
func init() {
	// Initialize DB connection pool
	app.MustInitDB()

	// Set up Gin router
	router := gin.Default()

	// Note the leading "/" (works better with API Gateway paths)
	router.GET("/chessgames/:username", app.GetChessGames)
	router.GET("/errors/:username", app.GetErrorPositions)

	// Wrap Gin router with Lambda adapter
	ginLambda = ginadapter.New(router)
}

// Handler is the Lambda entrypoint for API Gateway REST/HTTP API (proxy integration)
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return ginLambda.ProxyWithContext(ctx, req)
}

func main() {
	lambda.Start(Handler)
}
