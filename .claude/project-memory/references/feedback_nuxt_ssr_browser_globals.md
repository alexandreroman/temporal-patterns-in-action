---
name: "Nuxt SSR: guard browser globals with onMounted or import.meta.client"
description: "Top-level <script setup> code runs during SSR; browser-only globals must sit inside onMounted or behind import.meta.client, and watch immediate:true is NOT a substitute."
type: feedback
---

# Nuxt SSR: guard browser globals with onMounted or import.meta.client

Any access to browser-only globals
(`requestAnimationFrame`, `cancelAnimationFrame`,
`window`, `document`, `navigator`,
`ResizeObserver`, `EventSource`, `localStorage`,
…) inside a Nuxt component must live inside
`onMounted` (paired with `onBeforeUnmount` for
teardown) or behind `if (import.meta.client)`.
Top-level code in `<script setup>` runs during
SSR too. `watch(..., { immediate: true })` is
**not** an SSR guard — its callback fires
synchronously at watcher creation, including
during prerender, so it does NOT replace an
`onMounted` wrapper.

**Why:** a top-level
`watch(() => props.running, …, { immediate: true })`
driving a `requestAnimationFrame` loop fires
during SSR, hits `cancelAnimationFrame` (undefined
in Node), and crashes the page with HTTP 500.
`eslint` and `vue-tsc` both pass because
`cancelAnimationFrame` is declared by `lib.dom` —
the static checks don't catch SSR-only failures.

**How to apply:**

1. When writing or reviewing Vue/Nuxt code,
   keep a top-level guard around any browser
   global. Prefer `onMounted` for setup and
   `onBeforeUnmount` for teardown; use
   `if (import.meta.client)` only when the
   code can't be moved into a hook.
2. After any change to a frontend page or
   component, smoke the dev server with
   `curl -o /dev/null -w "%{http_code}\n" http://localhost:3000/<route>`
   in addition to `lint` and `vue-tsc`. It
   costs one tool call and catches SSR
   crashes that the static checks miss. This
   complements the existing CLAUDE.md rule
   to verify UI changes in a browser.
3. When dispatching a frontend edit to
   code-writer, never tell it to remove an
   `onMounted` / `import.meta.client` guard
   without specifying the equivalent
   replacement guard. State the SSR
   constraint explicitly in the brief.
