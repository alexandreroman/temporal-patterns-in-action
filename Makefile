.DEFAULT_GOAL := app-up

# Per-worktree port isolation. When Casper assigns this workspace a
# CASPER_PORT, the `compose-override` target (a dep of bootstrap, infra-up and
# app-up) remaps the container host ports via compose.override.yaml: frontend
# on CASPER_PORT, Temporal gRPC +1, Temporal Web UI +2, NATS +3 and Codec
# Server +4. So even `make dev` alone brings infra up on the matching ports.
#
# The exports mirror that for the host-run dev processes, so several worktrees
# can `make dev` at once: the frontend listens on CASPER_PORT and both frontend
# and workers reach the remapped infra. compose.yaml hardcodes the in-container
# addresses, so exporting these never affects the containers.
#
# The *_PORT variables are the host ports advertised by `make endpoints` and by
# the workspace info panel that mirrors it. Unset CASPER_PORT keeps the
# compose.yaml defaults.

ifneq ($(CASPER_PORT),)
export PORT             := $(CASPER_PORT)
export TEMPORAL_ADDRESS := localhost:$(shell expr $(CASPER_PORT) + 1)
export NATS_URL         := nats://localhost:$(shell expr $(CASPER_PORT) + 3)
FRONTEND_PORT           := $(CASPER_PORT)
TEMPORAL_UI_PORT        := $(shell expr $(CASPER_PORT) + 2)
CODEC_SERVER_PORT       := $(shell expr $(CASPER_PORT) + 4)
else
FRONTEND_PORT           := 3000
TEMPORAL_UI_PORT        := 8233
CODEC_SERVER_PORT       := 8888
endif

##@ Infra

.PHONY: infra-up
infra-up: compose-override ## Start local Temporal server and NATS (docker-compose)
	docker-compose up -d temporal nats

# This stops the Temporal server, so the Web UI address the panel advertises
# stops answering even though the frontend container may still be running.
.PHONY: infra-down
infra-down: ## Stop local Temporal server and NATS
	docker-compose stop temporal nats
	@$(clear-endpoints)

.PHONY: infra-logs
infra-logs: ## Follow Temporal server logs
	docker-compose logs -f temporal

##@ App

# Markdown on stdout, so the answer to "where is this worktree listening?" can
# be read in a terminal or piped into whatever renders it. The Codec Server row
# is an address to paste rather than one the stack wires up for you, so the
# line below the table says where it goes.
.PHONY: endpoints
endpoints: ## Print this worktree's published endpoints as Markdown
	@printf '%s\n' \
		'# Temporal Patterns in Action' \
		'' \
		'| Service | Address |' \
		'| --- | --- |' \
		'| App | <http://localhost:$(FRONTEND_PORT)> |' \
		'| Temporal Web UI | <http://localhost:$(TEMPORAL_UI_PORT)> |' \
		'| Codec Server | <http://localhost:$(CODEC_SERVER_PORT)> |' \
		'' \
		'The Codec Server is opt-in: paste its address into the Temporal Web UI' \
		'(glasses icon, top bar) to decrypt encryption-pattern payloads.'

# The workspace info panel mirrors `make endpoints`, so whichever command
# brought the stack up or down leaves it telling the truth. The casper CLI is on
# PATH outside a workspace too, hence the CASPER_WORKSPACE_ID test alongside it,
# and `|| true` keeps a panel update from ever failing the target that asked for
# it.
in-casper-workspace = [ -n "$$CASPER_WORKSPACE_ID" ] && command -v casper >/dev/null 2>&1

define publish-endpoints
$(in-casper-workspace) && $(MAKE) -s endpoints | casper info set - >/dev/null || true
endef

define clear-endpoints
$(in-casper-workspace) && casper info clear >/dev/null || true
endef

# The endpoints are printed and published first, because `docker-compose up`
# runs attached: it streams container logs, never returns until Ctrl-C, and
# nothing else reaches the terminal once it takes over.
.PHONY: app-up
app-up: compose-override ## Build and run the full stack (infra, frontend, workers) in containers
	@$(MAKE) -s endpoints
	@$(publish-endpoints)
	docker-compose up --build

.PHONY: app-down
app-down: ## Stop the full stack and remove containers
	docker-compose down
	@$(clear-endpoints)

.PHONY: app-logs
app-logs: ## Follow logs from every container
	docker-compose logs -f

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
dev: infra-up ## Start infra, then run the frontend and all workers in parallel with hot-reload
	@$(MAKE) -s endpoints
	@$(publish-endpoints)
	@$(MAKE) -j frontend dev-workers

.PHONY: check
check: ## Run all checks across modules
	$(MAKE) -C frontend check
	$(MAKE) -C workers check
	cd codec-server && go vet ./... && go build ./...

.PHONY: setup
setup: ## Install the versioned git hooks (one-time, per clone)
	git config core.hooksPath .githooks
	@echo "git hooks: core.hooksPath -> .githooks"

##@ Workspace

.PHONY: bootstrap
bootstrap: compose-override ## Prepare a fresh worktree: install deps and pin container host ports to $CASPER_PORT
	cd frontend && corepack enable || true
	cd frontend && pnpm install --frozen-lockfile
	cd workers && go mod download

.PHONY: compose-override
compose-override: ## Pin container host ports to $CASPER_PORT (compose.override.yaml); no-op when unset
	@if [ -z "$${CASPER_PORT:-}" ] && [ -f compose.override.yaml ]; then \
		echo "warning: CASPER_PORT unset, but compose.override.yaml still remaps the host ports" \
			"(and 'make endpoints' shows the defaults); delete it to use the default ports" >&2; \
	fi
	@base="$${CASPER_PORT:-}"; \
	case "$$base" in \
		"" ) echo "CASPER_PORT unset — keeping default host ports"; exit 0 ;; \
		*[!0-9]* ) echo "CASPER_PORT='$$base' is not numeric — keeping default host ports"; exit 0 ;; \
	esac; \
	echo "Pinning host ports to $$base..$$((base+4)) in compose.override.yaml"; \
	{ \
		echo "# Generated by 'make compose-override' from CASPER_PORT — DO NOT COMMIT (gitignored)."; \
		echo "# '!override' replaces the base ports list (Compose concatenates by default)."; \
		echo "services:"; \
		echo "  frontend:"; \
		echo "    ports: !override [\"$$base:3000\"]"; \
		echo "  temporal:"; \
		echo "    ports: !override [\"$$((base+1)):7233\", \"$$((base+2)):8233\"]"; \
		echo "  nats:"; \
		echo "    ports: !override [\"$$((base+3)):4222\"]"; \
		echo "  codec-server:"; \
		echo "    ports: !override [\"$$((base+4)):8888\"]"; \
		echo "    environment:"; \
		echo "      UI_ORIGIN: \"http://localhost:$$((base+2)),http://127.0.0.1:$$((base+2))\""; \
	} > compose.override.yaml

.PHONY: teardown
teardown: app-down ## Stop this worktree's containers

##@ Helpers

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z_%-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
