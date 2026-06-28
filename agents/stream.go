package agents

// RunStreamEventType identifies a streamed agent event.
type RunStreamEventType string

const (
	RunStreamEventStart      RunStreamEventType = "start"
	RunStreamEventTextDelta  RunStreamEventType = "text_delta"
	RunStreamEventThinkDelta RunStreamEventType = "think_delta"
	RunStreamEventToolCall   RunStreamEventType = "tool_call"
	RunStreamEventToolResult RunStreamEventType = "tool_result"
	RunStreamEventStatus     RunStreamEventType = "status"
	RunStreamEventFinal      RunStreamEventType = "final"
	RunStreamEventError      RunStreamEventType = "error"
)

// RunStreamHandler receives progress events for a single agent run.
// Returning false stops the run, usually because the HTTP client disconnected.
type RunStreamHandler func(RunStreamEvent) bool

// RunStreamEvent is emitted while an agent run is still in progress.
type RunStreamEvent struct {
	Type             RunStreamEventType `json:"type"`
	SessionId        string             `json:"sessionId,omitempty"`
	Provider         string             `json:"provider,omitempty"`
	Model            string             `json:"model,omitempty"`
	Tool             string             `json:"tool,omitempty"`
	Args             map[string]any     `json:"args,omitempty"`
	Text             string             `json:"text,omitempty"`
	Thought          string             `json:"thought,omitempty"`
	Status           string             `json:"status,omitempty"`
	Trace            *RunTrace          `json:"trace,omitempty"`
	PendingApprovals []PendingApproval  `json:"pendingApprovals,omitempty"`
	Result           *RunResult         `json:"result,omitempty"`
	Error            string             `json:"error,omitempty"`
}
