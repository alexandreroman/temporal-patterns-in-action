<script setup lang="ts">
useSeoMeta({ title: "Patterns" });

const NuxtLink = resolveComponent("NuxtLink");

const patterns = [
  {
    slug: "saga",
    title: "Saga",
    description:
      "Order processing saga — reserve inventory, charge payment, ship, notify. Roll back on failure.",
    status: "available" as const,
  },
  {
    slug: "batch",
    title: "Long-Running Batch",
    description:
      "Parallel sliding window over a large image batch — retries, heartbeats, and a bounded backlog.",
    status: "available" as const,
  },
  {
    slug: "signals",
    title: "Signals & Queries",
    description: "Interact with a running workflow — approve, cancel, inspect progress.",
    status: "coming-soon" as const,
  },
];
</script>

<template>
  <section>
    <h1 class="text-3xl font-semibold tracking-tight text-slate-100">
      Temporal patterns, runnable.
    </h1>
    <p class="mt-3 max-w-2xl text-slate-400">
      A set of executable demos for core Temporal patterns. Pick a pattern, trigger it from the UI,
      watch it run — then inspect its history in the Temporal Web UI.
    </p>

    <ul class="mt-10 grid gap-4 sm:grid-cols-2">
      <li v-for="pattern in patterns" :key="pattern.slug">
        <component
          :is="pattern.status === 'available' ? NuxtLink : 'div'"
          :to="pattern.status === 'available' ? `/patterns/${pattern.slug}` : undefined"
          class="block rounded-xl border border-slate-800 bg-slate-900 p-5 transition"
          :class="
            pattern.status === 'available'
              ? 'hover:border-slate-600 hover:bg-slate-800/70'
              : 'opacity-60'
          "
        >
          <div class="flex items-center justify-between">
            <h2 class="text-lg font-medium text-slate-100">{{ pattern.title }}</h2>
            <span
              v-if="pattern.status === 'coming-soon'"
              class="rounded-full bg-slate-800 px-2 py-0.5 text-xs text-slate-400"
            >
              coming soon
            </span>
          </div>
          <p class="mt-2 text-sm text-slate-400">{{ pattern.description }}</p>
        </component>
      </li>
    </ul>
  </section>
</template>
