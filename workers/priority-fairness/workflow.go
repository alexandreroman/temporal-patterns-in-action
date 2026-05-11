// Package priorityfairness implements the Priority and Fairness pattern: a
// multi-tenant helpdesk dispatcher. The workflow seeds 52 tickets across 3
// tenants (Acme=5, Brick=12, Solo=35), then releases them onto the task queue
// at per-tenant arrival rates so the matching service always has a backlog
// to sort by Priority + Fairness. Two signals — burst-all-tenants and
// inject-p0-incident — append more tickets while the run is in flight. The
// per-activity Priority is what makes high-priority and weighted-fairness
// dispatch visible: with the worker's MaxConcurrentActivityExecutionSize=4
// cap and arrivals outpacing consumption, the matching service decides the
// order in which the queued tasks fire.
//
// Volume narrative: Solo (lowest weight, weight=1) carries the largest seed
// backlog so fairness has something visible to do — without fairness Solo
// starves behind the bigger-weight tenants; with fairness Solo gets a
// proportional slot share from the start. P0 incidents do NOT appear in the
// seed or in the burst-all-tenants signal: P0 is reserved for the explicit
// inject-p0-incident signal so the demo's "rare urgent ticket" story stays
// crisp. The burst-all-tenants signal appends BurstPerTenant P2 tickets to
// every tenant simultaneously so the proportional 10/3/1 dispatch ratio is
// unambiguous in the swim-lane.
package priorityfairness

import (
	"fmt"
	"math/rand/v2"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// HelpdeskRunWorkflow seeds a multi-tenant ticket backlog, releases each
// ticket onto the task queue at its tenant's arrival rate, and drains the
// resulting work while honouring burst-all-tenants and inject-p0-incident
// signals.
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

	// 2. Announce the full seed up front so the UI can populate its tenant
	//    queue panels with the planned backlog. Activities will be dispatched
	//    progressively (see the arrival timers below) but the planned queue
	//    is known from t=0.
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

	// 4. Per-tenant arrival drivers. Each tenant releases its seed tickets
	//    one at a time at ArrivalInterval[tenant]. The first arrival fires
	//    after one interval (not at t=0), so the matching service has time
	//    to receive several tickets before the first slot frees — the
	//    backlog is what makes priority + fairness ordering observable.
	type arrival struct {
		tenant Tenant
		queue  []Ticket
		timer  workflow.Future // nil once this tenant has drained its seed
	}
	arrivals := make([]*arrival, 0, 3)
	for _, tenant := range []Tenant{TenantAcme, TenantBrick, TenantSolo} {
		q := seedTickets[tenant]
		if len(q) == 0 {
			continue
		}
		arrivals = append(arrivals, &arrival{
			tenant: tenant,
			queue:  q,
			timer:  workflow.NewTimer(ctx, ArrivalInterval[tenant]),
		})
	}

	burstCh := workflow.GetSignalChannel(ctx, SignalBurstAll)
	incidentCh := workflow.GetSignalChannel(ctx, SignalInjectP0)

	// 5. Drain loop: race future completions, arrival timers, and signals.
	//    Loop exits when every dispatched activity has completed AND every
	//    tenant has drained its seed (no more arrival timers pending).
	pending := make([]workflow.Future, 0, 64)
	handled := make(map[int]bool)
	completed := 0
	for {
		pendingArrivals := false
		for _, ar := range arrivals {
			if ar.timer != nil {
				pendingArrivals = true
				break
			}
		}
		if completed == len(pending) && !pendingArrivals {
			break
		}

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
		for _, ar := range arrivals {
			if ar.timer == nil {
				continue
			}
			a := ar
			sel.AddFuture(ar.timer, func(workflow.Future) {
				if len(a.queue) == 0 {
					a.timer = nil
					return
				}
				t := a.queue[0]
				a.queue = a.queue[1:]
				pending = append(pending, dispatch(t))
				if len(a.queue) > 0 {
					a.timer = workflow.NewTimer(ctx, ArrivalInterval[a.tenant])
				} else {
					a.timer = nil
				}
			})
		}
		sel.AddReceive(burstCh, func(c workflow.ReceiveChannel, _ bool) {
			c.Receive(ctx, nil)
			// Priorities are uniformly random over P1..P3 — P0 stays out
			// (incidents only come from inject-p0-incident). The matching
			// service sorts by priority first, then by fairness key inside
			// each bucket, so the surge produces an interleaved drain where
			// the 10/3/1 weight ratio decides ordering within every bucket.
			var burstPriorities map[Tenant][]PriorityKey
			_ = workflow.SideEffect(ctx, func(workflow.Context) any {
				return generateBurst()
			}).Get(&burstPriorities)
			burst := make(map[Tenant][]Ticket, 3)
			for _, tenant := range []Tenant{TenantAcme, TenantBrick, TenantSolo} {
				prios := burstPriorities[tenant]
				tickets := make([]Ticket, 0, len(prios))
				for _, p := range prios {
					tickets = append(tickets, Ticket{ID: nextID(), Tenant: tenant, Priority: p})
				}
				burst[tenant] = tickets
			}
			// Block on the announce so the queue update lands before any
			// helpdesk.ticket.assigned event from the dispatched tickets.
			_ = workflow.ExecuteActivity(ctx, a.AnnounceBurstExecuted, AnnounceBurstInput{
				Tenants: burst,
			}).Get(ctx, nil)
			// Burst tickets bypass the per-tenant arrival timer: every
			// tenant's slice lands on the matching service at once, which
			// is the whole point of the symmetric-surge scenario. Iterate
			// the tenants in a fixed slice order — ranging over the map
			// directly would be non-deterministic on replay.
			for _, tenant := range []Tenant{TenantAcme, TenantBrick, TenantSolo} {
				for _, t := range burst[tenant] {
					pending = append(pending, dispatch(t))
				}
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

// generateSeed returns a fresh per-tenant priority distribution. Volumes are
// inverted relative to tenant weight so fairness has something visible to do:
// Acme (weight 10) carries few tickets, Solo (weight 1) carries the largest
// backlog. Mixes use 0 % P0 by design — P0 incidents are only ever generated
// by the inject-p0-incident signal. Called only from inside workflow.SideEffect.
func generateSeed() map[Tenant][]PriorityKey {
	return map[Tenant][]PriorityKey{
		TenantAcme:  pickFromMix(5, []int{0, 25, 55, 20}),
		TenantBrick: pickFromMix(12, []int{0, 33, 50, 17}),
		TenantSolo:  pickFromMix(35, []int{0, 40, 50, 10}),
	}
}

// generateBurst returns BurstPerTenant P1..P3 priorities for each tenant,
// drawn uniformly at random. P0 is excluded by design — incidents only come
// from inject-p0-incident. Called only from inside workflow.SideEffect.
func generateBurst() map[Tenant][]PriorityKey {
	out := make(map[Tenant][]PriorityKey, 3)
	for _, tenant := range []Tenant{TenantAcme, TenantBrick, TenantSolo} {
		// pickFromMix buckets are P0..P3; weight 0 on P0 zeroes out P0,
		// equal weight on P1/P2/P3 yields the uniform draw.
		out[tenant] = pickFromMix(BurstPerTenant, []int{0, 1, 1, 1})
	}
	return out
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
