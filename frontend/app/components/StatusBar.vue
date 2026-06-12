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
    <!--
      :key recreates the span on each new message so the CSS animation replays.
      A CSS animation (unlike a Vue <Transition>) never gates DOM mounting, so
      the text is always current even on a hidden tab; it just skips the visual
      animation while the engine is frozen.
    -->
    <span :key="message" class="status-message">{{ message }}</span>
  </div>
</template>

<style scoped>
.status-message {
  animation: status-message-in 180ms ease-out;
}
@keyframes status-message-in {
  from {
    opacity: 0;
    transform: translateY(2px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
@media (prefers-reduced-motion: reduce) {
  .status-message {
    animation: none;
  }
}
</style>
