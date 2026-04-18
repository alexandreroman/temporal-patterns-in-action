<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { createHighlighter, type ThemedToken } from "shiki";
import type { EventEnvelope } from "~~/shared/events";
import type { CodeLang } from "~/composables/useCodeLang";

/**
 * Code viewer that highlights the relevant lines
 * of the saga workflow based on the live event stream.
 */

const props = defineProps<{
  events: EventEnvelope[];
}>();

type Lang = CodeLang;
const lang = useCodeLang();

interface CodeSource {
  label: string;
  lines: string[];
  stepLines: Record<string, [number, number]>;
  compLines: Record<string, [number, number]>;
}

const SOURCES: Record<Lang, CodeSource> = {
  go: {
    label: "Go",
    lines: [
      "func OrderProcessingWorkflow(ctx workflow.Context, order OrderInput) error {",
      "    var compensations []func(workflow.Context) error",
      "",
      "    // Step 1: Reserve inventory",
      "    err := workflow.ExecuteActivity(ctx,",
      "        a.ReserveInventory, order,",
      "    ).Get(ctx, &itemID)",
      "    if err != nil {",
      "        return runCompensations(ctx, compensations)",
      "    }",
      "    compensations = append(compensations,",
      "        func(c workflow.Context) error {",
      "            return workflow.ExecuteActivity(c,",
      "                a.ReleaseInventory, itemID,",
      "            ).Get(c, nil)",
      "        },",
      "    )",
      "",
      "    // Step 2: Charge payment",
      "    err = workflow.ExecuteActivity(ctx,",
      "        a.ChargePayment, order, itemID,",
      "    ).Get(ctx, &txnID)",
      "    if err != nil {",
      "        return runCompensations(ctx, compensations)",
      "    }",
      "    compensations = append(compensations,",
      "        func(c workflow.Context) error {",
      "            return workflow.ExecuteActivity(c,",
      "                a.RefundPayment, txnID, order.Amount,",
      "            ).Get(c, nil)",
      "        },",
      "    )",
      "",
      "    // Step 3: Ship order",
      "    err = workflow.ExecuteActivity(ctx,",
      "        a.ShipOrder, order,",
      "    ).Get(ctx, &trackingID)",
      "    if err != nil {",
      "        return runCompensations(ctx, compensations)",
      "    }",
      "    compensations = append(compensations,",
      "        func(c workflow.Context) error {",
      "            return workflow.ExecuteActivity(c,",
      "                a.CancelShipment, trackingID,",
      "            ).Get(c, nil)",
      "        },",
      "    )",
      "",
      "    // Step 4: Send confirmation",
      "    err = workflow.ExecuteActivity(ctx,",
      "        a.SendConfirmation, order,",
      "    ).Get(ctx, &email)",
      "    if err != nil {",
      "        return runCompensations(ctx, compensations)",
      "    }",
      "    compensations = append(compensations,",
      "        func(c workflow.Context) error {",
      "            return workflow.ExecuteActivity(c,",
      "                a.RetractEmail, email,",
      "            ).Get(c, nil)",
      "        },",
      "    )",
      "",
      '    result.Status = "completed"',
      "    return result, nil",
      "}",
    ],
    stepLines: {
      "reserve-inventory": [3, 16],
      "charge-payment": [18, 31],
      "ship-order": [33, 46],
      "send-confirmation": [48, 61],
    },
    compLines: {
      "release-inventory": [10, 16],
      "refund-payment": [25, 31],
      "cancel-shipment": [40, 46],
      "retract-email": [55, 61],
    },
  },
  java: {
    label: "Java",
    lines: [
      "@WorkflowMethod",
      "public void processSaga(Order order) {",
      "    Saga saga = new Saga(",
      "        new Saga.Options.Builder().build());",
      "    try {",
      "        // Step 1: Reserve inventory",
      "        activities.reserveInventory(order);",
      "        saga.addCompensation(",
      "            activities::releaseInventory, order);",
      "",
      "        // Step 2: Charge payment",
      "        activities.chargePayment(order);",
      "        saga.addCompensation(",
      "            activities::refundPayment, order);",
      "",
      "        // Step 3: Ship order",
      "        activities.shipOrder(order);",
      "        saga.addCompensation(",
      "            activities::cancelShipment, order);",
      "",
      "        // Step 4: Send confirmation",
      "        activities.sendConfirmation(order);",
      "        saga.addCompensation(",
      "            activities::retractEmail, order);",
      "",
      "    } catch (ActivityFailure e) {",
      "        saga.compensate();",
      "        throw e;",
      "    }",
      "}",
    ],
    stepLines: {
      "reserve-inventory": [5, 8],
      "charge-payment": [10, 13],
      "ship-order": [15, 18],
      "send-confirmation": [20, 23],
    },
    compLines: {
      "release-inventory": [7, 8],
      "refund-payment": [12, 13],
      "cancel-shipment": [17, 18],
      "retract-email": [22, 23],
    },
  },
  python: {
    label: "Python",
    lines: [
      "@workflow.defn",
      "class OrderProcessingWorkflow:",
      "    @workflow.run",
      "    async def run(self, order: Order) -> None:",
      "        compensations: list[Callable] = []",
      "",
      "        # Step 1: Reserve inventory",
      "        await workflow.execute_activity(",
      "            reserve_inventory, order,",
      "            start_to_close_timeout=timedelta(seconds=30))",
      "        compensations.append(lambda:",
      "            workflow.execute_activity(",
      "                release_inventory, item_id,",
      "                start_to_close_timeout=timedelta(seconds=30)))",
      "",
      "        # Step 2: Charge payment",
      "        await workflow.execute_activity(",
      "            charge_payment, order,",
      "            start_to_close_timeout=timedelta(seconds=30))",
      "        compensations.append(lambda:",
      "            workflow.execute_activity(",
      "                refund_payment, order,",
      "                start_to_close_timeout=timedelta(seconds=30)))",
      "",
      "        # Step 3: Ship order",
      "        await workflow.execute_activity(",
      "            ship_order, order,",
      "            start_to_close_timeout=timedelta(seconds=30))",
      "        compensations.append(lambda:",
      "            workflow.execute_activity(",
      "                cancel_shipment, tracking_id,",
      "                start_to_close_timeout=timedelta(seconds=30)))",
      "",
      "        # Step 4: Send confirmation",
      "        await workflow.execute_activity(",
      "            send_confirmation, order,",
      "            start_to_close_timeout=timedelta(seconds=30))",
      "        compensations.append(lambda:",
      "            workflow.execute_activity(",
      "                retract_email, email,",
      "                start_to_close_timeout=timedelta(seconds=30)))",
    ],
    stepLines: {
      "reserve-inventory": [6, 13],
      "charge-payment": [15, 22],
      "ship-order": [24, 31],
      "send-confirmation": [33, 40],
    },
    compLines: {
      "release-inventory": [10, 13],
      "refund-payment": [19, 22],
      "cancel-shipment": [28, 31],
      "retract-email": [37, 40],
    },
  },
};

