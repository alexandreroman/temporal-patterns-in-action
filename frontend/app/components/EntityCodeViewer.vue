<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { CodeLang } from "~/composables/useCodeLang";
import type { CodeSource } from "~/types/code-viewer";

type HighlightKey =
  | "validate-item"
  | "price-item"
  | "update-qty"
  | "remove-item"
  | "process-payment"
  | "send-confirmation"
  | "getCart";

interface EntitySource extends CodeSource {
  hl: Record<HighlightKey, [number, number]>;
}

const props = defineProps<{
  events: EventEnvelope[];
}>();

const lang = useCodeLang();

const SOURCES: Record<CodeLang, EntitySource> = {
  go: {
    label: "Go",
    lines: [
      "func ShoppingCartWorkflow(ctx workflow.Context, state CartState) error {",
      '    addCh := workflow.GetSignalChannel(ctx, "addItem")',
      '    updateCh := workflow.GetSignalChannel(ctx, "updateQty")',
      '    removeCh := workflow.GetSignalChannel(ctx, "removeItem")',
      '    checkoutCh := workflow.GetSignalChannel(ctx, "checkout")',
      "",
      '    workflow.SetQueryHandler(ctx, "getCart", func() (Progress, error) {',
      "        state.QueriesAnswered++",
      "        return toProgress(state,",
      "            workflow.GetInfo(ctx).GetCurrentHistoryLength()), nil",
      "    })",
      "",
      "    var a *Activities",
      "    for !state.CheckedOut {",
      "        sel := workflow.NewSelector(ctx)",
      "",
      "        sel.AddReceive(addCh, func(ch workflow.ReceiveChannel, _ bool) {",
      "            var sig AddItemSignal",
      "            ch.Receive(ctx, &sig)",
      "            workflow.ExecuteActivity(ctx, a.ValidateItem, sig).Get(ctx, nil)",
      "            workflow.ExecuteActivity(ctx, a.PriceItem, sig).Get(ctx, nil)",
      "            state = applyAdd(state, sig)",
      "        })",
      "",
      "        sel.AddReceive(updateCh, func(ch workflow.ReceiveChannel, _ bool) {",
      "            var sig UpdateQtySignal",
      "            ch.Receive(ctx, &sig)",
      "            workflow.ExecuteActivity(ctx, a.UpdateQty, sig).Get(ctx, nil)",
      "            state = applyUpdate(state, sig)",
      "        })",
      "",
      "        sel.AddReceive(removeCh, func(ch workflow.ReceiveChannel, _ bool) {",
      "            var sig RemoveItemSignal",
      "            ch.Receive(ctx, &sig)",
      "            workflow.ExecuteActivity(ctx, a.RemoveItem, sig).Get(ctx, nil)",
      "            state = applyRemove(state, sig)",
      "        })",
      "",
      "        sel.AddReceive(checkoutCh, func(ch workflow.ReceiveChannel, _ bool) {",
      "            var sig struct{}",
      "            ch.Receive(ctx, &sig)",
      "            var orderID string",
      "            workflow.ExecuteActivity(ctx,",
      "                a.ProcessPayment, state.CartID, total(state),",
      "            ).Get(ctx, &orderID)",
      "            workflow.ExecuteActivity(ctx,",
      "                a.SendConfirmation, state.CartID, orderID,",
      "            ).Get(ctx, nil)",
      "            state.CheckedOut = true",
      "        })",
      "",
      "        sel.Select(ctx)",
      "        state.SignalsReceived++",
      "    }",
      "    return nil",
      "}",
    ],
    hl: {
      "validate-item": [16, 22],
      "price-item": [16, 22],
      "update-qty": [24, 29],
      "remove-item": [31, 36],
      "process-payment": [38, 49],
      "send-confirmation": [38, 49],
      getCart: [6, 10],
    },
  },
  java: {
    label: "Java",
    lines: [
      "@WorkflowInterface",
      "public interface ShoppingCartWorkflow {",
      "    @WorkflowMethod",
      "    void run(CartState state);",
      "",
      '    @SignalMethod(name = "addItem")',
      "    void addItem(AddItemSignal sig);",
      "",
      '    @SignalMethod(name = "updateQty")',
      "    void updateQty(UpdateQtySignal sig);",
      "",
      '    @SignalMethod(name = "removeItem")',
      "    void removeItem(RemoveItemSignal sig);",
      "",
      '    @SignalMethod(name = "checkout")',
      "    void checkout();",
      "",
      '    @QueryMethod(name = "getCart")',
      "    Progress getCart();",
      "}",
      "",
      "public class ShoppingCartWorkflowImpl implements ShoppingCartWorkflow {",
      "    private final Deque<Signal> inbox = new ArrayDeque<>();",
      "    private CartState state;",
      "    private final Activities a = Workflow.newActivityStub(",
      "        Activities.class, options);",
      "",
      "    public void run(CartState state) {",
      "        this.state = state;",
      "        while (!state.checkedOut()) {",
      "            Workflow.await(() -> !inbox.isEmpty());",
      "            Signal sig = inbox.removeFirst();",
      "            switch (sig) {",
      "                case AddItemSignal s -> {",
      "                    a.validateItem(s);",
      "                    a.priceItem(s);",
      "                    state = applyAdd(state, s);",
      "                }",
      "                case UpdateQtySignal s -> {",
      "                    a.updateQty(s);",
      "                    state = applyUpdate(state, s);",
      "                }",
      "                case RemoveItemSignal s -> {",
      "                    a.removeItem(s);",
      "                    state = applyRemove(state, s);",
      "                }",
      "                case CheckoutSignal s -> {",
      "                    String orderId = a.processPayment(",
      "                        state.cartId(), total(state));",
      "                    a.sendConfirmation(state.cartId(), orderId);",
      "                    state = state.withCheckedOut(true);",
      "                }",
      "            }",
      "            state = state.bumpSignals();",
      "        }",
      "    }",
      "",
      "    public Progress getCart() {",
      "        state = state.bumpQueries();",
      "        return toProgress(state,",
      "            Workflow.getInfo().getHistoryLength());",
      "    }",
      "}",
    ],
    hl: {
      "validate-item": [33, 37],
      "price-item": [33, 37],
      "update-qty": [38, 41],
      "remove-item": [42, 45],
      "process-payment": [46, 51],
      "send-confirmation": [46, 51],
      getCart: [57, 61],
    },
  },
  typescript: {
    label: "TypeScript",
    lines: [
      "import {",
      "    proxyActivities, defineSignal, defineQuery,",
      "    setHandler, condition, workflowInfo,",
      '} from "@temporalio/workflow";',
      'import type * as activities from "./activities";',
      "",
      "const a = proxyActivities<typeof activities>({",
      '    startToCloseTimeout: "5 seconds",',
      "});",
      "",
      'export const addItem = defineSignal<[AddItemSignal]>("addItem");',
      'export const updateQty = defineSignal<[UpdateQtySignal]>("updateQty");',
      'export const removeItem = defineSignal<[RemoveItemSignal]>("removeItem");',
      'export const checkout = defineSignal<[]>("checkout");',
      'export const getCart = defineQuery<Progress>("getCart");',
      "",
      "export async function shoppingCartWorkflow(state: CartState): Promise<void> {",
      "    const inbox: Signal[] = [];",
      '    setHandler(addItem, (s) => { inbox.push({ kind: "add", sig: s }); });',
      '    setHandler(updateQty, (s) => { inbox.push({ kind: "update", sig: s }); });',
      '    setHandler(removeItem, (s) => { inbox.push({ kind: "remove", sig: s }); });',
      '    setHandler(checkout, () => { inbox.push({ kind: "checkout" }); });',
      "",
      "    setHandler(getCart, () => {",
      "        state = { ...state, queriesAnswered: state.queriesAnswered + 1 };",
      "        return toProgress(state, workflowInfo().historyLength);",
      "    });",
      "",
      "    while (!state.checkedOut) {",
      "        await condition(() => inbox.length > 0);",
      "        const sig = inbox.shift()!;",
      "        switch (sig.kind) {",
      '            case "add":',
      "                await a.validateItem(sig.sig);",
      "                await a.priceItem(sig.sig);",
      "                state = applyAdd(state, sig.sig);",
      "                break;",
      '            case "update":',
      "                await a.updateQty(sig.sig);",
      "                state = applyUpdate(state, sig.sig);",
      "                break;",
      '            case "remove":',
      "                await a.removeItem(sig.sig);",
      "                state = applyRemove(state, sig.sig);",
      "                break;",
      '            case "checkout": {',
      "                const orderId = await a.processPayment(",
      "                    state.cartId, total(state));",
      "                await a.sendConfirmation(state.cartId, orderId);",
      "                state = { ...state, checkedOut: true };",
      "                break;",
      "            }",
      "        }",
      "        state = { ...state, signalsReceived: state.signalsReceived + 1 };",
      "    }",
      "}",
    ],
    hl: {
      "validate-item": [32, 36],
      "price-item": [32, 36],
      "update-qty": [37, 40],
      "remove-item": [41, 44],
      "process-payment": [45, 51],
      "send-confirmation": [45, 51],
      getCart: [23, 26],
    },
  },
  python: {
    label: "Python",
    lines: [
      "@workflow.defn",
      "class ShoppingCartWorkflow:",
      "    def __init__(self) -> None:",
      "        self._inbox: list[Signal] = []",
      "        self._state: CartState | None = None",
      "",
      '    @workflow.signal(name="addItem")',
      "    def add_item(self, sig: AddItemSignal) -> None:",
      '        self._inbox.append(("add", sig))',
      "",
      '    @workflow.signal(name="updateQty")',
      "    def update_qty(self, sig: UpdateQtySignal) -> None:",
      '        self._inbox.append(("update", sig))',
      "",
      '    @workflow.signal(name="removeItem")',
      "    def remove_item(self, sig: RemoveItemSignal) -> None:",
      '        self._inbox.append(("remove", sig))',
      "",
      '    @workflow.signal(name="checkout")',
      "    def checkout(self) -> None:",
      '        self._inbox.append(("checkout", None))',
      "",
      '    @workflow.query(name="getCart")',
      "    def get_cart(self) -> Progress:",
      "        assert self._state is not None",
      "        self._state = self._state.bump_queries()",
      "        return to_progress(self._state, workflow.info().history_length)",
      "",
      "    @workflow.run",
      "    async def run(self, state: CartState) -> None:",
      "        self._state = state",
      "        while not self._state.checked_out:",
      "            await workflow.wait_condition(lambda: len(self._inbox) > 0)",
      "            kind, sig = self._inbox.pop(0)",
      '            if kind == "add":',
      "                await workflow.execute_activity(validate_item, sig,",
      "                    start_to_close_timeout=timedelta(seconds=5))",
      "                await workflow.execute_activity(price_item, sig,",
      "                    start_to_close_timeout=timedelta(seconds=5))",
      "                self._state = apply_add(self._state, sig)",
      '            elif kind == "update":',
      "                await workflow.execute_activity(update_qty, sig,",
      "                    start_to_close_timeout=timedelta(seconds=5))",
      "                self._state = apply_update(self._state, sig)",
      '            elif kind == "remove":',
      "                await workflow.execute_activity(remove_item, sig,",
      "                    start_to_close_timeout=timedelta(seconds=5))",
      "                self._state = apply_remove(self._state, sig)",
      '            elif kind == "checkout":',
      "                order_id = await workflow.execute_activity(",
      "                    process_payment, self._state.cart_id, total(self._state),",
      "                    start_to_close_timeout=timedelta(seconds=5))",
      "                await workflow.execute_activity(",
      "                    send_confirmation, self._state.cart_id, order_id,",
      "                    start_to_close_timeout=timedelta(seconds=5))",
      "                self._state = self._state.with_checked_out()",
      "            self._state = self._state.bump_signals()",
    ],
    hl: {
      "validate-item": [34, 39],
      "price-item": [34, 39],
      "update-qty": [40, 43],
      "remove-item": [44, 47],
      "process-payment": [48, 55],
      "send-confirmation": [48, 55],
      getCart: [22, 26],
    },
  },
};

