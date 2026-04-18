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
      "func OrderProcessingWorkflow(ctx workflow.Context, input OrderInput) (OrderResult, error) {",
      "    var a *Activities",
      "    var compensations []func(workflow.Context) error",
      "    var result OrderResult",
      "",
      "    runCompensations := func() {",
      "        for i := len(compensations) - 1; i >= 0; i-- {",
      "            _ = compensations[i](ctx)",
      "        }",
      "    }",
      "",
      "    // Step 1 — reserve inventory",
      "    var itemID string",
      "    if err := workflow.ExecuteActivity(ctx,",
      "        a.ReserveInventory, input,",
      "    ).Get(ctx, &itemID); err != nil {",
      "        runCompensations()",
      "        return result, err",
      "    }",
      "    compensations = append(compensations,",
      "        func(c workflow.Context) error {",
      "            return workflow.ExecuteActivity(c,",
      "                a.ReleaseInventory, itemID,",
      "            ).Get(c, nil)",
      "        },",
      "    )",
      "",
      "    // Step 2 — charge payment",
      "    var txnID string",
      "    if err := workflow.ExecuteActivity(ctx,",
      "        a.ChargePayment, input, itemID,",
      "    ).Get(ctx, &txnID); err != nil {",
      "        runCompensations()",
      "        return result, err",
      "    }",
      "    compensations = append(compensations,",
      "        func(c workflow.Context) error {",
      "            return workflow.ExecuteActivity(c,",
      "                a.RefundPayment, txnID, input.Amount,",
      "            ).Get(c, nil)",
      "        },",
      "    )",
      "",
      "    // Step 3 — ship the order",
      "    var trackingID string",
      "    if err := workflow.ExecuteActivity(ctx,",
      "        a.ShipOrder, input,",
      "    ).Get(ctx, &trackingID); err != nil {",
      "        runCompensations()",
      "        return result, err",
      "    }",
      "    compensations = append(compensations,",
      "        func(c workflow.Context) error {",
      "            return workflow.ExecuteActivity(c,",
      "                a.CancelShipment, trackingID,",
      "            ).Get(c, nil)",
      "        },",
      "    )",
      "",
      "    // Step 4 — send confirmation",
      "    if err := workflow.ExecuteActivity(ctx,",
      "        a.SendConfirmation, input,",
      "    ).Get(ctx, nil); err != nil {",
      "        runCompensations()",
      "        return result, err",
      "    }",
      "",
      '    result.Status = "completed"',
      "    return result, nil",
      "}",
    ],
    stepLines: {
      "reserve-inventory": [11, 25],
      "charge-payment": [27, 41],
      "ship-order": [43, 57],
      "send-confirmation": [59, 65],
    },
    compLines: {
      "release-inventory": [20, 24],
      "refund-payment": [36, 40],
      "cancel-shipment": [52, 56],
    },
  },
  java: {
    label: "Java",
    lines: [
      "@WorkflowMethod",
      "public OrderResult processOrder(OrderInput input) {",
      "    Saga saga = new Saga(new Saga.Options.Builder().build());",
      "    try {",
      "        // Step 1 — reserve inventory",
      "        String itemId = activities.reserveInventory(input);",
      "        saga.addCompensation(",
      "            () -> activities.releaseInventory(itemId));",
      "",
      "        // Step 2 — charge payment",
      "        String txnId = activities.chargePayment(input, itemId);",
      "        saga.addCompensation(",
      "            () -> activities.refundPayment(txnId, input.amount()));",
      "",
      "        // Step 3 — ship the order",
      "        String trackingId = activities.shipOrder(input);",
      "        saga.addCompensation(",
      "            () -> activities.cancelShipment(trackingId));",
      "",
      "        // Step 4 — send confirmation",
      "        activities.sendConfirmation(input);",
      "",
      '        return new OrderResult(input.orderId(), "completed");',
      "    } catch (ActivityFailure e) {",
      "        saga.compensate();",
      "        throw e;",
      "    }",
      "}",
    ],
    stepLines: {
      "reserve-inventory": [4, 7],
      "charge-payment": [9, 12],
      "ship-order": [14, 17],
      "send-confirmation": [19, 20],
    },
    compLines: {
      "release-inventory": [7, 7],
      "refund-payment": [12, 12],
      "cancel-shipment": [17, 17],
    },
  },
  python: {
    label: "Python",
    lines: [
      "@workflow.defn",
      "class OrderProcessingWorkflow:",
      "    @workflow.run",
      "    async def run(self, input: OrderInput) -> OrderResult:",
      "        compensations: list[Callable] = []",
      "        try:",
      "            # Step 1 — reserve inventory",
      "            item_id = await workflow.execute_activity(",
      "                reserve_inventory, input,",
      "                start_to_close_timeout=timedelta(seconds=6))",
      "            compensations.append(lambda:",
      "                workflow.execute_activity(",
      "                    release_inventory, item_id,",
      "                    start_to_close_timeout=timedelta(seconds=6)))",
      "",
      "            # Step 2 — charge payment",
      "            txn_id = await workflow.execute_activity(",
      "                charge_payment, input, item_id,",
      "                start_to_close_timeout=timedelta(seconds=6))",
      "            compensations.append(lambda:",
      "                workflow.execute_activity(",
      "                    refund_payment, txn_id, input.amount,",
      "                    start_to_close_timeout=timedelta(seconds=6)))",
      "",
      "            # Step 3 — ship the order",
      "            tracking_id = await workflow.execute_activity(",
      "                ship_order, input,",
      "                start_to_close_timeout=timedelta(seconds=6))",
      "            compensations.append(lambda:",
      "                workflow.execute_activity(",
      "                    cancel_shipment, tracking_id,",
      "                    start_to_close_timeout=timedelta(seconds=6)))",
      "",
      "            # Step 4 — send confirmation",
      "            await workflow.execute_activity(",
      "                send_confirmation, input,",
      "                start_to_close_timeout=timedelta(seconds=6))",
      "        except ActivityError:",
      "            for c in reversed(compensations):",
      "                await c()",
      "            raise",
      "",
      '        return OrderResult(input.order_id, "completed")',
    ],
    stepLines: {
      "reserve-inventory": [6, 13],
      "charge-payment": [15, 22],
      "ship-order": [24, 31],
      "send-confirmation": [33, 36],
    },
    compLines: {
      "release-inventory": [11, 13],
      "refund-payment": [20, 22],
      "cancel-shipment": [29, 31],
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
