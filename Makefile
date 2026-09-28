BINARY := bin/api
MODULE := github.com/matheus-aicorp/sample-test-project
PKGS   := ./internal/...

.PHONY: help build run test cover mocks fmt vet check tidy clean

help: ## List available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

build: ## Compile the API binary into bin/api
	go build -o $(BINARY) ./cmd/api

run: ## Start the API on :8080
	go run ./cmd/api

test: ## Run the unit tests with the race detector
	go test $(PKGS) -race

cover: ## Report coverage across all internal packages
	go test $(PKGS) -race -covermode=atomic -coverpkg=$(PKGS) -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1

cover-html: cover ## Write an HTML coverage report to coverage.html
	go tool cover -html=coverage.out -o coverage.html

mocks: ## Regenerate the gomock mocks
	go tool mockgen -destination=internal/mocks/mock_domain.go -package=mocks $(MODULE)/internal/domain CapitalRepository,WeatherProvider,Clock
	go tool mockgen -destination=internal/mocks/mock_rest.go -package=mocks $(MODULE)/internal/adapter/driving/rest WeatherQuery
	go tool mockgen -destination=internal/mocks/mock_openmeteo.go -package=mocks $(MODULE)/internal/adapter/driven/openmeteo Doer

fmt: ## Format the source tree
	gofmt -l -w .

vet: ## Run go vet
	go vet ./...

check: fmt vet test ## Format, vet and test

tidy: ## Sync go.mod and go.sum
	go mod tidy

clean: ## Remove build and coverage artifacts
	rm -rf bin coverage.out coverage.html
