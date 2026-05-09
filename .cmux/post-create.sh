#!/usr/bin/env bash
#
# cmux post-create hook — initializes a freshly-created feature worktree.
# Invoked by /cmux:start-feature with cwd = feature worktree.
#
# Env exported by cmux:
#   CMUX_FEATURE_SLUG     — short slug, e.g. "auth-jwt"
#   CMUX_FEATURE_BRANCH   — "feature/<slug>"
#   CMUX_FEATURE_WORKTREE — absolute path of the new worktree
#   CMUX_MAIN_WORKTREE    — absolute path of the main worktree

set -euo pipefail

log() {
  printf '[post-create] %s\n' "$*"
  if command -v cmux >/dev/null 2>&1 && [ -n "${CMUX_WORKSPACE_ID:-}" ]; then
    cmux log --level progress --source "post-create" -- "$*" \
      >/dev/null 2>&1 || true
  fi
}

log "Initializing feature/${CMUX_FEATURE_SLUG:-?} in $(pwd)"

# Frontend — node_modules/ is gitignored, so each worktree needs its own
# install. corepack pins the pnpm version declared in package.json.
if [ -f frontend/package.json ]; then
  log "Enabling pnpm via corepack"
  corepack enable >/dev/null 2>&1 || true

  log "Installing frontend dependencies"
  cd frontend
  pnpm install --frozen-lockfile
  cd ..
fi

# Workers — pre-warm the Go module cache so the first build/run in this
# worktree does not stall on network.
if [ -f workers/go.mod ]; then
  log "Downloading Go modules"
  cd workers
  go mod download
  cd ..
fi

log "Done"
