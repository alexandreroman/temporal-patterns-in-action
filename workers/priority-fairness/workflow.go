// Package priorityfairness implements the Priority and Fairness pattern: a
// multi-tenant helpdesk dispatcher. The parent HelpdeskRunWorkflow seeds 60
// tickets evenly across 3 tiers (Mission Critical=20, Enterprise=20,
// Business=20) with the same priority mix, then fans out into one
// ResolveTicketWorkflow child per ticket at t=0. The parent constructs each
// per-ticket temporal.Priority (PriorityKey always; FairnessKey +
// FairnessWeight when fairness is enabled) and attaches it to the child's
// ChildWorkflowOptions.Priority. The ResolveTicket activity inside the
// child inherits that Priority automatically via SDK semantics, so the
// matching service still sees per-task priority on every activity
// schedule. Two signals on the parent — burst-all-tenants and
// inject-p0-incident — spawn additional children while the run is in
// flight. The worker's MaxConcurrentActivityExecutionSize=4 cap is still
// what creates the visible backlog: 60+ children all schedule their
// ResolveTicket activity onto the same task queue, the matching service
// holds the backlog, and the per-activity Priority decides which task
// fires next.
//
// Volume narrative: every tier carries the same seed count and the same
// priority distribution — the only thing that differs is the FairnessKey
// and FairnessWeight. With fairness off the matching service drains FIFO
// inside each priority bucket and the three tiers finish together; with
// fairness on the 10/3/1 weights split the slots, so Mission Critical
// drains first, Enterprise second, Business last. That contrast is the
// whole point of the demo. P0 incidents do NOT appear in the seed or in
// the burst-all-tenants signal: P0 is reserved for the explicit
// inject-p0-incident signal so the demo's "rare urgent ticket" story
// stays crisp. The burst-all-tenants signal appends BurstPerTenant P2
// tickets to every tenant simultaneously so the proportional 10/3/1
// dispatch ratio is unambiguous in the swim-lane.
//
// Design note: an earlier version released tickets through per-tenant
// arrival timers so the queue would fill gradually. Empirically that
// produced no backlog at all — Temporal timers round up to the server's
// matcher tick (≈1 s), which knocked sub-second per-tenant arrivals down
// to one ticket per tier per second and let consumption keep pace with
// arrivals. Without a backlog the matching service has nothing to
// reorder and fairness becomes invisible. Dispatching the full seed at
// t=0 sidesteps the timer-precision floor.
package priorityfairness

