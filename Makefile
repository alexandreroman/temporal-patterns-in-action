.DEFAULT_GOAL := help

##@ Infra

.PHONY: infra-up
infra-up: ## Start local Temporal server and NATS (docker-compose)
	docker-compose up -d temporal nats

.PHONY: infra-down
infra-down: ## Stop local Temporal server and NATS
	docker-compose stop temporal nats

.PHONY: infra-logs
infra-logs: ## Follow Temporal server logs
	docker-compose logs -f temporal

##@ Modules

.PHONY: frontend
frontend: ## Run the frontend dev server
	$(MAKE) -C frontend dev

.PHONY: worker-%
worker-%: ## Run a pattern worker (e.g. make worker-saga)
	$(MAKE) -C workers run-$*

.PHONY: dev-workers
dev-workers: ## Run every pattern worker with hot-reload (requires Air)
	$(MAKE) -C workers dev-all

.PHONY: dev
dev: ## Run the frontend and all workers in parallel with hot-reload
	@$(MAKE) -j frontend dev-workers

.PHONY: check
check: ## Run all checks across modules
	$(MAKE) -C frontend check
	$(MAKE) -C workers check

##@ Helpers

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
