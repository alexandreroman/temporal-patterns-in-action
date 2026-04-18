<script setup lang="ts">
useHead({
  titleTemplate: (title) => (title ? `${title} — Temporal Patterns` : "Temporal Patterns in Action"),
});
</script>

<template>
  <div class="dark min-h-screen bg-slate-950 text-slate-100">
    <header class="border-b border-slate-800 bg-slate-900/80 backdrop-blur">
      <div class="mx-auto flex max-w-5xl items-center justify-between px-6 py-4">
        <NuxtLink to="/" class="text-lg font-semibold tracking-tight text-slate-100">
          Temporal Patterns in Action
        </NuxtLink>
        <a
          href="https://docs.temporal.io"
          target="_blank"
          rel="noopener"
          class="text-sm text-slate-400 hover:text-slate-100"
        >
          docs.temporal.io
        </a>
      </div>
    </header>

    <main class="mx-auto max-w-5xl px-6 py-10">
      <NuxtPage />
    </main>

    <footer class="mx-auto max-w-5xl px-6 py-10 text-sm text-slate-500">
      Built with Nuxt + Temporal.
      <NuxtLink to="http://localhost:8233" external class="underline hover:text-slate-300">
        Open Temporal UI
      </NuxtLink>
    </footer>
  </div>
</template>
