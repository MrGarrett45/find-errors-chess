package main

import (
	"context"
	"encoding/json"
	"log"

	"example/my-go-api/app"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
)

var ginLambda *ginadapter.GinLambda
var ginLambdaV2 *ginadapter.GinLambdaV2

// init runs once per Lambda container (cold start)
func init() {
	// Initialize DB connection pool
	app.MustInitDB()
	app.InitStripe()

	// Set up Gin router
	router, err := app.NewRouter()
	if err != nil {
		log.Fatalf("failed to initialize router: %v", err)
	}

	// Wrap Gin router with Lambda adapter
	ginLambda = ginadapter.New(router)
	ginLambdaV2 = ginadapter.NewV2(router)
}

// Handler is the Lambda entrypoint for API Gateway REST/HTTP API (proxy integration)
func Handler(ctx context.Context, payload json.RawMessage) (any, error) {
	if isV2Event(payload) {
		var req events.APIGatewayV2HTTPRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			return events.APIGatewayV2HTTPResponse{StatusCode: 500, Body: `{"error":"invalid request"}`}, nil
		}
		return ginLambdaV2.ProxyWithContext(ctx, req)
	}

	var req events.APIGatewayProxyRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: `{"error":"invalid request"}`}, nil
	}
	return ginLambda.ProxyWithContext(ctx, req)
}

func main() {
	lambda.Start(Handler)
}

type v2Probe struct {
	Version        string `json:"version"`
	RequestContext struct {
		HTTP any `json:"http"`
	} `json:"requestContext"`
}

func isV2Event(payload json.RawMessage) bool {
	if len(payload) == 0 {
		return false
	}
	var probe v2Probe
	if err := json.Unmarshal(payload, &probe); err != nil {
		return false
	}
	if probe.Version == "2.0" {
		return true
	}
	return probe.RequestContext.HTTP != nil
}
