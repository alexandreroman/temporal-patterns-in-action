#!/usr/bin/env bash
#
# cmux pre-destroy hook — stops everything this worktree spun up before
# /cmux:finish-feature or /cmux:abandon-feature wipes it.
#
# Two cleanup passes:
#   1. Containers — `compose down` for this worktree's project (compose
#      auto-namespaces by directory, so the main worktree's containers are
#      left alone).
#   2. Host processes — Go workers (`air`, `go run`, built binaries) and the
#      Nuxt frontend (`pnpm dev`, `node`) running on the host machine. Any
#      process anchored in the worktree (cwd or executable path) is stopped.
#
# Invoked by cmux with cwd = feature worktree.
#
# Env exported by cmux:
#   CMUX_FEATURE_SLUG     — short slug, e.g. "auth-jwt"
#   CMUX_FEATURE_BRANCH   — "feature/<slug>"
#   CMUX_FEATURE_WORKTREE — absolute path of the worktree
#   CMUX_MAIN_WORKTREE    — absolute path of the main worktree

set -euo pipefail

WORKTREE="${CMUX_FEATURE_WORKTREE:-$(pwd)}"

log() {
  printf '[pre-destroy] %s\n' "$*"
  if command -v cmux >/dev/null 2>&1 && [ -n "${CMUX_WORKSPACE_ID:-}" ]; then
    cmux log --level progress --source "pre-destroy" -- "$*" \
      >/dev/null 2>&1 || true
  fi
}

##
## 1. Stop containers tied to this worktree's compose project.
##

detect_compose() {
  if command -v docker >/dev/null 2>&1 \
       && docker compose version >/dev/null 2>&1; then
    echo "docker compose"
  elif command -v docker-compose >/dev/null 2>&1; then
    echo "docker-compose"
  elif command -v podman >/dev/null 2>&1 \
         && podman compose version >/dev/null 2>&1; then
    echo "podman compose"
  elif command -v podman-compose >/dev/null 2>&1; then
    echo "podman-compose"
  else
    return 1
  fi
}

if [ -f "$WORKTREE/compose.yaml" ] || [ -f "$WORKTREE/compose.yml" ] \
     || [ -f "$WORKTREE/docker-compose.yaml" ] \
     || [ -f "$WORKTREE/docker-compose.yml" ]; then
  if compose=$(detect_compose); then
    log "Stopping containers via $compose down"
    # --remove-orphans cleans up anything compose remembers but the file no
    # longer declares. -v drops named volumes scoped to this project.
    (cd "$WORKTREE" && $compose down --remove-orphans -v) \
      || log "compose down exited non-zero (continuing)"
  else
    log "No docker/podman compose binary found — skipping container cleanup"
  fi
fi

##
## 2. Stop host processes anchored in this worktree.
##

# Build the chain of our own ancestor PIDs so we never kill ourselves or the
# shell that invoked us.
self_chain=""
p=$$
while [ -n "$p" ] && [ "$p" != "0" ] && [ "$p" != "1" ]; do
  self_chain="$self_chain $p"
  p=$(ps -o ppid= -p "$p" 2>/dev/null | tr -d ' \n' || echo "")
  [ -z "$p" ] && break
done

is_self() {
  case " $self_chain " in *" $1 "*) return 0 ;; esac
  return 1
}

# Known dev-tool basenames we are willing to kill when their cwd matches the
# worktree. The list is deliberately narrow so an interactive shell or editor
# that happens to be cd'd into the worktree is left alone.
is_dev_tool() {
  case "$1" in
    air|go|node|nuxi|pnpm|nitro|main|worker) return 0 ;;
  esac
  return 1
}

collect_candidates() {
  # Processes whose argv references the worktree path. Catches node/nuxt/pnpm
  # invoked with absolute paths from this worktree.
  if command -v pgrep >/dev/null 2>&1; then
    pgrep -f "${WORKTREE}/" 2>/dev/null || true
  fi

  # Processes whose cwd is inside the worktree. Catches `air` and `go run`
  # started with bare commands.
  if command -v lsof >/dev/null 2>&1; then
    lsof -a -d cwd -F pn 2>/dev/null \
      | awk -v wt="${WORKTREE}/" '
          /^p/ { pid = substr($0, 2); next }
          /^n/ {
            path = substr($0, 2)
            if (index(path "/", wt) == 1) print pid
          }'
  fi
}

victims=()
seen=" "
while IFS= read -r pid; do
  [ -z "$pid" ] && continue
  case "$seen" in *" $pid "*) continue ;; esac
  seen="$seen$pid "
  is_self "$pid" && continue

  cmd=$(ps -o command= -p "$pid" 2>/dev/null || true)
  [ -z "$cmd" ] && continue

  comm=$(ps -o comm= -p "$pid" 2>/dev/null || true)
  base="${comm##*/}"

  # Argv mentions the worktree path → kill regardless of basename (covers
  # binaries built into bin/ or .output/server/index.mjs).
  if printf '%s' "$cmd" | grep -qF "${WORKTREE}/"; then
    victims+=("$pid")
  elif is_dev_tool "$base"; then
    victims+=("$pid")
  fi
done < <(collect_candidates)

if [ "${#victims[@]}" -eq 0 ]; then
  log "No host processes to stop"
  log "Done"
  exit 0
fi

log "Stopping ${#victims[@]} host process(es):"
for pid in "${victims[@]}"; do
  log "  pid=$pid $(ps -o command= -p "$pid" 2>/dev/null || echo '<gone>')"
done

kill -TERM "${victims[@]}" 2>/dev/null || true

deadline=$(( $(date +%s) + 5 ))
while [ "$(date +%s)" -lt "$deadline" ]; do
  alive=0
  for pid in "${victims[@]}"; do
    if kill -0 "$pid" 2>/dev/null; then
      alive=1
      break
    fi
  done
  [ "$alive" -eq 0 ] && break
  sleep 0.5
done

for pid in "${victims[@]}"; do
  if kill -0 "$pid" 2>/dev/null; then
    log "  pid=$pid still alive — SIGKILL"
    kill -KILL "$pid" 2>/dev/null || true
  fi
done

log "Done"
