// Package entity implements the entity workflow pattern: a long-lived
// workflow whose ID maps one-to-one to a business entity (here, a shopping
// cart). It mutates state in response to signals and answers queries
// against that state.
package entity

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// ShoppingCartWorkflow is a long-lived entity workflow: each user cart maps to
// one workflow instance (workflowID = cartID). The workflow blocks on a signal
// selector, mutates state in response to addItem/updateQty/removeItem/checkout
// signals, and terminates only on checkout.
func ShoppingCartWorkflow(ctx workflow.Context, state CartState) error {
	logger := workflow.GetLogger(ctx)

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	})

	// Queries are side-effect-free observers by Temporal contract, but
	// QueriesAnswered is explicitly advisory (see types.go) — we only mutate
	// it for UI counters, never for replay branching.
	if err := workflow.SetQueryHandler(ctx, QueryGetCart, func() (Progress, error) {
		state.QueriesAnswered++
		return buildProgress(ctx, &state), nil
	}); err != nil {
		return err
	}

	addCh := workflow.GetSignalChannel(ctx, SignalAddItem)
	updateCh := workflow.GetSignalChannel(ctx, SignalUpdateQty)
	removeCh := workflow.GetSignalChannel(ctx, SignalRemoveItem)
	checkoutCh := workflow.GetSignalChannel(ctx, SignalCheckout)

	var a *Activities

	for {
		var (
			addSig      AddItemSignal
			updateSig   UpdateQtySignal
			removeSig   RemoveItemSignal
			checkoutSig CheckoutSignal
			gotAdd      bool
			gotUpdate   bool
			gotRemove   bool
			gotCheckout bool
		)

		sel := workflow.NewSelector(ctx)
		sel.AddReceive(addCh, func(c workflow.ReceiveChannel, _ bool) {
			c.Receive(ctx, &addSig)
			gotAdd = true
		})
		sel.AddReceive(updateCh, func(c workflow.ReceiveChannel, _ bool) {
			c.Receive(ctx, &updateSig)
			gotUpdate = true
		})
		sel.AddReceive(removeCh, func(c workflow.ReceiveChannel, _ bool) {
			c.Receive(ctx, &removeSig)
			gotRemove = true
		})
		sel.AddReceive(checkoutCh, func(c workflow.ReceiveChannel, _ bool) {
			c.Receive(ctx, &checkoutSig)
			gotCheckout = true
		})
		sel.Select(ctx)

		state.SignalsReceived++

		switch {
		case gotAdd:
			if err := workflow.ExecuteActivity(ctx, a.ValidateItem, addSig).Get(ctx, nil); err != nil {
				logger.Warn("validate-item failed", "error", err)
				continue
			}
			var priceCents int
			if err := workflow.ExecuteActivity(ctx, a.PriceItem, addSig).Get(ctx, &priceCents); err != nil {
				logger.Warn("price-item failed", "error", err)
				continue
			}
			mergeItem(&state, CartItem{
				ItemID:     addSig.ItemID,
				Name:       addSig.Name,
				PriceCents: priceCents,
				Qty:        addSig.Qty,
			})

		case gotUpdate:
			if err := workflow.ExecuteActivity(ctx, a.UpdateQty, updateSig).Get(ctx, nil); err != nil {
				logger.Warn("update-qty failed", "error", err)
				continue
			}
			applyQtyUpdate(&state, updateSig)

		case gotRemove:
			if err := workflow.ExecuteActivity(ctx, a.RemoveItem, removeSig).Get(ctx, nil); err != nil {
				logger.Warn("remove-item failed", "error", err)
				continue
			}
			applyRemove(&state, removeSig.ItemID)

		case gotCheckout:
			totalCents := cartTotal(state.Items)
			var orderID string
			if err := workflow.ExecuteActivity(ctx, a.ProcessPayment, state.CartID, totalCents).Get(ctx, &orderID); err != nil {
				return err
			}
			if err := workflow.ExecuteActivity(ctx, a.SendConfirmation, state.CartID, orderID).Get(ctx, nil); err != nil {
				return err
			}
			state.CheckedOut = true
			return nil
		}
	}
}

// buildProgress materialises a Progress snapshot for the getCart query.
func buildProgress(ctx workflow.Context, state *CartState) Progress {
	items := make([]CartItem, len(state.Items))
	copy(items, state.Items)
	return Progress{
		CartID:          state.CartID,
		Items:           items,
		TotalCents:      cartTotal(state.Items),
		SignalsReceived: state.SignalsReceived,
		QueriesAnswered: state.QueriesAnswered,
		CheckedOut:      state.CheckedOut,
		HistoryLength:   workflow.GetInfo(ctx).GetCurrentHistoryLength(),
	}
}

func cartTotal(items []CartItem) int {
	total := 0
	for _, it := range items {
		total += it.PriceCents * it.Qty
	}
	return total
}

// mergeItem appends an item to the cart, or increments qty on an existing line
// when the itemID already present.
func mergeItem(state *CartState, item CartItem) {
	for i := range state.Items {
		if state.Items[i].ItemID == item.ItemID {
			state.Items[i].Qty += item.Qty
			// Refresh price/name in case they drifted (demo convenience).
			if item.PriceCents > 0 {
				state.Items[i].PriceCents = item.PriceCents
			}
			if item.Name != "" {
				state.Items[i].Name = item.Name
			}
			return
		}
	}
	state.Items = append(state.Items, item)
}

// applyQtyUpdate sets the qty of a matching line; a qty ≤ 0 removes it.
func applyQtyUpdate(state *CartState, sig UpdateQtySignal) {
	if sig.Qty <= 0 {
		applyRemove(state, sig.ItemID)
		return
	}
	for i := range state.Items {
		if state.Items[i].ItemID == sig.ItemID {
			state.Items[i].Qty = sig.Qty
			return
		}
	}
}

func applyRemove(state *CartState, itemID string) {
	out := make([]CartItem, 0, len(state.Items))
	for _, it := range state.Items {
		if it.ItemID != itemID {
			out = append(out, it)
		}
	}
	state.Items = out
}
