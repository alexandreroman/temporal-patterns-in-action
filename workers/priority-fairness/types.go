package priorityfairness

// TaskQueue is the Temporal task queue used by the priority-fairness pattern
// worker. The worker opts into Temporal's task-queue Priority and Fairness
// dispatch on every activity it schedules onto this queue.
const TaskQueue = "patterns-priority-fairness"

// Signal names accepted by HelpdeskRunWorkflow.
const (
	SignalAcmeDump80 = "acme-dump-80"
	SignalInjectP0   = "inject-p0-incident"
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
	TenantAcme  Tenant = "acme"
	TenantBrick Tenant = "brick"
	TenantSolo  Tenant = "solo"
)

// PriorityKey is 1..4 — lower = higher priority. P0=1 .. P3=4.
type PriorityKey int

// TenantWeight is the fairness weight used as FairnessWeight on the activity.
// Mirrors the contract tier in the UI: Enterprise / Pro / Free.
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

// AnnounceSeedInput is the input to the announce-run-seeded activity, which
// publishes the initial per-tenant queues to the UI. The map keys are
// stringified tenant ids so the JSON payload matches the frontend types
// directly.
type AnnounceSeedInput struct {
	FairnessOn bool                `json:"fairnessOn"`
	Tenants    map[Tenant][]Ticket `json:"tenants"`
}

// AnnounceDumpInput is the input to the announce-dump-executed activity.
type AnnounceDumpInput struct {
	TenantID Tenant   `json:"tenantId"`
	Tickets  []Ticket `json:"tickets"`
}

// AnnounceIncidentInput is the input to the announce-incident-injected activity.
type AnnounceIncidentInput struct {
	TenantID Tenant `json:"tenantId"`
	Ticket   Ticket `json:"ticket"`
}
