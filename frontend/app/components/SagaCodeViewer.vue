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
      "    // Step 1 — check fraud",
      "    var checkID string",
      "    if err := workflow.ExecuteActivity(ctx,",
      "        a.CheckFraud, txID, input,",
      "    ).Get(ctx, &checkID); err != nil {",
      "        runCompensations()",
      "        return result, err",
      "    }",
      "    compensations = append(compensations,",
      "        func(c workflow.Context) error {",
      "            return workflow.ExecuteActivity(c,",
      "                a.ReleaseFraudHold, txID, checkID,",
      "            ).Get(c, nil)",
      "        },",
      "    )",
      "",
      "    // Step 2 — prepare shipment",
      "    var shipmentID string",
      "    if err := workflow.ExecuteActivity(ctx,",
      "        a.PrepareShipment, txID, input, checkID,",
      "    ).Get(ctx, &shipmentID); err != nil {",
      "        runCompensations()",
      "        return result, err",
      "    }",
      "    compensations = append(compensations,",
      "        func(c workflow.Context) error {",
      "            return workflow.ExecuteActivity(c,",
      "                a.CancelShipment, txID, shipmentID,",
      "            ).Get(c, nil)",
      "        },",
      "    )",
      "",
      "    // Step 3 — charge customer",
      "    var paymentID string",
      "    if err := workflow.ExecuteActivity(ctx,",
      "        a.ChargeCustomer, txID, input, shipmentID,",
      "    ).Get(ctx, &paymentID); err != nil {",
      "        runCompensations()",
      "        return result, err",
      "    }",
      "    compensations = append(compensations,",
      "        func(c workflow.Context) error {",
      "            return workflow.ExecuteActivity(c,",
      "                a.RefundCustomer, txID, paymentID, input.Amount,",
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
      "check-fraud": [12, 26],
      "prepare-shipment": [28, 42],
      "charge-customer": [44, 58],
      "send-confirmation": [60, 66],
    },
    compLines: {
      "release-fraud-hold": [21, 25],
      "cancel-shipment": [37, 41],
      "refund-customer": [53, 57],
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
      "        // Step 1 — check fraud",
      "        String checkId = activities.checkFraud(txId, input);",
      "        saga.addCompensation(",
      "            () -> activities.releaseFraudHold(txId, checkId));",
      "",
      "        // Step 2 — prepare shipment",
      "        String shipmentId = activities.prepareShipment(txId, input, checkId);",
      "        saga.addCompensation(",
      "            () -> activities.cancelShipment(txId, shipmentId));",
      "",
      "        // Step 3 — charge customer",
      "        String paymentId = activities.chargeCustomer(txId, input, shipmentId);",
      "        saga.addCompensation(",
      "            () -> activities.refundCustomer(txId, paymentId, input.amount()));",
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
      "check-fraud": [5, 8],
      "prepare-shipment": [10, 13],
      "charge-customer": [15, 18],
      "send-confirmation": [20, 21],
    },
    compLines: {
      "release-fraud-hold": [8, 8],
      "cancel-shipment": [13, 13],
      "refund-customer": [18, 18],
    },
  },
  typescript: {
    label: "TypeScript",
    lines: [
      'import { proxyActivities } from "@temporalio/workflow";',
      'import type * as activities from "./activities";',
      "",
      "const a = proxyActivities<typeof activities>({",
      '    startToCloseTimeout: "6 seconds",',
      "});",
      "",
      "export async function orderProcessingWorkflow(",
      "    input: OrderInput,",
      "): Promise<OrderResult> {",
      "    const txID = input.transactionId;  // reused on retries — idempotency key",
      "    const compensations: Array<() => Promise<void>> = [];",
      "    try {",
      "        // Step 1 — check fraud",
      "        const checkId = await a.checkFraud(txID, input);",
      "        compensations.push(() => a.releaseFraudHold(txID, checkId));",
      "",
      "        // Step 2 — prepare shipment",
      "        const shipmentId = await a.prepareShipment(txID, input, checkId);",
      "        compensations.push(() => a.cancelShipment(txID, shipmentId));",
      "",
      "        // Step 3 — charge customer",
      "        const paymentId = await a.chargeCustomer(txID, input, shipmentId);",
      "        compensations.push(() => a.refundCustomer(txID, paymentId, input.amount));",
      "",
      "        // Step 4 — send confirmation",
      "        await a.sendConfirmation(txID, input);",
      "",
      '        return { orderId: input.orderId, status: "completed" };',
      "    } catch (err) {",
      "        for (const c of compensations.reverse()) await c();",
      "        throw err;",
      "    }",
      "}",
    ],
    stepLines: {
      "check-fraud": [13, 15],
      "prepare-shipment": [17, 19],
      "charge-customer": [21, 23],
      "send-confirmation": [25, 26],
    },
    compLines: {
      "release-fraud-hold": [15, 15],
      "cancel-shipment": [19, 19],
      "refund-customer": [23, 23],
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
      "            # Step 1 — check fraud",
      "            check_id = await workflow.execute_activity(",
      "                check_fraud, tx_id, input,",
      "                start_to_close_timeout=timedelta(seconds=6))",
      "            compensations.append(lambda:",
      "                workflow.execute_activity(",
      "                    release_fraud_hold, tx_id, check_id,",
      "                    start_to_close_timeout=timedelta(seconds=6)))",
      "",
      "            # Step 2 — prepare shipment",
      "            shipment_id = await workflow.execute_activity(",
      "                prepare_shipment, tx_id, input, check_id,",
      "                start_to_close_timeout=timedelta(seconds=6))",
      "            compensations.append(lambda:",
      "                workflow.execute_activity(",
      "                    cancel_shipment, tx_id, shipment_id,",
      "                    start_to_close_timeout=timedelta(seconds=6)))",
      "",
      "            # Step 3 — charge customer",
      "            payment_id = await workflow.execute_activity(",
      "                charge_customer, tx_id, input, shipment_id,",
      "                start_to_close_timeout=timedelta(seconds=6))",
      "            compensations.append(lambda:",
      "                workflow.execute_activity(",
      "                    refund_customer, tx_id, payment_id, input.amount,",
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
      "check-fraud": [7, 14],
      "prepare-shipment": [16, 23],
      "charge-customer": [25, 32],
      "send-confirmation": [34, 37],
    },
    compLines: {
      "release-fraud-hold": [12, 14],
      "cancel-shipment": [21, 23],
      "refund-customer": [30, 32],
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
