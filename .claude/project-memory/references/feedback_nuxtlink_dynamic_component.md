---
name: "Dynamic NuxtLink via <component :is>"
description: "Pass a resolved component reference, not the string 'NuxtLink', when rendering NuxtLink through <component :is> in Nuxt 4 / Vue 3."
type: feedback
---

# Dynamic NuxtLink via <component :is>

When conditionally rendering a `NuxtLink` via
`<component :is>`, pass a component reference
obtained with `resolveComponent("NuxtLink")` in
`<script setup>` — never the string
`'NuxtLink'`.

```vue
<script setup lang="ts">
const NuxtLink = resolveComponent("NuxtLink");
</script>

<template>
  <component :is="condition ? NuxtLink : 'div'" :to="...">
    ...
  </component>
</template>
```

**Why:** In Nuxt 4, `NuxtLink` is auto-imported
for direct template use (`<NuxtLink>`), but it
is not registered as a global component under
that string name. Passing the string `'NuxtLink'`
to `<component :is>` makes Vue treat it as an
unknown component and fall back to an inert
element — the card renders but is not a real
hyperlink. Observed on
`frontend/app/pages/index.vue` where the Saga
card lost its link behavior until switched to
`resolveComponent`.

**How to apply:** Any time a Vue template in
this project uses `<component :is>` to pick
between `NuxtLink` and another tag (or another
component), resolve the component in the script
block and reference the variable. Applies to
any Nuxt page or component using conditional
rendering between a link and a non-link element.