// Tokenize every language once at setup time. The source snippets are static, so
// re-running Shiki on each render would be wasted work — the highlighter is the
// heaviest part of the component.
const highlighter = await createHighlighter({
  themes: ["github-light", "github-dark"],
  langs: ["go", "java", "python"],
});

const TOKENIZED: Record<Lang, ThemedToken[][]> = {
  go: highlighter.codeToTokens(SOURCES.go.lines.join("\n"), {
    lang: "go",
    themes: { light: "github-light", dark: "github-dark" },
    defaultColor: false,
  }).tokens,
  java: highlighter.codeToTokens(SOURCES.java.lines.join("\n"), {
    lang: "java",
    themes: { light: "github-light", dark: "github-dark" },
    defaultColor: false,
  }).tokens,
  python: highlighter.codeToTokens(SOURCES.python.lines.join("\n"), {
    lang: "python",
    themes: { light: "github-light", dark: "github-dark" },
    defaultColor: false,
  }).tokens,
};

const currentHighlight = computed<[number, number] | null>(() => {
  const src = SOURCES[lang.value];
  for (let i = props.events.length - 1; i >= 0; i--) {
    const env = props.events[i];
    if (!env) continue;
    const data = env.data as Record<string, unknown>;
    const step = String(data.step ?? "");

    if (env.type === "progress.step.started" || env.type === "progress.step.failed") {
      // Check compensation activities first
      const comp = src.compLines[step];
      if (comp) return comp;
      const line = src.stepLines[step];
      if (line) return line;
    }
    if (env.type === "progress.workflow.completed") return null;
    if (env.type === "progress.workflow.failed") return null;
  }
  return null;
});

