package priorityfairness

// Pattern is the pattern name used in the NATS subject and envelope source.
const Pattern = "priority-fairness"

// IMPORTANT: the frontend components (PriorityFairness*) consume `helpdesk.*`
// type strings from the existing client-side simulation. Keep the worker
// emitting the same strings so the components stay untouched. This is a
// deliberate exception to the project's "<pattern>.<subtype>" convention; the
// `helpdesk` prefix is a domain name. Do not change without updating the
// components in lockstep.
const (
	TypeRunSeeded        = "helpdesk.run.seeded"
	TypeTicketAssigned   = "helpdesk.ticket.assigned"
	TypeTicketResolved   = "helpdesk.ticket.resolved"
	TypeBurstExecuted    = "helpdesk.burst.executed"
	TypeIncidentInjected = "helpdesk.incident.injected"
)
