.PHONY: build run test test-verbose test-coverage test-unit test-race clean docker docker-up docker-down deps fmt lint

# Variables
BINARY_NAME=open-atlas-search
DOCKER_IMAGE=open-atlas-search
VERSION=latest

# Build the application
build:
	go build -o $(BINARY_NAME) .

# Run the application
run:
	go run . server

# Run tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run unit tests only
test-unit:
	go test -short ./...

# Run tests with race detection
test-race:
	go test -race ./...

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -rf indexes/
	rm -f coverage.out coverage.html
	rm -f sync_state.json*

# Install dependencies
deps:
	go mod download
	go mod tidy

# Build Docker image
docker:
	docker build -t $(DOCKER_IMAGE):$(VERSION) .

# Start services with Docker Compose
docker-up:
	docker-compose up -d

# Stop services with Docker Compose
docker-down:
	docker-compose down

# View logs
logs:
	docker-compose logs -f open-atlas-search

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o $(BINARY_NAME)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe .

# Install the application
install:
	go install .

# Development mode - rebuild and restart on changes
dev:
	air -c .air.toml
