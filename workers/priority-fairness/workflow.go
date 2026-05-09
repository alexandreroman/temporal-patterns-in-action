// Package priorityfairness implements the Priority and Fairness pattern: a
// multi-tenant helpdesk dispatcher. The workflow seeds 52 tickets across 3
// tenants, dispatches each as a ResolveTicket activity with a temporal.Priority
// attached, then drains the resulting backlog. Two signals — acme-dump-80 and
// inject-p0-incident — append more tickets while the run is in flight. The
// per-activity Priority is what makes high-priority and weighted-fairness
// dispatch visible: with the worker's MaxConcurrentActivityExecutionSize=4
// cap, the matching service decides the order in which the queued tasks fire.
package priorityfairness

import (
	"fmt"
	"math/rand/v2"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// HelpdeskRunWorkflow seeds a multi-tenant ticket backlog, dispatches each
// ticket as a prioritised activity, and drains the resulting work while
// honouring acme-dump-80 and inject-p0-incident signals.
func HelpdeskRunWorkflow(ctx workflow.Context, input HelpdeskInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})

	var a *Activities

	// 1. Generate the seed priorities deterministically. SideEffect records
	//    the result in history so replay sees the same draw.
	var seedPriorities map[Tenant][]PriorityKey
	if err := workflow.SideEffect(ctx, func(workflow.Context) any {
		return generateSeed()
	}).Get(&seedPriorities); err != nil {
		return err
	}

	// Workflow-scope ticket id counter — incremented in workflow code, never
	// inside SideEffect, so replay sees the same sequence.
	counter := 0
	nextID := func() string {
		counter++
		return fmt.Sprintf("%04d", counter)
	}

	seedTickets := map[Tenant][]Ticket{}
	for _, tenant := range []Tenant{TenantAcme, TenantBrick, TenantSolo} {
		for _, p := range seedPriorities[tenant] {
			seedTickets[tenant] = append(seedTickets[tenant], Ticket{
				ID: nextID(), Tenant: tenant, Priority: p,
			})
		}
	}

	// 2. Announce the seed. Block until done so the UI sees the seed event
	//    before any helpdesk.ticket.assigned arrives.
	if err := workflow.ExecuteActivity(ctx, a.AnnounceRunSeeded, AnnounceSeedInput{
		FairnessOn: input.FairnessOn,
		Tenants:    seedTickets,
	}).Get(ctx, nil); err != nil {
		return err
	}

	// 3. Dispatch helper — every activity carries a temporal.Priority set
	//    from the ticket. With fairness off, FairnessKey is the empty string
	//    (per the SDK contract: empty FairnessKey inherits from the workflow,
	//    which has none, so the matching service falls back to FIFO at the
	//    priority bucket).
	dispatch := func(t Ticket) workflow.Future {
		opts := workflow.ActivityOptions{
			StartToCloseTimeout: 10 * time.Second,
			Priority:            temporal.Priority{PriorityKey: int(t.Priority)},
		}
		if input.FairnessOn {
			opts.Priority.FairnessKey = string(t.Tenant)
			opts.Priority.FairnessWeight = TenantWeight[t.Tenant]
		}
		cctx := workflow.WithActivityOptions(ctx, opts)
		return workflow.ExecuteActivity(cctx, a.ResolveTicket, t)
	}

	pending := make([]workflow.Future, 0)
	for _, tenant := range []Tenant{TenantAcme, TenantBrick, TenantSolo} {
		for _, t := range seedTickets[tenant] {
			pending = append(pending, dispatch(t))
		}
	}

	dumpCh := workflow.GetSignalChannel(ctx, SignalAcmeDump80)
	incidentCh := workflow.GetSignalChannel(ctx, SignalInjectP0)

	// 4. Drain loop: race future completions against signals. New futures
	//    appended by signal handlers are picked up on the next iteration.
	handled := make(map[int]bool)
	completed := 0
	for completed < len(pending) {
		sel := workflow.NewSelector(ctx)
		for i, f := range pending {
			if handled[i] {
				continue
			}
			idx := i
			sel.AddFuture(f, func(workflow.Future) {
				handled[idx] = true
				completed++
			})
		}
		sel.AddReceive(dumpCh, func(c workflow.ReceiveChannel, _ bool) {
			c.Receive(ctx, nil)
			var dump []PriorityKey
			_ = workflow.SideEffect(ctx, func(workflow.Context) any {
				return generateDump80()
			}).Get(&dump)
			tickets := make([]Ticket, 0, len(dump))
			for _, p := range dump {
				tickets = append(tickets, Ticket{ID: nextID(), Tenant: TenantAcme, Priority: p})
			}
			// Block on the announce so the queue update lands before any
			// helpdesk.ticket.assigned event from the dispatched tickets.
			_ = workflow.ExecuteActivity(ctx, a.AnnounceDumpExecuted, AnnounceDumpInput{
				TenantID: TenantAcme, Tickets: tickets,
			}).Get(ctx, nil)
			for _, t := range tickets {
				pending = append(pending, dispatch(t))
			}
		})
		sel.AddReceive(incidentCh, func(c workflow.ReceiveChannel, _ bool) {
			c.Receive(ctx, nil)
			var tenant Tenant
			_ = workflow.SideEffect(ctx, func(workflow.Context) any {
				return generateRandomTenant()
			}).Get(&tenant)
			t := Ticket{ID: nextID(), Tenant: tenant, Priority: 1} // P0
			_ = workflow.ExecuteActivity(ctx, a.AnnounceIncidentInjected, AnnounceIncidentInput{
				TenantID: tenant, Ticket: t,
			}).Get(ctx, nil)
			pending = append(pending, dispatch(t))
		})
		sel.Select(ctx)
	}
	return nil
}

// generateSeed returns a fresh per-tenant priority distribution matching the
// frontend's INITIAL_SEED constants. Called only from inside workflow.SideEffect.
func generateSeed() map[Tenant][]PriorityKey {
	return map[Tenant][]PriorityKey{
		TenantAcme:  pickFromMix(35, []int{5, 20, 55, 20}),
		TenantBrick: pickFromMix(12, []int{8, 25, 50, 17}),
		TenantSolo:  pickFromMix(5, []int{10, 30, 50, 10}),
	}
}

// generateDump80 returns the 80-ticket dump distribution for Acme. Called
// only from inside workflow.SideEffect.
func generateDump80() []PriorityKey {
	return pickFromMix(80, []int{2, 10, 78, 10})
}

// generateRandomTenant picks one of the three tenants uniformly. Called only
// from inside workflow.SideEffect.
func generateRandomTenant() Tenant {
	tenants := []Tenant{TenantAcme, TenantBrick, TenantSolo}
	return tenants[rand.IntN(len(tenants))]
}

// pickFromMix samples count priorities using a 4-bucket weighted distribution.
// Mix slots correspond to PriorityKey 1..4 (P0..P3). Used inside SideEffect.
func pickFromMix(count int, mix []int) []PriorityKey {
	total := 0
	for _, v := range mix {
		total += v
	}
	out := make([]PriorityKey, 0, count)
	for i := 0; i < count; i++ {
		r := rand.IntN(total)
		acc := 0
		chosen := PriorityKey(4)
		for k, v := range mix {
			acc += v
			if r < acc {
				chosen = PriorityKey(k + 1)
				break
			}
		}
		out = append(out, chosen)
	}
	return out
}
