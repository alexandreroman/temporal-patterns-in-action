package priorityfairness

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/events"
)

// Activities groups the helpdesk pattern activities. The Publisher is wired
// from the worker's main; tests pass a NopPublisher.
type Activities struct {
	Publisher events.Publisher

	once sync.Once
	pool *slotPool
}

// slotPoolHandle returns the activity's process-local slot pool. The pool
// tracks the worker's MaxConcurrentActivities slots so each
// `helpdesk.ticket.assigned` event carries a stable a1..a4 agent id for the
// UI cards. The pool is allocated lazily on first call so tests can
// construct an Activities with just the Publisher field. Kept unexported so
// Temporal's reflection-based RegisterActivity doesn't try to register it as
// an activity (its second return must be error).
func (a *Activities) slotPoolHandle() *slotPool {
	a.once.Do(func() { a.pool = newSlotPool() })
	return a.pool
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

// AnnounceBurstExecuted publishes one helpdesk.burst.executed business event
// so the UI can append the burst tickets to every tenant's queue at once. The
// total counts the tickets across all tenants — the event-stream label uses it
// to summarise the surge in a single line.
func (a *Activities) AnnounceBurstExecuted(ctx context.Context, in AnnounceBurstInput) error {
	total := 0
	tenants := make(map[string][]Ticket, len(in.Tenants))
	for tenant, tickets := range in.Tenants {
		tenants[string(tenant)] = tickets
		total += len(tickets)
	}
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeBurstExecuted, map[string]any{
		"tenants": tenants,
		"total":   total,
	})
	return nil
}

// AnnounceIncidentInjected publishes one helpdesk.incident.injected business
// event for the P0 ticket the workflow just queued. ticketId is kept at the
// top level so the generic event-stream label can render it without reaching
// into ticket.id.
func (a *Activities) AnnounceIncidentInjected(ctx context.Context, in AnnounceIncidentInput) error {
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeIncidentInjected, map[string]any{
		"tenantId": string(in.TenantID),
		"ticket":   in.Ticket,
		"ticketId": in.Ticket.ID,
	})
	return nil
}

// ResolveTicket simulates an agent processing a ticket. It acquires a slot
// from the pool, publishes helpdesk.ticket.assigned with the agent id, sleeps
// for a priority-dependent duration to mimic resolution time (P0 incidents
// take longer so the block stays visible in the swim-lane), then publishes
// helpdesk.ticket.resolved. Business events are published with the parent
// workflow's id (carried in the input) so they land on the NATS subject the
// frontend SSE endpoint subscribes to — the child's own id is invisible to
// the UI.
func (a *Activities) ResolveTicket(ctx context.Context, in ResolveTicketActivityInput) error {
	t := in.Ticket
	pool := a.slotPoolHandle()
	agent := pool.Acquire()
	defer pool.Release(agent)

	events.PublishBusinessAs(ctx, a.Publisher, Pattern, in.ParentWorkflowID, in.ParentRunID,
		TypeTicketAssigned, map[string]any{
			"tenantId":    string(t.Tenant),
			"priorityKey": int(t.Priority),
			"ticketId":    t.ID,
			"agent":       agent,
		})

	dur := resolutionDuration(t.Priority)
	time.Sleep(dur)

	events.PublishBusinessAs(ctx, a.Publisher, Pattern, in.ParentWorkflowID, in.ParentRunID,
		TypeTicketResolved, map[string]any{
			"tenantId":    string(t.Tenant),
			"priorityKey": int(t.Priority),
			"ticketId":    t.ID,
			"agent":       agent,
		})
	return nil
}

// resolutionDuration returns the simulated handling time for a ticket. P0
// incidents get a 2.5-3.5s window so the rare incident block stays
// unmistakable in the 20s swim-lane; P1..P3 use a 0.8-1.5s range — fast
// enough that the 60-ticket demo wraps inside the chart history while
// keeping consumption (≈3.5 tickets/s) below arrival (6 tickets/s) so a
// backlog grows for fairness to reorder. An injected P0 still gets a
// visible ~0.3-1.2s wait before a slot frees, so the swim-lane addition
// stays consistent with the resolution-log delay.
func resolutionDuration(p PriorityKey) time.Duration {
	if p == 1 {
		return time.Duration(2500+rand.IntN(1000)) * time.Millisecond
	}
	return time.Duration(800+rand.IntN(700)) * time.Millisecond
}

// slotPool tracks MaxConcurrentActivities in-process activity slots so we can
// assign each running ticket a stable a1..a4 agent id for the UI. The pool is
// mu-locked; the agent id maps to the worker pool's logical slot, not
// Temporal's matching slot.
type slotPool struct {
	mu   sync.Mutex
	busy [MaxConcurrentActivities]bool
}

func newSlotPool() *slotPool { return &slotPool{} }

// Acquire returns the next free slot id ("a1".."aN") or "a?" if the pool is
// exhausted (shouldn't happen with MaxConcurrentActivityExecutionSize set to
// MaxConcurrentActivities).
func (p *slotPool) Acquire() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := 0; i < MaxConcurrentActivities; i++ {
		if !p.busy[i] {
			p.busy[i] = true
			return fmt.Sprintf("a%d", i+1)
		}
	}
	return "a?"
}

// Release frees a previously-acquired slot. Unknown slot ids are ignored.
func (p *slotPool) Release(slot string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var i int
	if _, err := fmt.Sscanf(slot, "a%d", &i); err != nil {
		return
	}
	if i >= 1 && i <= MaxConcurrentActivities {
		p.busy[i-1] = false
	}
}
