---
name: "Verify Temporal dynamic config keys with a wrong-type probe"
description: "temporal server start-dev silently ignores unknown --dynamic-config-value keys; prove a key is live by feeding it a bad type at debug log level"
type: project
---

# Verify Temporal dynamic config keys with a wrong-type probe

`temporal server start-dev` accepts **any**
`--dynamic-config-value KEY=VALUE` without complaint. An
unknown, renamed, or removed key produces no warning, no
error, and a zero exit code — the server starts perfectly
and the setting is simply never applied.

**Why:** the Priority and Fairness pattern depends on three
keys passed in `compose.yaml`
(`matching.useNewMatcher`, `matching.enableFairness`,
`matching.enableMigration`). If a server upgrade renamed one,
the demo would lose its behaviour silently. "It boots clean"
is therefore worthless evidence on its own.

**How to apply:** on every Temporal server image bump, run the
wrong-type probe. Give the key a value of the wrong JSON type
and raise the log level, then drive one workflow so the
matching service actually reads the setting (reads are lazy):

```sh
podman run -d --name probe temporalio/temporal:<tag> \
  server start-dev --ip 0.0.0.0 --log-level debug \
  --dynamic-config-value 'matching.enableFairness="not-a-bool"'
podman exec probe temporal workflow start \
  --type ProbeWF --task-queue probe-tq --workflow-id p1
podman logs probe 2>&1 | grep 'Failed to convert value'
```

A **registered** key logs
`level=WARN msg="Failed to convert value, using default" key=matching.enablefairness ...`
(the key is lowercased in the log). An **unregistered** key
logs nothing at all. Always include a deliberately bogus key
such as `matching.thisKeyDoesNotExistAtAll` as a control, so a
silent result cannot be mistaken for a pass.

Two details that make the probe fail if skipped:

- `--log-level debug` is required. The documented default is
  `warn` for `start-dev`, but the container prints only the
  startup banner, so the warning never surfaces otherwise.
- Traffic is required. Without a workflow the setting is never
  read and even a live key stays silent.

A quick corroborating check is
`grep -aoE 'matching\.[A-Za-z]*([Ff]air|[Mm]atcher|[Mm]igration|[Pp]riority)[A-Za-z]*' /usr/local/bin/temporal | sort -u`
inside the image, diffed between the old and new tags to spot
renames in the key surface.
