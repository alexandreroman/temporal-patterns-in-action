---
name: "Coding conventions"
description: "Line length limits, markdown style, and the LTS/stable dependency rule."
type: feedback
---

# Coding conventions

- **Line length:** Markdown/text ≤80 columns,
  code ≤120 columns.
- **Markdown:** blank line before and after
  headings and lists; fenced code blocks must
  carry a language tag.
- **Dependency versions:** always pick the
  latest LTS or stable release. Verify against
  official docs or via context7 before choosing
  a version.

**Why:** These are load-bearing preferences the
user holds, not optional style suggestions.

**How to apply:** Apply to every file written in
this repo.
