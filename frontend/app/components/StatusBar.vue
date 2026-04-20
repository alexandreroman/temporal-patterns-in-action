<script setup lang="ts">
import type { StatusTone } from "~/types/status-bar";

defineProps<{
  tone: StatusTone;
  message: string;
}>();
</script>

<template>
  <div
    class="flex items-center gap-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-600 dark:border-slate-700 dark:bg-slate-800/60 dark:text-slate-300"
  >
    <Transition
      mode="out-in"
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0 translate-y-0.5"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100 translate-y-0"
      leave-to-class="opacity-0 -translate-y-0.5"
    >
      <div :key="`${tone}:${message}`" class="flex items-center gap-3">
        <svg
          v-if="tone === 'idle'"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="size-3.5 shrink-0 text-slate-400"
          aria-hidden="true"
        >
          <circle cx="12" cy="12" r="9" />
        </svg>
        <svg
          v-else-if="tone === 'running'"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="size-3.5 shrink-0 animate-spin text-blue-500"
          aria-hidden="true"
        >
          <path d="M21 12a9 9 0 1 1-6.3-8.6" />
        </svg>
        <svg
          v-else-if="tone === 'success'"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="size-3.5 shrink-0 text-emerald-500"
          aria-hidden="true"
        >
          <path d="M5 12l4 4L19 7" />
        </svg>
        <svg
          v-else
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="size-3.5 shrink-0 text-rose-500"
          aria-hidden="true"
        >
          <circle cx="12" cy="12" r="9" />
          <path d="M12 8v4" />
          <path d="M12 16h.01" />
        </svg>
        <span>{{ message }}</span>
      </div>
    </Transition>
  </div>
</template>