const STEP_KEYS: HighlightKey[] = [
  "validate-item",
  "price-item",
  "update-qty",
  "remove-item",
  "process-payment",
  "send-confirmation",
];

const highlight = computed<[number, number] | null>(() => {
  const src = SOURCES[lang.value];

  for (let i = props.events.length - 1; i >= 0; i--) {
    const env = props.events[i];
    if (!env) continue;
    const data = env.data as Record<string, unknown>;

    if (env.type === "progress.workflow.completed" || env.type === "progress.workflow.failed") {
      // Keep the last checkout step highlighted so viewers see where the
      // workflow returned rather than jumping back to the top on success.
      for (let j = props.events.length - 1; j >= 0; j--) {
        const e = props.events[j];
        if (!e) continue;
        const d = e.data as Record<string, unknown>;
        const s = String(d.step ?? "");
        if (
          e.type === "progress.step.completed" &&
          (s === "process-payment" || s === "send-confirmation")
        ) {
          return src.hl["send-confirmation"];
        }
      }
      return null;
    }
    // Synthetic envelope pushed by the page when it issues a getCart query.
    if (env.type === "entity.query.getCart") {
      return src.hl["getCart"];
    }
    if (env.type === "progress.step.started" || env.type === "progress.step.failed") {
      const step = String(data.step ?? "");
      if ((STEP_KEYS as string[]).includes(step)) {
        return src.hl[step as HighlightKey];
      }
    }
  }
  return null;
});
</script>

<template>
  <CodeViewer :sources="SOURCES" :highlight="highlight" />
</template>
