package priorityfairness

// TaskQueue is the Temporal task queue used by the priority-fairness pattern
// worker. The worker opts into Temporal's task-queue Priority and Fairness
// dispatch on every activity it schedules onto this queue.
const TaskQueue = "patterns-priority-fairness"

// Signal names accepted by HelpdeskRunWorkflow.
const (
	SignalInjectP0 = "inject-p0-incident"
)

// MaxConcurrentActivities caps the worker's activity slot count. With 4 slots
// and many backlogged tickets, Temporal's task queue dispatches according to
// the Priority + Fairness on each activity.
const MaxConcurrentActivities = 4

// Tenant is one of the three multi-tenant helpdesk customers. Tenant ids are
// stable strings shared with the frontend.
type Tenant string

// Tenant identifiers — must match the frontend's TenantId union.
const (
	TenantMissionCritical Tenant = "mission-critical"
	TenantEnterprise      Tenant = "enterprise"
	TenantBusiness        Tenant = "business"
)

// PriorityKey is 1..4 — lower = higher priority. P0=1 .. P3=4.
type PriorityKey int

// TenantWeight is the fairness weight used as FairnessWeight on the activity.
// Mirrors the contract tier in the UI: Mission Critical / Enterprise / Business.
var TenantWeight = map[Tenant]float32{
	TenantMissionCritical: 10,
	TenantEnterprise:      3,
	TenantBusiness:        1,
}

// Ticket is a single helpdesk ticket queued behind a tenant.
type Ticket struct {
	ID       string      `json:"id"`
	Tenant   Tenant      `json:"tenant"`
	Priority PriorityKey `json:"priority"`
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

// AnnounceIncidentInput is the input to the announce-incident-injected activity.
type AnnounceIncidentInput struct {
	TenantID Tenant `json:"tenant"`
	Ticket   Ticket `json:"ticket"`
}
