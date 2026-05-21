---
name: "Codec Server is opt-in by design — do not advertise via --ui-codec-endpoint"
description: "The encryption demo intentionally leaves the codec endpoint unconfigured server-side so users see ciphertext first, then opt into decoding."
type: project
---

# Codec Server is opt-in by design — do not advertise via --ui-codec-endpoint

The `codec-server` service runs in `compose.yaml`,
but the `temporal` service does **NOT** set
`--ui-codec-endpoint`. This is deliberate.

The point of the encryption pattern is to show
that Temporal stores ciphertext: when a user
opens an encrypted workflow in the Web UI, the
payloads must first appear as opaque base64. Only
once the user enables the codec server manually
in the UI (`Settings → Codec Server → Use my
browser setting → http://localhost:8888`) does
the clear payload appear — that's the "aha"
moment of the demo.

Setting `--ui-codec-endpoint http://localhost:8888`
on the `temporal` service would make decoding
automatic and silent. It would *look* like
Temporal isn't encrypting anything, defeating the
demo.

**Why:** Pedagogical. The visible toggle between
ciphertext and plaintext is the demo's payoff,
not a configuration chore. See also
[[feedback_demo_priorities]] — bias toward
visibility over production ergonomics.

**How to apply:**

- Do NOT add `--ui-codec-endpoint` to the
  `temporal` service in `compose.yaml`, even if
  a future contributor argues it's more
  convenient.
- If someone reports "payloads stay encrypted in
  the UI", that's *expected* until they enable
  the codec endpoint in their browser. Document
  the toggle in README rather than papering over
  it server-side.
- The codec server's CORS `UI_ORIGIN` env still
  needs to match the Web UI origin
  (`http://localhost:8233`); that's the only
  server-side wiring required.
