<script setup lang="ts">
useHead({
  titleTemplate: (title) =>
    title ? `${title} — Temporal Patterns` : "Temporal Patterns in Action",
});
</script>

<template>
  <div class="dark min-h-screen bg-slate-950 text-slate-100">
    <header class="border-b border-slate-800 bg-slate-900/80 backdrop-blur">
      <div class="mx-auto flex max-w-5xl items-center px-6 py-4">
        <NuxtLink to="/" class="flex items-center gap-3">
          <img :src="'/images/temporal-logo.png'" alt="Temporal" class="h-10 w-auto" />
          <span class="-translate-x-[8px] translate-y-[1px] text-base leading-none tracking-tight text-[#f2f2f2]">patterns in action</span>
        </NuxtLink>
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
