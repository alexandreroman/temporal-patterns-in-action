<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import type {
  EncryptionStartRequest,
  EncryptionStartResponse,
  SensitiveOrder,
} from "~~/shared/types";

useSeoMeta({ title: "Payload Encryption" });

type Scenario = EncryptionStartRequest["scenario"];

const form = reactive({
  scenario: "clear" as Scenario,
});

const workflowId = ref<string | null>(null);
const starting = ref(false);
const finalError = ref<string | null>(null);

const clientPayload = ref<SensitiveOrder | null>(null);
const storedPayload = ref<EncryptionStartResponse["storedPayload"] | null>(null);

const { events, waitForOpen } = usePatternStream("encryption", workflowId);

// Switching scenario swaps the codec on the wire, so any existing run's UI
// state is stale — reset to the empty slate so the user has to trigger a new
// workflow to observe the new encoding.
watch(
  () => form.scenario,
  () => {
    workflowId.value = null;
    clientPayload.value = null;
    storedPayload.value = null;
    finalError.value = null;
  },
);

const TERMINAL_EVENTS = new Set(["progress.workflow.completed", "progress.workflow.failed"]);

const running = computed(() => {
  if (starting.value) return true;
  if (!workflowId.value) return false;
  return !events.value.some((e) => TERMINAL_EVENTS.has(e.type));
});

function randomSuffix(): string {
  // 6-char base36 is plenty for a per-run order ID in a demo.
  return Math.random().toString(36).slice(2, 8);
}

async function start() {
  finalError.value = null;
  starting.value = true;
  clientPayload.value = null;
  storedPayload.value = null;

  const orderId = `order-${randomSuffix()}`;
  // Subscribe BEFORE starting the workflow: core NATS has no replay, and the
  // first progress.step.started fires almost immediately after start — we'd
  // miss it if the SSE stream opened only after the start response came back.
  workflowId.value = `encryption-${orderId}`;
  try {
    await waitForOpen();
    const res = await $fetch<EncryptionStartResponse>("/api/encryption/start", {
      method: "POST",
      body: {
        orderId,
        scenario: form.scenario,
      },
    });
    clientPayload.value = res.clientPayload;
    storedPayload.value = res.storedPayload;
  } catch (error) {
    finalError.value = error instanceof Error ? error.message : String(error);
    workflowId.value = null;
  } finally {
    starting.value = false;
  }
}
</script>

<template>
  <section>
    <NuxtLink to="/" class="text-sm text-slate-400 hover:text-slate-100"> &larr; back </NuxtLink>

    <!-- Control bar -->
    <div class="mt-2 flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <span
          class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-800 bg-slate-950 text-slate-300"
        >
          <IconEncryption class="h-5 w-5" />
        </span>
        <h1 class="text-2xl font-semibold tracking-tight text-slate-100">
          Payload Encryption &mdash; Symmetric PayloadCodec
        </h1>
      </div>
      <div class="flex items-center gap-2">
        <select
          v-model="form.scenario"
          :disabled="running"
          class="rounded-md border border-slate-700 bg-slate-800 px-2 py-1 text-xs text-slate-200 disabled:opacity-50"
        >
          <option value="encrypted">Encrypted (AES-256-GCM)</option>
          <option value="clear">Clear (no codec)</option>
        </select>
        <button
          type="button"
          :disabled="running"
          class="cursor-pointer rounded-md bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-50"
          @click="start"
        >
          {{ starting ? "Starting…" : running ? "Running…" : "Run workflow" }}
        </button>
      </div>
    </div>

    <!-- Architecture diagram -->
    <div class="mt-2">
      <EncryptionArchitecture :events="events" :scenario="form.scenario" />
    </div>

    <!-- Payload flow: client → Temporal → worker -->
    <div class="mt-2">
      <EncryptionPayloadFlow
        :scenario="form.scenario"
        :client-payload="clientPayload"
        :stored-payload="storedPayload"
        :events="events"
      />
    </div>

    <!-- Status bar -->
    <EncryptionStatusBar :events="events" :scenario="form.scenario" class="mt-6" />

    <!-- Code + event stream -->
    <div class="mt-4 flex flex-col gap-3 lg:flex-row">
      <div class="min-w-0 lg:w-[560px] lg:shrink-0">
        <EncryptionCodeViewer :events="events" />
      </div>
      <div class="min-w-0 flex-1">
        <EncryptionEventStream :events="events" />
      </div>
    </div>

    <p v-if="finalError" class="mt-4 text-sm text-rose-400">
      {{ finalError }}
    </p>
  </section>
</template>
