package priorityfairness

// TaskQueue is the Temporal task queue used by the priority-fairness pattern
// worker. The worker opts into Temporal's task-queue Priority and Fairness
// dispatch on every activity it schedules onto this queue.
const TaskQueue = "patterns-priority-fairness"

// Signal names accepted by HelpdeskRunWorkflow.
const (
	SignalBurstAll = "burst-all-tenants"
	SignalInjectP0 = "inject-p0-incident"
)

// BurstPerTenant is how many P2 tickets the burst-all-tenants signal appends
// to *each* tenant's queue at once. Equal volume across tiers isolates the
// fairness mechanism: with fairness off the matching service drains FIFO at
// equal priority, so all tenants progress together; with fairness on the
// 10/3/1 weights produce a clean proportional drain (Mission Critical first,
// Business last), making the SLA-by-weight story visually unambiguous.
const BurstPerTenant = 15

// MaxConcurrentActivities caps the worker's activity slot count. With 4 slots
// and many backlogged tickets, Temporal's task queue dispatches according to
// the Priority + Fairness on each activity.
const MaxConcurrentActivities = 4

// Tenant is one of the three multi-tenant helpdesk customers. Tenant ids are
// stable strings shared with the frontend.
type Tenant string

// Tenant identifiers — must match the frontend's TenantId union.
const (
	TenantAcme  Tenant = "acme"
	TenantBrick Tenant = "brick"
	TenantSolo  Tenant = "solo"
)

// PriorityKey is 1..4 — lower = higher priority. P0=1 .. P3=4.
type PriorityKey int

// TenantWeight is the fairness weight used as FairnessWeight on the activity.
// Mirrors the contract tier in the UI: Mission Critical / Enterprise / Business.
var TenantWeight = map[Tenant]float32{
	TenantAcme:  10,
	TenantBrick: 3,
	TenantSolo:  1,
}

// Ticket is a single helpdesk ticket queued behind a tenant.
type Ticket struct {
	ID       string      `json:"id"`
	Tenant   Tenant      `json:"tenantId"`
	Priority PriorityKey `json:"priorityKey"`
}

// HelpdeskInput is the input to HelpdeskRunWorkflow.
type HelpdeskInput struct {
	FairnessOn bool `json:"fairnessOn"`
}

// ResolveTicketActivityInput is the input to the ResolveTicket activity. It
// carries the parent workflow's id and run id so the activity can publish its
// business events onto the parent's NATS subject (the only subject the
// frontend SSE endpoint subscribes to). Without this the events would land on
// the child's subject and the UI would never see them.
type ResolveTicketActivityInput struct {
	Ticket           Ticket `json:"ticket"`
	ParentWorkflowID string `json:"parentWorkflowId"`
	ParentRunID      string `json:"parentRunId"`
}

// AnnounceSeedInput is the input to the announce-run-seeded activity, which
// publishes the initial per-tenant queues to the UI. The map keys are
// stringified tenant ids so the JSON payload matches the frontend types
// directly.
type AnnounceSeedInput struct {
	FairnessOn bool                `json:"fairnessOn"`
	Tenants    map[Tenant][]Ticket `json:"tenants"`
}

// AnnounceBurstInput is the input to the announce-burst-executed activity.
// Tenants maps each tenant id to the tickets the burst just appended to that
// tenant's queue.
type AnnounceBurstInput struct {
	Tenants map[Tenant][]Ticket `json:"tenants"`
}

// AnnounceIncidentInput is the input to the announce-incident-injected activity.
type AnnounceIncidentInput struct {
	TenantID Tenant `json:"tenantId"`
	Ticket   Ticket `json:"ticket"`
}
