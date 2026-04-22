package multiagent

// TaskQueue is the Temporal task queue used by the multi-agent deep-research worker.
const TaskQueue = "patterns-multi-agent"

// Scenario selects the scripted demo trajectory. Keeping scenarios on the
// input lets the workflow stay generic while each run tells a distinct story.
type Scenario string

const (
	// ScenarioHappy — every search succeeds, every child returns a full set.
	ScenarioHappy Scenario = "happy"
	// ScenarioPartial — a fixed pair of searches fail non-retryably so two
	// children return partial results, while the parent still produces a report.
	ScenarioPartial Scenario = "partial"
)

// Fixed subtopic names so the demo story stays predictable across runs.
var demoSubtopics = []string{
	"Job displacement",
	"New job creation",
	"Policy & regulation",
}

// Fixed query set per subtopic (2 queries each) so the UI can lay out a stable
// grid and failure injection points are deterministic.
var demoQueries = map[string][]string{
	"Job displacement": {
		"AI impact on white-collar jobs 2026",
		"automation unemployment forecasts",
	},
	"New job creation": {
		"AI-adjacent roles emerging 2026",
		"reskilling programs effectiveness",
	},
	"Policy & regulation": {
		"EU AI Act enforcement timeline",
		"US federal AI workforce policy",
	},
}

// DeepResearchRequest is the input to the parent workflow.
type DeepResearchRequest struct {
	Prompt   string   `json:"prompt"`
	Scenario Scenario `json:"scenario"`
}

// Subtopic is one unit of planned research fanned out to a child agent.
type Subtopic struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
}

// ResearchPlan is the output of PlanResearch.
type ResearchPlan struct {
	Prompt    string     `json:"prompt"`
	Scenario  Scenario   `json:"scenario"`
	Subtopics []Subtopic `json:"subtopics"`
}

// TopicQueries holds the generated queries for a single subtopic.
type TopicQueries struct {
	TopicIndex int      `json:"topicIndex"`
	TopicName  string   `json:"topicName"`
	Queries    []string `json:"queries"`
}

// ResearchQueries is the output of GenerateQueries.
type ResearchQueries struct {
	Scenario Scenario       `json:"scenario"`
	Topics   []TopicQueries `json:"topics"`
}

// ResearchAgentInput is the input to each child ResearchAgentWorkflow.
type ResearchAgentInput struct {
	// ParentWorkflowID / ParentRunID route business events emitted by child
	// activities back onto the parent's NATS subject via PublishBusinessAs.
	ParentWorkflowID string   `json:"parentWorkflowId"`
	ParentRunID      string   `json:"parentRunId"`
	Scenario         Scenario `json:"scenario"`
	TopicIndex       int      `json:"topicIndex"`
	TopicName        string   `json:"topicName"`
	Queries          []string `json:"queries"`
}

// Source is a single scripted "web page" harvested by a search.
type Source struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

// SearchInput is the input to the WebSearch activity (run inside a child).
type SearchInput struct {
	ParentWorkflowID string   `json:"parentWorkflowId"`
	ParentRunID      string   `json:"parentRunId"`
	Scenario         Scenario `json:"scenario"`
	TopicIndex       int      `json:"topicIndex"`
	TopicName        string   `json:"topicName"`
	QueryIndex       int      `json:"queryIndex"`
	Query            string   `json:"query"`
}

// SearchResult is the output of WebSearch.
type SearchResult struct {
	TopicIndex int      `json:"topicIndex"`
	QueryIndex int      `json:"queryIndex"`
	Sources    []Source `json:"sources"`
}

// ResearchResult is what each child ResearchAgentWorkflow returns.
type ResearchResult struct {
	TopicIndex int      `json:"topicIndex"`
	TopicName  string   `json:"topicName"`
	Sources    []Source `json:"sources"`
	// Partial is true when at least one query under this topic failed but the
	// child still returned at least one successful source.
	Partial bool `json:"partial"`
}

// ChildOutcomeInput is handed to RecordChildOutcome so the parent can emit a
// business event per settled child without doing I/O from workflow scope.
type ChildOutcomeInput struct {
	TopicIndex int    `json:"topicIndex"`
	TopicName  string `json:"topicName"`
	Sources    int    `json:"sources"`
	Partial    bool   `json:"partial"`
	Failed     bool   `json:"failed"`
	Error      string `json:"error,omitempty"`
}

// SynthesisInput is the input to SynthesizeReport.
type SynthesisInput struct {
	Prompt  string           `json:"prompt"`
	Results []ResearchResult `json:"results"`
}

// Report is the parent workflow's final answer.
type Report struct {
	Summary      string `json:"summary"`
	Sections     int    `json:"sections"`
	SourcesUsed  int    `json:"sourcesUsed"`
	PartialCount int    `json:"partialCount"`
}

// Progress is returned by the getProgress query handler. Parallels the
// counters exposed by agent / batch so the demo page has a uniform surface.
type Progress struct {
	Phase     string `json:"phase"`
	LLMCalls  int    `json:"llmCalls"`
	Completed bool   `json:"completed"`
}

// Phase strings reported by the getProgress query.
const (
	PhaseIdle      = "idle"
	PhasePlanning  = "planning"
	PhaseQueries   = "queries"
	PhaseResearch  = "research"
	PhaseSynthesis = "synthesis"
	PhaseCompleted = "completed"
)
