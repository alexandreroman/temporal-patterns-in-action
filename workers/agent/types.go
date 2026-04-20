package agent

// TaskQueue is the Temporal task queue used by the durable AI agent worker.
const TaskQueue = "patterns-agent"

// SignalApproval is the signal name used to deliver a human-in-the-loop
// approval decision to a waiting agent workflow.
const SignalApproval = "approval"

// Scenario selects the scripted demo trajectory the simulated LLM follows.
// Keeping the scenario on the input (rather than in the activity) lets the
// workflow stay generic while each run still tells a distinct story.
type Scenario string

const (
	ScenarioHappy    Scenario = "happy"
	ScenarioRetry    Scenario = "retry"
	ScenarioApproval Scenario = "approval"
)

// Role is the author of a conversation message.
type Role string

const (
	RoleUser   Role = "user"
	RoleLLM    Role = "llm"
	RoleTool   Role = "tool"
	RoleSystem Role = "system"
)

// UserRequest is the input to the agent workflow.
type UserRequest struct {
	Prompt   string   `json:"prompt"`
	Scenario Scenario `json:"scenario"`
}

// Message is a single entry in the agent's conversation history.
type Message struct {
	Role     Role   `json:"role"`
	Content  string `json:"content"`
	ToolName string `json:"toolName,omitempty"`
}

// LLMRequest is what the workflow hands to the CallLLM activity on every
// iteration. The full history is resent each time so the activity is
// stateless — Temporal already retains the authoritative conversation in
// workflow state.
type LLMRequest struct {
	Scenario Scenario  `json:"scenario"`
	Loop     int       `json:"loop"`
	History  []Message `json:"history"`
}

// ToolCall describes an MCP tool invocation the LLM wants to make.
type ToolCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

// LLMResponse is the decision the LLM returns on every iteration: a chat
// message plus exactly one of ToolCall / NeedsApproval / Plan (or none, which
// is treated as a soft loop continuation).
type LLMResponse struct {
	Message       Message   `json:"message"`
	Tokens        int       `json:"tokens"`
	ToolCall      *ToolCall `json:"toolCall,omitempty"`
	NeedsApproval bool      `json:"needsApproval,omitempty"`
	Plan          *Plan     `json:"plan,omitempty"`
}

// ToolResult is the output of an MCP tool execution.
type ToolResult struct {
	Name   string `json:"name"`
	Output string `json:"output"`
}

// Plan is the agent's final answer.
type Plan struct {
	Summary string `json:"summary"`
}

// ApprovalDecision is the signal payload delivered on SignalApproval.
type ApprovalDecision struct {
	Approved bool `json:"approved"`
}

// Progress is returned by the getProgress query handler.
type Progress struct {
	Loop      int    `json:"loop"`
	Phase     string `json:"phase"`
	LLMCalls  int    `json:"llmCalls"`
	ToolCalls int    `json:"toolCalls"`
	Tokens    int    `json:"tokens"`
	Completed bool   `json:"completed"`
}
