.DEFAULT_GOAL := help

# Known patterns — each has a dedicated binary under <pattern>/cmd/worker.
PATTERNS := saga

##@ Development

.PHONY: run
run: ## Show per-pattern run targets
	@echo "Pick a pattern to run:"
	@for p in $(PATTERNS); do echo "  make run-$$p"; done

.PHONY: run-saga
run-saga: ## Run the saga worker
	go run ./saga/cmd/worker

.PHONY: dev
dev: ## Show per-pattern dev targets (hot-reload)
	@echo "Pick a pattern to hot-reload:"
	@for p in $(PATTERNS); do echo "  make dev-$$p"; done

.PHONY: dev-saga
dev-saga: ## Run the saga worker with hot-reload (requires Air)
	cd saga && air

.PHONY: build
build: ## Build all pattern binaries into bin/
	@mkdir -p bin
	@for p in $(PATTERNS); do \
		echo "building bin/$$p"; \
		go build -o bin/$$p ./$$p/cmd/worker || exit 1; \
	done

.PHONY: build-saga
build-saga: ## Build the saga binary into bin/saga
	@mkdir -p bin
	go build -o bin/saga ./saga/cmd/worker

##@ Quality

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: lint
lint: ## Run linter (requires golangci-lint)
	golangci-lint run ./...

.PHONY: format
format: ## Format code
	gofmt -w .
	goimports -w .

.PHONY: vet
vet: ## Run static analysis
	go vet ./...

.PHONY: workflowcheck
workflowcheck: ## Check for non-deterministic workflow code
	workflowcheck ./...

.PHONY: check
check: vet lint test ## Run all checks

##@ Helpers

.PHONY: tidy
tidy: ## Tidy and verify module dependencies
	go mod tidy
	go mod verify

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
