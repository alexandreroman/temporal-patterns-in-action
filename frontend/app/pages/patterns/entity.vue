<script setup lang="ts">
import { computed, ref } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type {
  EntityAddItemPayload,
  EntityQueryResponse,
  EntitySignalPayload,
  EntitySignalRequest,
  EntitySignalResponse,
  EntityStartRequest,
  EntityStartResponse,
} from "~~/shared/types";

useSeoMeta({ title: "Entity Workflow" });

const STEP_DELAY_MS = 1200;

const {
  workflowId,
  events: streamEvents,
  running,
  error: finalError,
  run,
} = usePatternRun("entity");

// Local UI events spliced into the live stream so the code viewer can
// highlight the getCart range without a real NATS event flowing through.
const localEvents = ref<EventEnvelope[]>([]);

// Merge and sort by time so synthetic query events appear in the right slot.
const events = computed<EventEnvelope[]>(() => {
  const all = [...streamEvents.value, ...localEvents.value];
  all.sort((a, b) => a.time.localeCompare(b.time));
  return all;
});

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function sendSignal(signal: EntitySignalPayload): Promise<void> {
  const id = workflowId.value;
  if (!id) return;
  await $fetch<EntitySignalResponse>("/api/entity/signal", {
    method: "POST",
    body: { workflowId: id, signal } satisfies EntitySignalRequest,
  });
}

async function queryCart(): Promise<void> {
  const id = workflowId.value;
  if (!id) return;
  await $fetch<EntityQueryResponse>("/api/entity/query", {
    query: { workflowId: id },
  });
  // Synthetic envelope so EntityCodeViewer can highlight the getCart range —
  // not a real NATS event, purely a UI affordance.
  localEvents.value = [
    ...localEvents.value,
    {
      specversion: "1.0",
      id: `ui-query-${Date.now()}`,
      source: "ui",
      type: "entity.query.getCart",
      workflowId: id,
      runId: "",
      time: new Date().toISOString(),
      data: {},
    },
  ];
}

// Scripted shopping session: start the cart, then drive it with signals and
// queries until checkout.
async function playScenario(cartId: string): Promise<void> {
  await $fetch<EntityStartResponse>("/api/entity/start", {
    method: "POST",
    body: { cartId } satisfies EntityStartRequest,
  });

  const addItem = async (payload: EntityAddItemPayload) => {
    await sleep(STEP_DELAY_MS);
    await sendSignal({ type: "addItem", payload });
  };

  await addItem({ itemId: "p1", name: "Wireless headphones", priceCents: 7999, qty: 1 });
  await addItem({ itemId: "p2", name: "USB-C hub", priceCents: 3499, qty: 1 });
  await addItem({ itemId: "p3", name: "Mechanical keyboard", priceCents: 14999, qty: 1 });

  await sleep(STEP_DELAY_MS);
  await queryCart();

  await sleep(STEP_DELAY_MS / 2);
  await sendSignal({ type: "updateQty", payload: { itemId: "p1", qty: 3 } });

  await addItem({ itemId: "p4", name: "Monitor stand", priceCents: 4499, qty: 1 });
  await addItem({ itemId: "p5", name: "Webcam HD", priceCents: 5999, qty: 1 });
  await addItem({ itemId: "p6", name: "Laptop sleeve", priceCents: 2499, qty: 1 });

  await sleep(STEP_DELAY_MS / 2);
  await sendSignal({ type: "removeItem", payload: { itemId: "p2" } });

  await addItem({ itemId: "p7", name: "Portable SSD 1TB", priceCents: 8999, qty: 1 });
  await addItem({ itemId: "p8", name: "Studio microphone", priceCents: 11999, qty: 1 });
  await addItem({ itemId: "p9", name: "Desk mat XL", priceCents: 1999, qty: 2 });

  await sleep(STEP_DELAY_MS / 2);
  await sendSignal({ type: "updateQty", payload: { itemId: "p3", qty: 2 } });

  await addItem({ itemId: "p10", name: "Ergonomic mouse", priceCents: 6499, qty: 1 });
  await addItem({ itemId: "p11", name: "Adjustable lamp", priceCents: 9499, qty: 1 });

  await sleep(STEP_DELAY_MS / 2);
  await sendSignal({ type: "removeItem", payload: { itemId: "p9" } });

  await sleep(STEP_DELAY_MS);
  await queryCart();

  await sleep(STEP_DELAY_MS);
  await sendSignal({ type: "checkout", payload: {} });
}

async function runScenario(): Promise<void> {
  localEvents.value = [];
  const cartId = randomSuffix();
  // The whole scripted session runs as the start step, so the page stays
  // "starting" (and the button disabled) until checkout has been sent.
  await run(`entity-${cartId}`, () => playScenario(cartId));
}
</script>

<template>
  <section>
    <PatternHeader
      title="Entity Workflow — Shopping Cart"
      label="Run scenario"
      :disabled="running"
      @run="runScenario"
    >
      <template #icon><IconEntity class="h-5 w-5" /></template>
    </PatternHeader>

    <!-- Architecture diagram -->
    <div class="mt-2">
      <EntityArchitecture :events="events" />
    </div>

    <!-- Cart + state panel -->
    <div class="mt-4 flex flex-col gap-3 lg:flex-row lg:items-stretch">
      <div class="min-w-0 flex-1">
        <EntityCart :events="events" :workflow-id="workflowId" />
      </div>
      <EntityStatePanel :events="events" />
    </div>

    <!-- Status bar -->
    <EntityStatusBar :events="events" class="mt-6" />

    <!-- Code + event stream -->
    <div class="mt-4 flex flex-col gap-3 lg:flex-row">
      <div class="min-w-0 lg:w-[560px] lg:shrink-0">
        <EntityCodeViewer :events="events" />
      </div>
      <div class="min-w-0 flex-1">
        <EntityEventStream :events="events" />
      </div>
    </div>

    <p v-if="finalError" class="mt-4 text-sm text-rose-400">
      {{ finalError }}
    </p>
  </section>
</template>
