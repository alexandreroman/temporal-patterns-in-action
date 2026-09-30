---
name: "Codec Server is opt-in by design"
description: "The temporal service never sets --ui-codec-endpoint; users enable the codec per browser via the glasses icon, so ciphertext shows first."
type: project
---

# Codec Server is opt-in by design

The `temporal` service in `compose.yaml` does not set
`--ui-codec-endpoint`. An encrypted workflow shows opaque base64 in
the Web UI until the user clicks the **glasses icon** in the top
bar, picks "Use my browser setting and ignore Cluster-level
setting" and enters the codec address. The Codec Server section of
the global Settings panel (bottom-left avatar) is not the entry
point that docs, screenshots or UI automation use.

**Why:** the visible ciphertext-to-plaintext toggle is the demo's
payoff; a server-side endpoint would decode silently and make
Temporal look unencrypted. See [[feedback_demo_priorities]].

**How to apply:** keep `--ui-codec-endpoint` off the `temporal`
service; "payloads stay encrypted" is expected until the toggle is
set. The only server-side wiring is the codec server's `UI_ORIGIN`
CORS list, which must match the Web UI origin.
