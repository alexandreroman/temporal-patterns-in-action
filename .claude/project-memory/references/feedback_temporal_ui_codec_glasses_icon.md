---
name: "Temporal UI Codec Server: glasses icon, not the user/settings menu"
description: "The per-workflow Codec Server modal is opened by the glasses icon in the top bar, not by the user avatar or the global Settings panel."
type: feedback
---

# Temporal UI Codec Server: glasses icon, not the user/settings menu

When documenting or driving the Temporal Web UI's
codec server toggle, the entry point is the
**glasses icon in the top bar** (next to the user
avatar, top-right). Clicking it opens a *Codec
Server* modal scoped to the current workflow.

The global Settings panel — opened from the user
avatar in the bottom-left — also exposes a Codec
Server section, but it's a different modal and is
NOT what the user wants to point demo readers at.
The per-workflow glasses-icon modal is the
canonical demo entry point.

**Why:** The glasses icon is contextual — it sits
on each workflow view and matches the "decode
this workflow's payloads" mental model, unlike
the global Settings entry, which is not the
intended entry point for demo readers.

**How to apply:**

- README, blog posts, and screenshots about the
  encryption pattern must point to the glasses
  icon in the top bar.
- When automating the UI to enable the codec
  endpoint, click the glasses icon (it lives in
  the topbar buttons near the user avatar), not
  the bottom-left user menu.
- The modal contents — radios *"Use Cluster-level
  setting"* vs *"Use my browser setting and
  ignore Cluster-level setting"*, the *Codec
  Server browser endpoint* textarea, *Pass the
  user access token* and *Include cross-origin
  credentials* toggles, *Apply* button — are the
  same as the Settings-menu version. Only the
  entry point differs.
