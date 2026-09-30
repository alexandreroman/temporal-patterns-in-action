package priorityfairness

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

// Activities groups the helpdesk pattern activities. The Publisher and
// Client are wired from the worker's main; tests pass a NopPublisher and
// usually mock StartResolveTicket so the nil Client never gets dialled.
type Activities struct {
	Publisher events.Publisher
	Client    client.Client

	// slots gives each `helpdesk.ticket.assigned` event a stable A1..A4 agent
	// id for the UI cards. Its zero value is ready to use.
	slots slotPool
}

// AnnounceRunSeeded publishes one helpdesk.run.seeded business event so the
// UI can populate its tenant lanes. Pure side-effect activity.
func (a *Activities) AnnounceRunSeeded(ctx context.Context, in AnnounceSeedInput) error {
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeRunSeeded, map[string]any{
		"fairnessOn": in.FairnessOn,
		"tenants":    in.Tenants,
	})
	return nil
}

// AnnounceIncidentInjected publishes one helpdesk.incident.injected business
// event for the P0 ticket the workflow just queued. ticketId is kept at the
// top level so the generic event-stream label can render it without reaching
// into ticket.id.
func (a *Activities) AnnounceIncidentInjected(ctx context.Context, in AnnounceIncidentInput) error {
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeIncidentInjected, map[string]any{
		"tenant":   string(in.TenantID),
		"ticket":   in.Ticket,
		"ticketId": in.Ticket.ID,
	})
	return nil
}

// StartResolveTicket is a local activity that hands a single ticket off to
// the Temporal client, creating a brand-new top-level ResolveTicketWorkflow
// (not a ChildWorkflow). The per-ticket temporal.Priority is built here and
// pinned on StartWorkflowOptions; the ResolveTicket activity inside the new
// workflow inherits that Priority via SDK semantics, so the matching service
// sees per-task Priority on every schedule. Returns as soon as the workflow
// is created — completion is awaited later by WaitTicketDone.
func (a *Activities) StartResolveTicket(ctx context.Context, in StartResolveTicketInput) error {
	p := temporal.Priority{PriorityKey: int(in.PriorityKey)}
	if in.FairnessOn {
		p.FairnessKey = string(in.Ticket.Tenant)
		p.FairnessWeight = TenantWeight[in.Ticket.Tenant]
	}
	_, err := a.Client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        in.WorkflowID,
		TaskQueue: TaskQueue,
		Priority:  p,
	}, ResolveTicketWorkflow, ResolveTicketWorkflowInput{
		Ticket:           in.Ticket,
		ParentWorkflowID: in.ParentWorkflowID,
		ParentRunID:      in.ParentRunID,
	})
	return err
}

// WaitTicketDone blocks until the named ResolveTicketWorkflow closes. It is
// invoked as a local activity by HelpdeskRunWorkflow so the dispatcher can
// drain without the per-ticket workflows having to signal back. The wait is
// a server-side long poll on workflow history (not visibility polling), so it
// scales fine even with hundreds of in-flight tickets. On worker restart the
// helpdesk workflow replays and re-issues this activity; if the ticket
// workflow has already closed, the long poll returns immediately with the
// recorded result and the drain loop catches up without manual recovery.
func (a *Activities) WaitTicketDone(ctx context.Context, workflowID string) error {
	err := a.Client.GetWorkflow(ctx, workflowID, "").Get(ctx, nil)
	// A ticket workflow that closed with an error (failed, cancelled,
	// terminated, timed out) is still done: returning nil stops the
	// local-activity retries so the drain loop counts it.
	var wee *temporal.WorkflowExecutionError
	if errors.As(err, &wee) {
		return nil
	}
	return err
}

// ResolveTicket simulates an agent processing a ticket. It acquires a slot
// from the pool, publishes helpdesk.ticket.assigned with the agent id, sleeps
// for a priority-dependent duration to mimic resolution time (P0 incidents
// take longer so the block stays visible in the swim-lane), then publishes
// helpdesk.ticket.resolved. The activity runs inside the per-ticket
// ResolveTicketWorkflow, so its activity-context workflow id is the
// per-ticket workflow's — we publish business events with the helpdesk
// run's id (carried in the input) so they land on the NATS subject the
// frontend SSE endpoint subscribes to.
func (a *Activities) ResolveTicket(ctx context.Context, in ResolveTicketWorkflowInput) error {
	t := in.Ticket
	slot := a.slots.Acquire()
	defer a.slots.Release(slot)
	agent := "A?"
	if slot >= 0 {
		agent = fmt.Sprintf("A%d", slot+1)
	}

	events.PublishBusinessAs(ctx, a.Publisher, Pattern, in.ParentWorkflowID, in.ParentRunID,
		TypeTicketAssigned, map[string]any{
			"tenant":   string(t.Tenant),
			"priority": int(t.Priority),
			"ticketId": t.ID,
			"agent":    agent,
		})

	dur := resolutionDuration(t.Priority)
	time.Sleep(dur)

	events.PublishBusinessAs(ctx, a.Publisher, Pattern, in.ParentWorkflowID, in.ParentRunID,
		TypeTicketResolved, map[string]any{
			"tenant":   string(t.Tenant),
			"priority": int(t.Priority),
			"ticketId": t.ID,
			"agent":    agent,
		})
	return nil
}

// resolutionDuration returns the simulated handling time for a ticket. P0
// incidents take a fixed 3s so the rare incident block stays unmistakable
// in the 20s swim-lane; P1..P3 take a fixed 1.2s. With 4 slots draining at
// 1.2s/ticket, consumption is ≈3.3 tickets/s — below the seed arrival rate,
// so a backlog grows and fairness has something to reorder. An injected P0
// still gets a visible ~0.3-1.2s wait before a slot frees, so the swim-lane
// addition stays consistent with the resolution-log delay.
func resolutionDuration(p PriorityKey) time.Duration {
	if p == 1 {
		return 3 * time.Second
	}
	return 1200 * time.Millisecond
}

// slotPool tracks MaxConcurrentActivities in-process activity slots so we can
// assign each running ticket a stable A1..A4 agent id for the UI. The pool is
// mu-locked; the agent id maps to the worker pool's logical slot, not
// Temporal's matching slot.
type slotPool struct {
	mu   sync.Mutex
	busy [MaxConcurrentActivities]bool
}

// Acquire marks the first free slot busy and returns its index, or -1 if the
// pool is exhausted (shouldn't happen with MaxConcurrentActivityExecutionSize
// set to MaxConcurrentActivities).
func (p *slotPool) Acquire() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := range p.busy {
		if !p.busy[i] {
			p.busy[i] = true
			return i
		}
	}
	return -1
}

// Release frees a slot returned by Acquire. A negative index (no slot was
// acquired) is ignored.
func (p *slotPool) Release(i int) {
	if i < 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.busy[i] = false
}
