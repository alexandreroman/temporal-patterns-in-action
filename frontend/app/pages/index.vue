<script setup lang="ts">
useSeoMeta({ title: "Patterns" });

const NuxtLink = resolveComponent("NuxtLink");

type PatternIcon = "saga" | "batch" | "encryption";

const patterns: {
  slug: string;
  title: string;
  description: string;
  status: "available" | "coming-soon";
  icon: PatternIcon;
}[] = [
  {
    slug: "saga",
    title: "Saga",
    description:
      "Order processing saga — reserve inventory, charge payment, ship, notify. Roll back on failure.",
    status: "available",
    icon: "saga",
  },
  {
    slug: "batch",
    title: "Long-Running Batch",
    description:
      "Worker-throttled fan-out over a large image batch — retries, heartbeats, and a bounded backlog.",
    status: "available",
    icon: "batch",
  },
  {
    slug: "encryption",
    title: "Payload Encryption",
    description:
      "Symmetric PayloadCodec — AES-256-GCM encryption so Temporal stores only ciphertext end-to-end.",
    status: "available",
    icon: "encryption",
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
          <div class="flex items-start justify-between gap-3">
            <div class="flex items-center gap-3">
              <span
                class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-800 bg-slate-950 text-slate-300"
              >
                <IconSaga v-if="pattern.icon === 'saga'" class="h-5 w-5" />
                <IconBatch v-else-if="pattern.icon === 'batch'" class="h-5 w-5" />
                <IconEncryption v-else class="h-5 w-5" />
              </span>
              <h2 class="text-lg font-medium text-slate-100">{{ pattern.title }}</h2>
            </div>
            <span
              v-if="pattern.status === 'coming-soon'"
              class="shrink-0 rounded-full bg-slate-800 px-2 py-0.5 text-xs text-slate-400"
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
