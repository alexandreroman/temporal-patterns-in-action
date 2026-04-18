<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { CodeLang } from "~/composables/useCodeLang";
import type { CodeSource } from "~/types/code-viewer";

const props = defineProps<{
  events: EventEnvelope[];
}>();

const lang = useCodeLang();

interface SagaSource extends CodeSource {
  stepLines: Record<string, [number, number]>;
  compLines: Record<string, [number, number]>;
}

const SOURCES: Record<CodeLang, SagaSource> = {
  go: {
    label: "Go",
    lines: [
      "func OrderProcessingWorkflow(ctx workflow.Context, input OrderInput) (OrderResult, error) {",
      "    txID := input.TransactionID  // reused on retries — idempotency key",
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
      "        a.ReserveInventory, txID, input,",
      "    ).Get(ctx, &itemID); err != nil {",
      "        runCompensations()",
      "        return result, err",
      "    }",
      "    compensations = append(compensations,",
      "        func(c workflow.Context) error {",
      "            return workflow.ExecuteActivity(c,",
      "                a.ReleaseInventory, txID, itemID,",
      "            ).Get(c, nil)",
      "        },",
      "    )",
      "",
      "    // Step 2 — charge payment",
      "    var paymentID string",
      "    if err := workflow.ExecuteActivity(ctx,",
      "        a.ChargePayment, txID, input, itemID,",
      "    ).Get(ctx, &paymentID); err != nil {",
      "        runCompensations()",
      "        return result, err",
      "    }",
      "    compensations = append(compensations,",
      "        func(c workflow.Context) error {",
      "            return workflow.ExecuteActivity(c,",
      "                a.RefundPayment, txID, paymentID, input.Amount,",
      "            ).Get(c, nil)",
      "        },",
      "    )",
      "",
      "    // Step 3 — ship the order",
      "    var trackingID string",
      "    if err := workflow.ExecuteActivity(ctx,",
      "        a.ShipOrder, txID, input,",
      "    ).Get(ctx, &trackingID); err != nil {",
      "        runCompensations()",
      "        return result, err",
      "    }",
      "    compensations = append(compensations,",
      "        func(c workflow.Context) error {",
      "            return workflow.ExecuteActivity(c,",
      "                a.CancelShipment, txID, trackingID,",
      "            ).Get(c, nil)",
      "        },",
      "    )",
      "",
      "    // Step 4 — send confirmation",
      "    if err := workflow.ExecuteActivity(ctx,",
      "        a.SendConfirmation, txID, input,",
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
      "reserve-inventory": [12, 26],
      "charge-payment": [28, 42],
      "ship-order": [44, 58],
      "send-confirmation": [60, 66],
    },
    compLines: {
      "release-inventory": [21, 25],
      "refund-payment": [37, 41],
      "cancel-shipment": [53, 57],
    },
  },
  java: {
    label: "Java",
    lines: [
      "@WorkflowMethod",
      "public OrderResult processOrder(OrderInput input) {",
      "    var txId = input.transactionId();  // reused on retries — idempotency key",
      "    Saga saga = new Saga(new Saga.Options.Builder().build());",
      "    try {",
      "        // Step 1 — reserve inventory",
      "        String itemId = activities.reserveInventory(txId, input);",
      "        saga.addCompensation(",
      "            () -> activities.releaseInventory(txId, itemId));",
      "",
      "        // Step 2 — charge payment",
      "        String paymentId = activities.chargePayment(txId, input, itemId);",
      "        saga.addCompensation(",
      "            () -> activities.refundPayment(txId, paymentId, input.amount()));",
      "",
      "        // Step 3 — ship the order",
      "        String trackingId = activities.shipOrder(txId, input);",
      "        saga.addCompensation(",
      "            () -> activities.cancelShipment(txId, trackingId));",
      "",
      "        // Step 4 — send confirmation",
      "        activities.sendConfirmation(txId, input);",
      "",
      '        return new OrderResult(input.orderId(), "completed");',
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
      "send-confirmation": [20, 21],
    },
    compLines: {
      "release-inventory": [8, 8],
      "refund-payment": [13, 13],
      "cancel-shipment": [18, 18],
    },
  },
  python: {
    label: "Python",
    lines: [
      "@workflow.defn",
      "class OrderProcessingWorkflow:",
      "    @workflow.run",
      "    async def run(self, input: OrderInput) -> OrderResult:",
      "        tx_id = input.transaction_id  # reused on retries — idempotency key",
      "        compensations: list[Callable] = []",
      "        try:",
      "            # Step 1 — reserve inventory",
      "            item_id = await workflow.execute_activity(",
      "                reserve_inventory, tx_id, input,",
      "                start_to_close_timeout=timedelta(seconds=6))",
      "            compensations.append(lambda:",
      "                workflow.execute_activity(",
      "                    release_inventory, tx_id, item_id,",
      "                    start_to_close_timeout=timedelta(seconds=6)))",
      "",
      "            # Step 2 — charge payment",
      "            payment_id = await workflow.execute_activity(",
      "                charge_payment, tx_id, input, item_id,",
      "                start_to_close_timeout=timedelta(seconds=6))",
      "            compensations.append(lambda:",
      "                workflow.execute_activity(",
      "                    refund_payment, tx_id, payment_id, input.amount,",
      "                    start_to_close_timeout=timedelta(seconds=6)))",
      "",
      "            # Step 3 — ship the order",
      "            tracking_id = await workflow.execute_activity(",
      "                ship_order, tx_id, input,",
      "                start_to_close_timeout=timedelta(seconds=6))",
      "            compensations.append(lambda:",
      "                workflow.execute_activity(",
      "                    cancel_shipment, tx_id, tracking_id,",
      "                    start_to_close_timeout=timedelta(seconds=6)))",
      "",
      "            # Step 4 — send confirmation",
      "            await workflow.execute_activity(",
      "                send_confirmation, tx_id, input,",
      "                start_to_close_timeout=timedelta(seconds=6))",
      "        except ActivityError:",
      "            for c in reversed(compensations):",
      "                await c()",
      "            raise",
      "",
      '        return OrderResult(input.order_id, "completed")',
    ],
    stepLines: {
      "reserve-inventory": [7, 14],
      "charge-payment": [16, 23],
      "ship-order": [25, 32],
      "send-confirmation": [34, 37],
    },
    compLines: {
      "release-inventory": [12, 14],
      "refund-payment": [21, 23],
      "cancel-shipment": [30, 32],
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
