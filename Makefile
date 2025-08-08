.PHONY: help build test test-fast test-integration fmt lint vet install clean release

APP_NAME=flickr
VERSION=$(shell cat VERSION 2>/dev/null || echo "0.1.0")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

VERSION_PKG=github.com/yourorg/flickr/internal/version
LD_FLAGS=-X '$(VERSION_PKG).Version=$(VERSION)' \
         -X '$(VERSION_PKG).Commit=$(COMMIT)' \
         -X '$(VERSION_PKG).BuildTime=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")'

GO_FLAGS=-ldflags "$(LD_FLAGS)"
GO=$(shell which go)
BIN=./bin
RELEASE_DIR=./release

# ANSI color codes
CYAN=\033[36m
RESET=\033[0m
GREEN=\033[32m
YELLOW=\033[33m

help: ## Show available commands
	@echo "$(CYAN)Flickr - Docker runner for AVS releases$(RESET)"
	@echo ""
	@echo "$(GREEN)Available commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'

build: ## Build the flickr binary
	@echo "$(GREEN)Building $(APP_NAME) v$(VERSION)...$(RESET)"
	@mkdir -p $(BIN)
	@$(GO) build $(GO_FLAGS) -o $(BIN)/$(APP_NAME) cmd/flickr/main.go
	@echo "$(GREEN)✓ Binary built at $(BIN)/$(APP_NAME)$(RESET)"

test: ## Run all tests including integration tests
	@echo "$(GREEN)Running all tests...$(RESET)"
	@$(GO) test -v ./... -timeout 60s
	@echo "$(GREEN)✓ All tests passed$(RESET)"

test-fast: ## Run fast tests (skip integration tests)
	@echo "$(GREEN)Running fast tests...$(RESET)"
	@$(GO) test -v ./... -short -timeout 5m
	@echo "$(GREEN)✓ Fast tests passed$(RESET)"

test-integration: ## Run Docker integration tests only
	@echo "$(YELLOW)Running Docker integration tests...$(RESET)"
	@echo "$(YELLOW)Note: Requires Docker to be installed and running$(RESET)"
	@$(GO) test -v ./internal/controller -run TestRealDocker -timeout 120s
	@echo "$(GREEN)✓ Integration tests passed$(RESET)"

test-unit: ## Run unit tests only (no integration)
	@echo "$(GREEN)Running unit tests...$(RESET)"
	@$(GO) test -v ./internal/ref ./internal/controller -run '^Test[^R][^e][^a][^l]' -timeout 30s
	@echo "$(GREEN)✓ Unit tests passed$(RESET)"

fmt: ## Format Go code
	@echo "$(GREEN)Formatting code...$(RESET)"
	@$(GO) fmt ./...
	@echo "$(GREEN)✓ Code formatted$(RESET)"

lint: ## Run golangci-lint
	@echo "$(GREEN)Running linter...$(RESET)"
	@if ! which golangci-lint > /dev/null; then \
		echo "$(YELLOW)Installing golangci-lint...$(RESET)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@golangci-lint run ./...
	@echo "$(GREEN)✓ Linting passed$(RESET)"

vet: ## Run go vet
	@echo "$(GREEN)Running go vet...$(RESET)"
	@$(GO) vet ./...
	@echo "$(GREEN)✓ Vet passed$(RESET)"

install: build ## Install binary to ~/bin
	@echo "$(GREEN)Installing $(APP_NAME) to ~/bin...$(RESET)"
	@mkdir -p ~/bin
	@cp $(BIN)/$(APP_NAME) ~/bin/
	@echo "$(GREEN)✓ Installed to ~/bin/$(APP_NAME)$(RESET)"
	@echo ""
	@echo "$(YELLOW)Make sure ~/bin is in your PATH:$(RESET)"
	@echo '  export PATH=$$PATH:~/bin'

clean: ## Remove built binaries and artifacts
	@echo "$(GREEN)Cleaning...$(RESET)"
	@rm -rf $(BIN) $(RELEASE_DIR)
	@rm -f ~/bin/$(APP_NAME)
	@echo "$(GREEN)✓ Cleaned$(RESET)"

deps: ## Download and tidy dependencies
	@echo "$(GREEN)Downloading dependencies...$(RESET)"
	@$(GO) mod download
	@$(GO) mod tidy
	@echo "$(GREEN)✓ Dependencies updated$(RESET)"

# Cross-platform builds
build-darwin-arm64: ## Build for macOS ARM64
	@echo "$(GREEN)Building for Darwin ARM64...$(RESET)"
	@mkdir -p $(RELEASE_DIR)/darwin-arm64
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(GO_FLAGS) \
		-o $(RELEASE_DIR)/darwin-arm64/$(APP_NAME) cmd/flickr/main.go
	@echo "$(GREEN)✓ Built darwin-arm64$(RESET)"

build-darwin-amd64: ## Build for macOS AMD64
	@echo "$(GREEN)Building for Darwin AMD64...$(RESET)"
	@mkdir -p $(RELEASE_DIR)/darwin-amd64
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(GO_FLAGS) \
		-o $(RELEASE_DIR)/darwin-amd64/$(APP_NAME) cmd/flickr/main.go
	@echo "$(GREEN)✓ Built darwin-amd64$(RESET)"

build-linux-arm64: ## Build for Linux ARM64
	@echo "$(GREEN)Building for Linux ARM64...$(RESET)"
	@mkdir -p $(RELEASE_DIR)/linux-arm64
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(GO_FLAGS) \
		-o $(RELEASE_DIR)/linux-arm64/$(APP_NAME) cmd/flickr/main.go
	@echo "$(GREEN)✓ Built linux-arm64$(RESET)"

build-linux-amd64: ## Build for Linux AMD64
	@echo "$(GREEN)Building for Linux AMD64...$(RESET)"
	@mkdir -p $(RELEASE_DIR)/linux-amd64
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GO_FLAGS) \
		-o $(RELEASE_DIR)/linux-amd64/$(APP_NAME) cmd/flickr/main.go
	@echo "$(GREEN)✓ Built linux-amd64$(RESET)"

release: ## Build all release binaries
	@echo "$(CYAN)Building release binaries for v$(VERSION)...$(RESET)"
	@$(MAKE) build-darwin-arm64
	@$(MAKE) build-darwin-amd64
	@$(MAKE) build-linux-arm64
	@$(MAKE) build-linux-amd64
	@echo ""
	@echo "$(GREEN)✓ All release binaries built in $(RELEASE_DIR)/$(RESET)"
	@echo ""
	@echo "$(CYAN)Release artifacts:$(RESET)"
	@ls -la $(RELEASE_DIR)/*/$(APP_NAME) | awk '{print "  " $$9}'

verify: fmt vet test-fast ## Run all verification steps (fmt, vet, test-fast)
	@echo "$(GREEN)✓ All verification steps passed$(RESET)"

coverage: ## Run tests with coverage
	@echo "$(GREEN)Running tests with coverage...$(RESET)"
	@$(GO) test -v -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report generated: coverage.html$(RESET)"

run: build ## Build and run the CLI (pass args with ARGS="...")
	@echo "$(GREEN)Running $(APP_NAME)...$(RESET)"
	@$(BIN)/$(APP_NAME) $(ARGS)

docker-test: ## Run a quick Docker test to verify Docker is working
	@echo "$(GREEN)Testing Docker setup...$(RESET)"
	@docker run --rm hello-world > /dev/null 2>&1 && \
		echo "$(GREEN)✓ Docker is working$(RESET)" || \
		echo "$(YELLOW)✗ Docker is not running or not installed$(RESET)"

.DEFAULT_GOAL := help