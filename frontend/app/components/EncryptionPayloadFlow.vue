<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { SensitiveOrder } from "~~/shared/types";

const props = defineProps<{
  scenario: "clear" | "encrypted";
  clientPayload: SensitiveOrder | null;
  storedPayload: { encoding: string; dataBase64: string } | null;
  events: EventEnvelope[];
}>();

// Any step.started event is enough to flag that the worker has received —
// and therefore decoded — the workflow input at least once.
const workerHasDecoded = computed(() =>
  props.events.some((e) => e.type === "progress.step.started"),
);

const clientJson = computed(() =>
  props.clientPayload ? JSON.stringify(props.clientPayload, null, 2) : "",
);

// Break the base64 string into fixed-width lines so the ciphertext panel
// wraps nicely inside the card.
const storedBase64Wrapped = computed(() => {
  if (!props.storedPayload || props.scenario !== "encrypted") return "";
  const raw = props.storedPayload.dataBase64;
  const lineLen = 22;
  const lines: string[] = [];
  for (let i = 0; i < raw.length; i += lineLen) {
    lines.push(raw.slice(i, i + lineLen));
  }
  return lines.join("\n");
});

const isEncrypted = computed(() => props.scenario === "encrypted");
</script>

<template>
  <div class="grid gap-3 md:grid-cols-[1fr_auto_1fr_auto_1fr] md:items-stretch">
    <!-- 1. Client payload -->
    <section
      class="flex h-52 flex-col rounded-xl border border-slate-700 bg-slate-900 p-4 text-xs text-slate-200"
    >
      <header class="mb-2 flex items-center justify-between gap-2">
        <h3 class="text-sm font-semibold text-slate-100">Client payload</h3>
        <span class="rounded-full bg-slate-800 px-2 py-0.5 text-[10px] text-slate-300">
          plaintext
        </span>
      </header>
      <pre
        class="min-h-0 flex-1 overflow-auto whitespace-pre-wrap break-all font-mono text-[11px] text-slate-300"
        >{{ clientPayload ? clientJson : "(run the workflow)" }}</pre
      >
    </section>

    <!-- arrow 1→2 -->
    <div class="hidden items-center justify-center text-slate-500 md:flex" aria-hidden="true">
      <span class="text-lg">&rarr;</span>
    </div>

    <!-- 2. Temporal stored payload -->
    <section
      class="flex h-52 flex-col rounded-xl border p-4 text-xs text-slate-200"
      :class="
        !storedPayload
          ? 'border-slate-700 bg-slate-900'
          : isEncrypted
            ? 'border-emerald-600 bg-emerald-950/40'
            : 'border-rose-600 bg-rose-950/40'
      "
    >
      <header class="mb-2 flex items-center justify-between gap-2">
        <h3 class="text-sm font-semibold text-slate-100">Temporal stored</h3>
        <span
          class="rounded-full px-2 py-0.5 text-[10px]"
          :class="
            !storedPayload
              ? 'bg-slate-800 text-slate-300'
              : isEncrypted
                ? 'bg-emerald-800/60 text-emerald-200'
                : 'bg-rose-800/60 text-rose-200'
          "
        >
          {{ isEncrypted ? "ciphertext" : "cleartext" }}
        </span>
      </header>
      <Transition
        enter-active-class="transition-opacity duration-500"
        enter-from-class="opacity-0"
        enter-to-class="opacity-100"
      >
        <div
          v-if="storedPayload"
          class="mb-2 text-[10px] uppercase tracking-wide text-slate-400"
        >
          encoding: {{ storedPayload.encoding }}
        </div>
      </Transition>
      <pre
        v-if="storedPayload"
        class="min-h-0 flex-1 overflow-auto font-mono text-[11px]"
        :class="[
          isEncrypted ? 'whitespace-pre text-emerald-200' : 'whitespace-pre-wrap break-all text-rose-200',
        ]"
        >{{ isEncrypted ? storedBase64Wrapped : clientJson }}</pre
      >
      <div v-else class="min-h-0 flex-1 overflow-auto">
        <p class="font-mono text-[11px] text-slate-500">(run the workflow)</p>
      </div>
      <Transition
        enter-active-class="transition-opacity duration-500"
        enter-from-class="opacity-0"
        enter-to-class="opacity-100"
      >
        <p
          v-if="!isEncrypted && storedPayload"
          class="mt-2 inline-flex items-center gap-1.5 truncate text-[11px] leading-snug text-rose-300"
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            class="size-3 shrink-0"
            aria-hidden="true"
          >
            <path d="M12 9v4" />
            <path d="M12 17h.01" />
            <path
              d="M4.93 19h14.14a2 2 0 0 0 1.74-3l-7.07-12.24a2 2 0 0 0-3.48 0L3.19 16a2 2 0 0 0 1.74 3z"
            />
          </svg>
          <span class="truncate">Raw JSON — PII exposed.</span>
        </p>
      </Transition>
    </section>

    <!-- arrow 2→3 -->
    <div class="hidden items-center justify-center text-slate-500 md:flex" aria-hidden="true">
      <span class="text-lg">&rarr;</span>
    </div>

    <!-- 3. Worker payload -->
    <section
      class="flex h-52 flex-col rounded-xl border border-slate-700 bg-slate-900 p-4 text-xs text-slate-200"
    >
      <header class="mb-2 flex items-center justify-between gap-2">
        <h3 class="text-sm font-semibold text-slate-100">Worker payload</h3>
        <span class="rounded-full bg-slate-800 px-2 py-0.5 text-[10px] text-slate-300">
          decoded
        </span>
      </header>
      <pre
        v-if="clientPayload && workerHasDecoded"
        class="min-h-0 flex-1 overflow-auto whitespace-pre-wrap break-all font-mono text-[11px] text-slate-300"
        >{{ clientJson }}</pre
      >
      <div v-else class="min-h-0 flex-1 overflow-auto">
        <p class="font-mono text-[11px] text-slate-500">
          {{ clientPayload ? "(waiting for worker)" : "(run the workflow)" }}
        </p>
      </div>
      <Transition
        enter-active-class="transition-opacity duration-500"
        enter-from-class="opacity-0"
        enter-to-class="opacity-100"
      >
        <p
          v-if="clientPayload && workerHasDecoded"
          class="mt-2 inline-flex items-center gap-1.5 text-[11px] text-slate-400"
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            class="size-3 shrink-0"
            aria-hidden="true"
          >
            <path d="M7 11V7a5 5 0 0 1 9.9-1" />
            <rect x="5" y="11" width="14" height="10" rx="2" />
          </svg>
          <span>Plaintext on the worker after codec decode.</span>
        </p>
      </Transition>
    </section>
  </div>
</template>
