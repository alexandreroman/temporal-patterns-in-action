<script setup lang="ts">
/**
 * Top of every pattern page: back link, icon badge, title, and the control
 * bar ending with the Run button. Pattern-specific controls (scenario select,
 * …) go in the default slot, before the button.
 */

withDefaults(
  defineProps<{
    title: string;
    /** Button text while the run can be started. */
    label: string;
    /** Button text while `disabled` is true. */
    busyLabel?: string;
    disabled?: boolean;
  }>(),
  { busyLabel: "Running…", disabled: false },
);

const emit = defineEmits<{ run: [] }>();
</script>

<template>
  <header>
    <NuxtLink to="/" class="text-sm text-slate-400 hover:text-slate-100"> &larr; back </NuxtLink>

    <div class="mt-2 flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <span
          class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-800 bg-slate-950 text-slate-300"
        >
          <slot name="icon" />
        </span>
        <h1 class="text-2xl font-semibold tracking-tight text-slate-100">{{ title }}</h1>
      </div>
      <div class="flex items-center gap-2">
        <slot />
        <button
          type="button"
          :disabled="disabled"
          class="cursor-pointer rounded-md bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-50"
          @click="emit('run')"
        >
          {{ disabled ? busyLabel : label }}
        </button>
      </div>
    </div>
  </header>
</template>