const currentTokens = computed(() => TOKENIZED[lang.value]);

const scrollerRef = ref<HTMLElement | null>(null);
const lineRefs = ref<(HTMLElement | null)[]>([]);

// Why: scrollIntoView() would scroll the page too; manually moving the scroller keeps the jump local.
watch(
  [lang, currentHighlight],
  ([, highlight]) => {
    const scroller = scrollerRef.value;
    if (!scroller) return;

    if (highlight === null) {
      scroller.scrollTo({ top: 0, behavior: "smooth" });
      return;
    }

    const [start, end] = highlight;
    const startEl = lineRefs.value[start];
    const endEl = lineRefs.value[end] ?? startEl;
    if (!startEl || !endEl) return;

    const scrollerRect = scroller.getBoundingClientRect();
    const offset = startEl.getBoundingClientRect().top - scrollerRect.top + scroller.scrollTop;
    const rangeHeight = endEl.getBoundingClientRect().bottom - startEl.getBoundingClientRect().top;

    const visibleTop = scroller.scrollTop;
    const visibleBottom = visibleTop + scroller.clientHeight;
    if (offset >= visibleTop && offset + rangeHeight <= visibleBottom) return;

    const desired = offset - (scroller.clientHeight - rangeHeight) / 2;
    const max = Math.max(0, scroller.scrollHeight - scroller.clientHeight);
    const clamped = Math.max(0, Math.min(desired, max));
    scroller.scrollTo({ top: clamped, behavior: "smooth" });
  },
  { flush: "post" },
);
</script>

<template>
  <div class="overflow-hidden rounded-xl border border-slate-200 dark:border-slate-700">
    <div class="flex border-b border-slate-200 dark:border-slate-700">
      <button
        v-for="(src, key) in SOURCES"
        :key="key"
        class="px-4 py-2 text-xs font-mono transition-colors border-b-2"
        :class="
          key === lang
            ? 'border-blue-500 text-slate-900 dark:text-slate-100'
            : 'border-transparent text-slate-400 hover:text-slate-600 dark:hover:text-slate-300'
        "
        @click="lang = key as Lang"
      >
        {{ src.label }}
      </button>
    </div>
    <div
      ref="scrollerRef"
      class="shiki-code max-h-72 overflow-auto bg-white dark:bg-slate-900 p-4 font-mono text-[11px] leading-relaxed"
    >
      <span
        v-for="(tokens, idx) in currentTokens"
        :key="idx"
        :ref="(el) => (lineRefs[idx] = el as HTMLElement | null)"
        class="block whitespace-pre rounded px-2 py-px transition-colors duration-300"
        :class="
          currentHighlight && idx >= currentHighlight[0] && idx <= currentHighlight[1]
            ? 'bg-blue-50 dark:bg-blue-950'
            : ''
        "
      >
        <template v-if="tokens.length">
          <span v-for="(token, tIdx) in tokens" :key="tIdx" :style="token.htmlStyle">{{
            token.content
          }}</span>
        </template>
        <template v-else>&nbsp;</template>
      </span>
    </div>
  </div>
</template>

<style>
/* Dual-theme Shiki emits `--shiki-light` / `--shiki-dark` CSS vars on every
   token. `defaultColor: false` means no inline color is set, so we pick one
   explicitly based on the ancestor `.dark` class (matches main.css). */
.shiki-code span {
  color: var(--shiki-light);
}
.dark .shiki-code span {
  color: var(--shiki-dark);
}
</style>
