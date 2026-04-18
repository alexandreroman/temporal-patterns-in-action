---
name: "Rogue host workers compete with containers"
description: "When timing/behavior of an activity looks stale after rebuild, check for a host-side `go run` worker still polling the same task queue — it competes with the containerised worker."
type: feedback
---

# Rogue host workers compete with containers

If an activity's behavior looks stale after
rebuilding the worker container (e.g., an
unexpectedly fast return, an old log message,
an old error shape), a host-side worker from a
prior `go run ./workers/<pattern>/cmd/worker`
may still be polling the same task queue and
stealing tasks.

**Symptoms:**
- Activity completes in milliseconds when the
  new code sleeps for seconds.
- The new activity body's log line (e.g.
  `"Shipping order"`) never appears in the
  container logs, even though Temporal shows
  the activity as Completed.
- `temporal task-queue describe --task-queue
  <name>` lists a Poller identity of the form
  `<pid>@<hostname>@` that is *not* one of the
  running containers (e.g. `34795@donnager@`).

**Diagnose:**

```bash
podman exec temporal-patterns-in-action-temporal-1 \
  temporal task-queue describe --task-queue patterns-<pattern>
ps aux | grep -E "(go-build|workers/<pattern>)"
kill <pid>
```

**Why:** Encountered 2026-04-18 during saga
timing work: a stale `go run` worker from a
previous session was returning ship/send
activities in ~3 ms while the new container
sleeps 2.5 s. Because Temporal's server
dispatches each activity task to whichever
poller picks it up first, two competing
workers produce inconsistent behavior that is
easy to misread as a build-cache problem.

**How to apply:** When timing or behavior
doesn't match freshly-built code, before
blaming Docker caching (`--no-cache` etc.),
run `task-queue describe` and look for extra
pollers. Kill any leftover host-side worker
PIDs before re-running the check.
