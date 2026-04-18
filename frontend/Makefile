.DEFAULT_GOAL := help

##@ Development

.PHONY: install
install: ## Install dependencies
	pnpm install

.PHONY: dev
dev: ## Start Nuxt dev server
	pnpm dev

.PHONY: build
build: ## Build for production
	pnpm build

.PHONY: preview
preview: ## Preview the production build
	pnpm preview

##@ Quality

.PHONY: typecheck
typecheck: ## Type-check the project
	pnpm typecheck

.PHONY: lint
lint: ## Run ESLint
	pnpm lint

.PHONY: lint-fix
lint-fix: ## Run ESLint with auto-fix
	pnpm lint:fix

.PHONY: format
format: ## Format source files
	pnpm format

.PHONY: format-check
format-check: ## Check formatting
	pnpm format:check

.PHONY: check
check: typecheck lint format-check ## Run all checks (CI)

##@ Helpers

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf .nuxt .output .nitro node_modules/.cache

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
