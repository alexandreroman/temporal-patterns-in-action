<script setup lang="ts">
useSeoMeta({ title: "Patterns" });

type PatternIcon = "saga" | "batch" | "encryption" | "agent";

const patterns: {
  slug: string;
  title: string;
  description: string;
  icon: PatternIcon;
}[] = [
  {
    slug: "saga",
    title: "Saga",
    description:
      "Order processing saga — reserve inventory, charge payment, ship, notify. Roll back on failure.",
    icon: "saga",
  },
  {
    slug: "batch",
    title: "Long-Running Batch",
    description:
      "Worker-throttled fan-out over a large image batch — retries, heartbeats, and a bounded backlog.",
    icon: "batch",
  },
  {
    slug: "encryption",
    title: "Payload Encryption",
    description:
      "Symmetric PayloadCodec — AES-256-GCM encryption so Temporal stores only ciphertext end-to-end.",
    icon: "encryption",
  },
  {
    slug: "agent",
    title: "Durable AI Agent",
    description:
      "Travel-planner agent loop — LLM reasoning, MCP tool calls, and a signal-gated human approval.",
    icon: "agent",
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
        <NuxtLink
          :to="`/patterns/${pattern.slug}`"
          class="block rounded-xl border border-slate-800 bg-slate-900 p-5 transition hover:border-slate-600 hover:bg-slate-800/70"
        >
          <div class="flex items-center gap-3">
            <span
              class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-800 bg-slate-950 text-slate-300"
            >
              <IconSaga v-if="pattern.icon === 'saga'" class="h-5 w-5" />
              <IconBatch v-else-if="pattern.icon === 'batch'" class="h-5 w-5" />
              <IconEncryption v-else-if="pattern.icon === 'encryption'" class="h-5 w-5" />
              <IconAgent v-else class="h-5 w-5" />
            </span>
            <h2 class="text-lg font-medium text-slate-100">{{ pattern.title }}</h2>
          </div>
          <p class="mt-2 text-sm text-slate-400">{{ pattern.description }}</p>
        </NuxtLink>
      </li>
    </ul>
  </section>
</template>
