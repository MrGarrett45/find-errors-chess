# syntax=docker/dockerfile:1

########################
# 1. Build stage
########################
FROM golang:1.25 AS build

WORKDIR /app

# Pre-download modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of your source code
COPY . .

# Build specifically the cmd/analyze main
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o analyze ./cmd/analyze-container

########################
# 2. Runtime stage
########################
FROM debian:bullseye-slim

# Install Stockfish + certs for HTTPS DB connections
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    stockfish \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the compiled binary from the build stage
COPY --from=build /app/analyze /app/analyze

# Default environment variables (override with docker run or compose)
ENV PORT=8080

# Expose the app port
EXPOSE 8080

# Start your Go program
CMD ["/app/analyze"]
