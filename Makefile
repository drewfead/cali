# Makefile for cal development

# Binary output directory (gitignored)
BIN_DIR := bin

# Installation directory (can be overridden via command line)
INSTALL_LOCATION ?= ~/bin

##@ Build

.PHONY: build
build: ## Build the cali binary
	@echo "Building cali..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/cali .
	@echo "✓ Built: $(BIN_DIR)/cali"

.PHONY: install
install: build ## Build and install cal to ~/bin (override with INSTALL_LOCATION=/path)
	@echo "Installing cali to $(INSTALL_LOCATION)..."
	@mkdir -p $(INSTALL_LOCATION)
	cp $(BIN_DIR)/cali $(INSTALL_LOCATION)/cali
	@echo "✓ Installed: $(INSTALL_LOCATION)/cali"

.PHONY: clean
clean: ## Clean build artifacts and generated proto files
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR)/
	rm -f proto/*.pb.go
	go clean
	@echo "✓ Clean complete"

##@ Proto

.PHONY: generate
generate: ## Generate proto files using buf
	@echo "Generating proto files..."
	go generate ./...
	@echo "✓ Proto generation complete"

.PHONY: generate/clean
generate/clean: ## Clean and regenerate all proto files
	@echo "Cleaning generated proto files..."
	rm -f proto/*.pb.go
	@echo "Regenerating proto files..."
	go generate ./...
	@echo "✓ Clean regeneration complete"

##@ Test

.PHONY: test
test: ## Run all tests
	go test -v -race ./...

.PHONY: test/unit
test/unit: ## Run unit tests only
	go test -v -race -run "^TestUnit_" ./...

.PHONY: test/integration
test/integration: ## Run integration tests only
	go test -v -race -run "^TestIntegration_" ./...

##@ Lint

.PHONY: lint
lint: ## Run linter on all files
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint run ./...

.PHONY: fmt
fmt: ## Auto-format code
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint fmt ./...
	go run mvdan.cc/gofumpt -l -w .

##@ Misc.

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php
.PHONY: help
help: ## Display usage help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9\/-]+:.*?##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