import (
	"fmt"
	"math/rand/v2"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// HelpdeskRunWorkflow seeds a multi-tenant ticket backlog and fans out one
// ResolveTicketWorkflow child per ticket. It also honours burst-all-tenants
// and inject-p0-incident signals by spawning additional children. The
// parent itself only runs the three announce-* activities; the per-ticket
// temporal.Priority is built here and attached to each child's
// ChildWorkflowOptions, and the activity inside the child inherits it.
func HelpdeskRunWorkflow(ctx workflow.Context, input HelpdeskInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})

	parentID := workflow.GetInfo(ctx).WorkflowExecution.ID

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
	for _, tenant := range []Tenant{TenantMissionCritical, TenantEnterprise, TenantBusiness} {
		for _, p := range seedPriorities[tenant] {
			seedTickets[tenant] = append(seedTickets[tenant], Ticket{
				ID: nextID(), Tenant: tenant, Priority: p,
			})
		}
	}

	// 2. Announce the full seed up front so the UI can populate its tenant
	//    queue panels with the planned backlog. Children are launched right
	//    after this (step 4) but the planned queue is known from t=0.
	if err := workflow.ExecuteActivity(ctx, a.AnnounceRunSeeded, AnnounceSeedInput{
		FairnessOn: input.FairnessOn,
		Tenants:    seedTickets,
	}).Get(ctx, nil); err != nil {
		return err
	}

	// 3. Dispatch helper — launch one ResolveTicketWorkflow child per ticket.
	//    The parent builds the per-ticket temporal.Priority and attaches it
	//    to ChildWorkflowOptions.Priority; the ResolveTicket activity inside
	//    the child inherits that Priority via SDK semantics, so the matching
	//    service sees the per-task Priority on the activity schedule and
	//    applies Priority + Fairness ordering across the backlog. The child
	//    WorkflowID is derived from the parent's id + ticket id so it's
	//    deterministic on replay; the TaskQueue is left empty on
	//    ChildWorkflowOptions so the SDK inherits the parent's, ensuring
	//    every ResolveTicket task lands on the same queue the matching
	//    service is sorting.
	dispatch := func(t Ticket) workflow.Future {
		p := temporal.Priority{PriorityKey: int(t.Priority)}
		if input.FairnessOn {
			p.FairnessKey = string(t.Tenant)
			p.FairnessWeight = TenantWeight[t.Tenant]
		}
		cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID: fmt.Sprintf("%s-ticket-%s", parentID, t.ID),
			Priority:   p,
		})
		return workflow.ExecuteChildWorkflow(cctx, ResolveTicketWorkflow, t)
	}

	// 4. Drop the full seed onto the matching service at t=0 — no arrival
	//    staggering. Temporal timers round up to the server's matcher tick
	//    (≈1 s), so sub-second per-tenant arrival rates can't build a real
	//    backlog: tasks get dispatched roughly as fast as they're scheduled
	//    and the matching service never sees enough queued work for
	//    fairness to reorder. Dispatching all 60 seed tickets at once
	//    gives the matching service the full pile from the start, which
	//    is what lets the 10/3/1 weights produce a visibly proportional
	//    drain. Iterate tenants in a fixed slice order so replay is
	//    deterministic.
	pending := make([]workflow.Future, 0, 64)
	for _, tenant := range []Tenant{TenantMissionCritical, TenantEnterprise, TenantBusiness} {
		for _, t := range seedTickets[tenant] {
			pending = append(pending, dispatch(t))
		}
	}

	burstCh := workflow.GetSignalChannel(ctx, SignalBurstAll)
	incidentCh := workflow.GetSignalChannel(ctx, SignalInjectP0)

	// 5. Drain loop: race child-workflow completions and signals. Loop exits
	//    when every dispatched child has completed.
	handled := make(map[int]bool)
	completed := 0
	for {
		if completed == len(pending) {
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
			for _, tenant := range []Tenant{TenantMissionCritical, TenantEnterprise, TenantBusiness} {
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
			// Every tenant's slice spawns its children at once, which is
			// the whole point of the symmetric-surge scenario. Iterate
			// the tenants in a fixed slice order — ranging over the map
			// directly would be non-deterministic on replay.
			for _, tenant := range []Tenant{TenantMissionCritical, TenantEnterprise, TenantBusiness} {
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

// ResolveTicketWorkflow runs one ticket: it executes the ResolveTicket
// activity. The per-task temporal.Priority is set by the parent on this
// child's ChildWorkflowOptions and inherited by the activity via SDK
// semantics, so no Priority field is set here.
func ResolveTicketWorkflow(ctx workflow.Context, ticket Ticket) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})

	// The activity publishes business events onto the parent's NATS subject so
	// the frontend (which only knows the parent workflow id) actually sees
	// them. Fall back to this workflow's own execution if there's no parent —
	// in this demo there always is one, but it keeps the activity safe to run
	// standalone in tests.
	info := workflow.GetInfo(ctx)
	parentID := info.WorkflowExecution.ID
	parentRunID := info.WorkflowExecution.RunID
	if info.ParentWorkflowExecution != nil {
		parentID = info.ParentWorkflowExecution.ID
		parentRunID = info.ParentWorkflowExecution.RunID
	}

	var a *Activities
	return workflow.ExecuteActivity(ctx, a.ResolveTicket, ResolveTicketActivityInput{
		Ticket:           ticket,
		ParentWorkflowID: parentID,
		ParentRunID:      parentRunID,
	}).Get(ctx, nil)
}

// generateSeed returns a fresh per-tenant priority distribution. Every
// tier gets the same count and the same priority mix so the FairnessKey /
// FairnessWeight is the only thing that distinguishes them: with fairness
// off they drain together, with fairness on the 10/3/1 weights split the
// slots and Mission Critical finishes first. Mixes use 0 % P0 by design —
// P0 incidents are only ever generated by the inject-p0-incident signal.
// Called only from inside workflow.SideEffect.
func generateSeed() map[Tenant][]PriorityKey {
	const perTier = 20
	mix := []int{0, 50, 40, 10}
	return map[Tenant][]PriorityKey{
		TenantMissionCritical: pickFromMix(perTier, mix),
		TenantEnterprise:      pickFromMix(perTier, mix),
		TenantBusiness:        pickFromMix(perTier, mix),
	}
}

// generateBurst returns BurstPerTenant P1..P3 priorities for each tenant,
// drawn uniformly at random. P0 is excluded by design — incidents only come
// from inject-p0-incident. Called only from inside workflow.SideEffect.
func generateBurst() map[Tenant][]PriorityKey {
	out := make(map[Tenant][]PriorityKey, 3)
	for _, tenant := range []Tenant{TenantMissionCritical, TenantEnterprise, TenantBusiness} {
		// pickFromMix buckets are P0..P3; weight 0 on P0 zeroes out P0,
		// equal weight on P1/P2/P3 yields the uniform draw.
		out[tenant] = pickFromMix(BurstPerTenant, []int{0, 1, 1, 1})
	}
	return out
}

// generateRandomTenant picks one of the three tenants uniformly. Called only
// from inside workflow.SideEffect.
func generateRandomTenant() Tenant {
	tenants := []Tenant{TenantMissionCritical, TenantEnterprise, TenantBusiness}
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
