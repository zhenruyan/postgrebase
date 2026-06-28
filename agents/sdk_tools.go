package agents

import (
	"context"
	"encoding/json"
	"strings"

	agentsdk "github.com/startvibecoding/vibecoding/agent"
	"github.com/zhenruyan/postgrebase/tools/list"
)

// sdkTool adapts a PostgreBase ToolSpec + executor to the vibecoding
// agent.ExternalTool interface. It enforces both the project boundary
// (proposal §4.4) and write-operation authorization (proposal §8.3), and
// records an audit entry for every decision (proposal §8.2).
type sdkTool struct {
	spec    ToolSpec
	exec    ToolExecutor
	project string
	opts    RunOptions
	audit   *auditSink
}

// Name returns a provider-safe tool name (no dots).
func (t *sdkTool) Name() string { return toolName(t.spec.Name) }

// Description returns the tool description.
func (t *sdkTool) Description() string { return t.spec.Description }

// Parameters returns the JSON Schema for the tool's parameters.
func (t *sdkTool) Parameters() []byte {
	if t.spec.InputSchema == nil {
		return []byte(`{"type":"object","properties":{}}`)
	}
	raw, err := json.Marshal(t.spec.InputSchema)
	if err != nil {
		return []byte(`{"type":"object","properties":{}}`)
	}
	return raw
}

// Execute enforces project scope + authorization, then runs the executor.
func (t *sdkTool) Execute(_ context.Context, params map[string]any) (agentsdk.ExternalToolResult, error) {
	if params == nil {
		params = map[string]any{}
	}
	// Enforce the project boundary regardless of model-provided value.
	if t.project != "" {
		params["project"] = t.project
	}

	// Authorization gate (proposal §8.3): write operations require approval.
	if allowed, reason := t.opts.authorize(t.spec); !allowed {
		t.audit.record(t.spec, "deny", trimReason(reason), "pending", "", params)
		return agentsdk.ExternalToolResult{
			Text: `{"status":"pending_approval","message":` + quote(reason) + `}`,
		}, nil
	}

	result, err := t.exec(params)
	if err != nil {
		t.audit.record(t.spec, "allow", "", "error", err.Error(), params)
		return agentsdk.ExternalToolResult{Text: err.Error(), IsError: true}, nil
	}

	errMsg := ""
	isError := result.Status == "error"
	if isError {
		errMsg = result.Message
	}
	t.audit.record(t.spec, "allow", "", result.Status, errMsg, params)

	encoded, mErr := json.Marshal(result)
	if mErr != nil {
		return agentsdk.ExternalToolResult{Text: result.Status}, nil
	}
	return agentsdk.ExternalToolResult{Text: string(encoded)}, nil
}

// PromptSnippet implements agentsdk.ExternalToolPromptInfo.
func (t *sdkTool) PromptSnippet() string {
	return t.spec.Name + " — " + t.spec.Description
}

// PromptGuidelines implements agentsdk.ExternalToolPromptInfo.
func (t *sdkTool) PromptGuidelines() []string { return nil }

// quote returns a JSON-quoted string.
func quote(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}

// externalTools builds the agent.ExternalTool set for a project, honoring the
// effective per-run policy (allowed tools + schema-change permission resolved
// from project config overlaid on global settings). Every tool is bound to the
// given project, run options and audit sink.
func (s *Service) externalTools(project string, policy projectPolicy, opts RunOptions, audit *auditSink) []agentsdk.ExternalTool {
	specs := s.tools.List()

	result := make([]agentsdk.ExternalTool, 0, len(specs))
	for _, spec := range specs {
		if len(policy.allowedTools) > 0 && !list.ExistInSlice(spec.Name, policy.allowedTools) {
			continue
		}
		if spec.Category == "write" && strings.HasPrefix(spec.Name, "schema.") && !policy.allowSchemaChange {
			continue
		}
		exec, ok := s.tools.executor(spec.Name)
		if !ok || exec == nil {
			continue
		}
		result = append(result, &sdkTool{
			spec:    spec,
			exec:    exec,
			project: project,
			opts:    opts,
			audit:   audit,
		})
	}

	return result
}
