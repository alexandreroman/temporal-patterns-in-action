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
#   CMUX_PORT             — first port of this workspace's port block
#   CMUX_PORT_END         — last port of this workspace's port block

set -euo pipefail

log() {
  printf '[post-create] %s\n' "$*"
  if command -v cmux >/dev/null 2>&1 && [ -n "${CMUX_WORKSPACE_ID:-}" ]; then
    cmux log --level progress --source "post-create" -- "$*" \
      >/dev/null 2>&1 || true
  fi
}

log "Initializing feature/${CMUX_FEATURE_SLUG:-?} in $(pwd)"

# The stack publishes five host ports, CMUX_PORT..CMUX_PORT+4 (see the
# compose-override target in the Makefile). Unset or non-numeric values are
# reported by `make bootstrap` itself.
case "${CMUX_PORT:-}" in
  "" | *[!0-9]*) ;;
  *)
    last_port=$((CMUX_PORT + 4))
    if [ -n "${CMUX_PORT_END:-}" ] && [ "$last_port" -gt "$CMUX_PORT_END" ]; then
      log "WARNING: port block ${CMUX_PORT}..${last_port} exceeds" \
        "CMUX_PORT_END=${CMUX_PORT_END}"
    fi
    ;;
esac

# `make bootstrap` installs the frontend and Go dependencies and writes the
# gitignored compose.override.yaml that remaps the host ports; the Makefile
# falls back to CMUX_PORT when CASPER_PORT is unset.
log "Running make bootstrap (host ports from CMUX_PORT=${CMUX_PORT:-unset})"
make bootstrap

log "Done"
