MODULE := github.com/matheus-aicorp/sample-test-project
PKGS   := ./internal/...

# Build and coverage artifacts are written outside the repository on purpose:
# running a target must never add files to the working tree.
ARTIFACTS ?= $(or $(TMPDIR),/tmp)/sample-test-project
BINARY    := $(ARTIFACTS)/api

.PHONY: help build run test cover cover-html mocks fmt vet check tidy clean

help: ## List available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

build: ## Compile the API binary into $(ARTIFACTS)
	@mkdir -p $(ARTIFACTS)
	go build -o $(BINARY) ./cmd/api
	@echo "wrote $(BINARY)"

run: ## Start the API on :8080 (leaves no artifact behind)
	go run ./cmd/api

test: ## Run the unit tests with the race detector
	go test $(PKGS) -race

cover: ## Report coverage across all internal packages
	@mkdir -p $(ARTIFACTS)
	go test $(PKGS) -race -covermode=atomic -coverpkg=$(PKGS) -coverprofile=$(ARTIFACTS)/coverage.out
	go tool cover -func=$(ARTIFACTS)/coverage.out | tail -1
	@echo "profile: $(ARTIFACTS)/coverage.out"

cover-html: cover ## Write an HTML coverage report into $(ARTIFACTS)
	go tool cover -html=$(ARTIFACTS)/coverage.out -o $(ARTIFACTS)/coverage.html
	@echo "report: $(ARTIFACTS)/coverage.html"

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

clean: ## Remove generated artifacts from $(ARTIFACTS)
	rm -rf $(ARTIFACTS)
