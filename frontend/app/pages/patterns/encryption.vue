<script setup lang="ts">
import { reactive, ref, watch } from "vue";
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

const {
  workflowId,
  events,
  starting,
  running,
  error: finalError,
  run,
} = usePatternRun("encryption");

const clientPayload = ref<SensitiveOrder | null>(null);
const storedPayload = ref<EncryptionStartResponse["storedPayload"] | null>(null);

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

async function start() {
  clientPayload.value = null;
  storedPayload.value = null;

  const orderId = `order-${randomSuffix()}`;
  await run(`encryption-${orderId}`, async () => {
    const res = await $fetch<EncryptionStartResponse>("/api/encryption/start", {
      method: "POST",
      body: {
        orderId,
        scenario: form.scenario,
      },
    });
    clientPayload.value = res.clientPayload;
    storedPayload.value = res.storedPayload;
  });
}
</script>

<template>
  <section>
    <PatternHeader
      title="Payload Encryption — Symmetric PayloadCodec"
      label="Run workflow"
      :busy-label="starting ? 'Starting…' : 'Running…'"
      :disabled="running"
      @run="start"
    >
      <template #icon><IconEncryption class="h-5 w-5" /></template>
      <select
        v-model="form.scenario"
        :disabled="running"
        class="rounded-md border border-slate-700 bg-slate-800 px-2 py-1 text-xs text-slate-200 disabled:opacity-50"
      >
        <option value="clear">Clear (no codec)</option>
        <option value="encrypted">Encrypted (AES-256-GCM)</option>
      </select>
    </PatternHeader>

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
        <EncryptionCodeViewer :events="events" :scenario="form.scenario" />
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
