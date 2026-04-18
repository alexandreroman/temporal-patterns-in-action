<script setup lang="ts">
import type { StatusTone } from "~/types/status-bar";

defineProps<{
  tone: StatusTone;
  message: string;
}>();

const dotClass: Record<StatusTone, string> = {
  idle: "bg-slate-400",
  running: "bg-blue-500 animate-pulse",
  success: "bg-emerald-500",
  error: "bg-rose-500",
};
</script>

<template>
  <div
    class="flex items-center gap-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-600 dark:border-slate-700 dark:bg-slate-800/60 dark:text-slate-300"
  >
    <span class="size-2 shrink-0 rounded-full" :class="dotClass[tone]" />
    <span>{{ message }}</span>
  </div>
</template>
