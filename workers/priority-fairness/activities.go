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

// AnnounceDumpExecuted publishes one helpdesk.dump.executed business event so
// the UI can append the dumped tickets to the affected tenant's queue.
func (a *Activities) AnnounceDumpExecuted(ctx context.Context, in AnnounceDumpInput) error {
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeDumpExecuted, map[string]any{
		"tenantId": string(in.TenantID),
		"tickets":  in.Tickets,
		"count":    len(in.Tickets),
	})
	return nil
}

// AnnounceIncidentInjected publishes one helpdesk.incident.injected business
// event for the P0 ticket the workflow just queued.
func (a *Activities) AnnounceIncidentInjected(ctx context.Context, in AnnounceIncidentInput) error {
	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeIncidentInjected, map[string]any{
		"tenantId":    string(in.TenantID),
		"ticket":      in.Ticket,
		"ticketId":    in.Ticket.ID,
		"priorityKey": int(in.Ticket.Priority),
	})
	return nil
}

// ResolveTicket simulates an agent processing a ticket. It acquires a slot
// from the pool, publishes helpdesk.ticket.assigned with the agent id, sleeps
// for a random 2.0-3.0s duration to mimic resolution time, then publishes
// helpdesk.ticket.resolved.
func (a *Activities) ResolveTicket(ctx context.Context, t Ticket) error {
	pool := a.slotPoolHandle()
	agent := pool.Acquire()
	defer pool.Release(agent)

	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeTicketAssigned, map[string]any{
		"tenantId":       string(t.Tenant),
		"priorityKey":    int(t.Priority),
		"ticketId":       t.ID,
		"agent":          agent,
		"fairnessKey":    string(t.Tenant),
		"fairnessWeight": TenantWeight[t.Tenant],
	})

	dur := time.Duration(2000+rand.IntN(1000)) * time.Millisecond
	time.Sleep(dur)

	events.PublishBusiness(ctx, a.Publisher, Pattern, TypeTicketResolved, map[string]any{
		"tenantId":    string(t.Tenant),
		"priorityKey": int(t.Priority),
		"ticketId":    t.ID,
		"agent":       agent,
	})
	return nil
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
