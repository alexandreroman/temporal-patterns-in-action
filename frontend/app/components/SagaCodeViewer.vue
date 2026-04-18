<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { CodeLang } from "~/composables/useCodeLang";
import type { CodeSource } from "~/types/code-viewer";

const props = defineProps<{
  events: EventEnvelope[];
}>();

type Lang = CodeLang;
const lang = useCodeLang();

interface SagaSource extends CodeSource {
  stepLines: Record<string, [number, number]>;
  compLines: Record<string, [number, number]>;
}

const SOURCES: Record<Lang, SagaSource> = {
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

const currentHighlight = computed<[number, number] | null>(() => {
  const src = SOURCES[lang.value];
  for (let i = props.events.length - 1; i >= 0; i--) {
    const env = props.events[i];
    if (!env) continue;
    const data = env.data as Record<string, unknown>;
    const step = String(data.step ?? "");

    if (env.type === "progress.step.started" || env.type === "progress.step.failed") {
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
</script>

<template>
  <CodeViewer :sources="SOURCES" :highlight="currentHighlight" />
</template>
