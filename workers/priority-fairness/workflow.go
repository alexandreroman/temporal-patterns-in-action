// Package priorityfairness implements the Priority and Fairness pattern: a
// multi-tenant helpdesk dispatcher. The HelpdeskRunWorkflow seeds 120
// tickets evenly across 3 tiers (Mission Critical=40, Enterprise=40,
// Business=40) with the same priority mix, then starts one separate
// top-level ResolveTicketWorkflow per ticket at t=0 — explicitly NOT a
// ChildWorkflow, but a fresh workflow created via the Temporal client from
// inside a local activity (StartResolveTicket). The helpdesk workflow
// constructs each per-ticket temporal.Priority (PriorityKey always;
// FairnessKey + FairnessWeight when fairness is enabled) and pins it on the
// new workflow's StartWorkflowOptions; the ResolveTicket activity inside
// the new workflow inherits that Priority via SDK semantics, so the
// matching service still sees per-task priority on every activity schedule.
// One signal on the parent — inject-p0-incident — kicks off an additional
// P0 ResolveTicketWorkflow while the run is in flight. The worker's
// MaxConcurrentActivityExecutionSize=4 cap is what creates the visible
// backlog: 120+ ResolveTicket activities land on the same task queue, the
// matching service holds the backlog, and the per-activity Priority decides
// which task fires next. Each ResolveTicketWorkflow signals SignalTicketDone
// back to the helpdesk workflow on completion, so the parent can drain
// without keeping ChildWorkflowFutures around.
//
// Volume narrative: every tier carries the same seed count and the same
// priority distribution — the only thing that differs is the FairnessKey
// and FairnessWeight. With fairness off the matching service drains FIFO
// inside each priority bucket and the three tiers finish together; with
// fairness on the 10/3/1 weights split the slots, so Mission Critical
// drains first, Enterprise second, Business last. That contrast is the
// whole point of the demo. P0 incidents do NOT appear in the seed: P0
// is reserved for the explicit inject-p0-incident signal so the demo's
// "rare urgent ticket" story stays crisp.
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

	"go.temporal.io/sdk/workflow"
)

// HelpdeskRunWorkflow seeds a multi-tenant ticket backlog and dispatches one
// top-level ResolveTicketWorkflow per ticket. It honours the
// inject-p0-incident signal by dispatching an additional P0 workflow, and
// waits for one SignalTicketDone per dispatched ticket before completing.
// The per-ticket temporal.Priority is built here and handed off to a local
// activity (StartResolveTicket) that uses the Temporal client to create the
// new workflow with StartWorkflowOptions.Priority set; the ResolveTicket
// activity inside the new workflow inherits that Priority via SDK semantics.
func HelpdeskRunWorkflow(ctx workflow.Context, input HelpdeskInput) error {
	// All of this workflow's own activity dispatches run as local
	// activities. The dispatch path (StartResolveTicket) just calls the
	// Temporal client to create a new top-level workflow — it doesn't
	// belong on the prioritised task queue, since putting it there would
	// queue the dispatcher behind the very backlog it is creating and
	// throttle the demo's "fire 120 tickets at t=0" story. Announce
	// activities use the same context for the same reason explained in the
	// SignalInjectP0 path below.
	lctx := workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	})

	info := workflow.GetInfo(ctx)
	parentID := info.WorkflowExecution.ID
	parentRunID := info.WorkflowExecution.RunID

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
	//    queue panels with the planned backlog. Top-level workflows are
	//    started right after this (step 4) but the planned queue is known
	//    from t=0.
	if err := workflow.ExecuteLocalActivity(lctx, a.AnnounceRunSeeded, AnnounceSeedInput{
		FairnessOn: input.FairnessOn,
		Tenants:    seedTickets,
	}).Get(ctx, nil); err != nil {
		return err
	}

	// 3. Dispatch helper — start one top-level ResolveTicketWorkflow per
	//    ticket via the StartResolveTicket local activity. The activity
	//    runs in-process and calls Client.ExecuteWorkflow with a
	//    StartWorkflowOptions that carries the per-ticket Priority +
	//    Fairness; the ResolveTicket activity inside the new workflow
	//    inherits that Priority. We deliberately do NOT use
	//    workflow.ExecuteChildWorkflow here — these are sibling top-level
	//    workflows, not children of the helpdesk run.
	dispatch := func(t Ticket) {
		_ = workflow.ExecuteLocalActivity(lctx, a.StartResolveTicket, StartResolveTicketInput{
			WorkflowID:       fmt.Sprintf("%s-ticket-%s", parentID, t.ID),
			Ticket:           t,
			ParentWorkflowID: parentID,
			ParentRunID:      parentRunID,
			PriorityKey:      t.Priority,
			FairnessOn:       input.FairnessOn,
		}).Get(ctx, nil)
	}

	// 4. Drop the full seed onto the matching service at t=0 — no arrival
	//    staggering. Temporal timers round up to the server's matcher tick
	//    (≈1 s), so sub-second per-tenant arrival rates can't build a real
	//    backlog: tasks get dispatched roughly as fast as they're scheduled
	//    and the matching service never sees enough queued work for
	//    fairness to reorder. Dispatching all 120 seed tickets at once
	//    gives the matching service the full pile from the start, which
	//    is what lets the 10/3/1 weights produce a visibly proportional
	//    drain. Iterate tenants in a fixed slice order so replay is
	//    deterministic.
	expected := 0
	for _, tenant := range []Tenant{TenantMissionCritical, TenantEnterprise, TenantBusiness} {
		for _, t := range seedTickets[tenant] {
			dispatch(t)
			expected++
		}
	}

	incidentCh := workflow.GetSignalChannel(ctx, SignalInjectP0)
	doneCh := workflow.GetSignalChannel(ctx, SignalTicketDone)

	// 5. Drain loop: count SignalTicketDone receipts (one per dispatched
	//    workflow) and listen for inject-p0 in parallel. Each P0 injection
	//    bumps the expected count by one.
	completed := 0
	for completed < expected {
		sel := workflow.NewSelector(ctx)
		sel.AddReceive(doneCh, func(c workflow.ReceiveChannel, _ bool) {
			var ticketID string
			c.Receive(ctx, &ticketID)
			completed++
		})
		sel.AddReceive(incidentCh, func(c workflow.ReceiveChannel, _ bool) {
			c.Receive(ctx, nil)
			var tenant Tenant
			_ = workflow.SideEffect(ctx, func(workflow.Context) any {
				return generateRandomTenant()
			}).Get(&tenant)
			t := Ticket{ID: nextID(), Tenant: tenant, Priority: 1} // P0
			_ = workflow.ExecuteLocalActivity(lctx, a.AnnounceIncidentInjected, AnnounceIncidentInput{
				TenantID: tenant, Ticket: t,
			}).Get(ctx, nil)
			dispatch(t)
			expected++
		})
		sel.Select(ctx)
	}
	return nil
}

// ResolveTicketWorkflow resolves a single ticket as its own top-level
// workflow. The Priority is set by the caller on StartWorkflowOptions and
// inherited by the ResolveTicket activity via SDK semantics, so this
// workflow body never sets a Priority of its own. On completion it signals
// SignalTicketDone back to the helpdesk run so the parent's drain loop can
// count.
func ResolveTicketWorkflow(ctx workflow.Context, in ResolveTicketWorkflowInput) error {
	actx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})

	var a *Activities
	if err := workflow.ExecuteActivity(actx, a.ResolveTicket, ResolveTicketActivityInput{
		Ticket:           in.Ticket,
		ParentWorkflowID: in.ParentWorkflowID,
		ParentRunID:      in.ParentRunID,
	}).Get(ctx, nil); err != nil {
		return err
	}

	// Signal completion back to the helpdesk run. Empty run id targets the
	// current run of the parent workflow id. Best-effort: if the parent has
	// already closed, the SignalExternalWorkflow future fails, but the
	// per-ticket work is done so we still return success.
	_ = workflow.SignalExternalWorkflow(ctx, in.ParentWorkflowID, "", SignalTicketDone, in.Ticket.ID).
		Get(ctx, nil)
	return nil
}

// generateSeed returns a fresh per-tenant priority distribution. Every
// tier gets the same count and the same priority mix so the FairnessKey /
// FairnessWeight is the only thing that distinguishes them: with fairness
// off they drain together, with fairness on the 10/3/1 weights split the
// slots and Mission Critical finishes first. Mixes use 0 % P0 by design —
// P0 incidents only enter the run via the inject-p0-incident signal.
// Called only from inside workflow.SideEffect.
func generateSeed() map[Tenant][]PriorityKey {
	const perTier = 40
	mix := []int{0, 50, 40, 10}
	return map[Tenant][]PriorityKey{
		TenantMissionCritical: pickFromMix(perTier, mix),
		TenantEnterprise:      pickFromMix(perTier, mix),
		TenantBusiness:        pickFromMix(perTier, mix),
	}
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
