package agents

import (
	"context"
	"strings"
	"testing"
)

func TestApplyToolMetadata(t *testing.T) {
	cases := []struct {
		name             string
		wantCategory     string
		wantRequires     bool
		wantAuditNonZero bool
	}{
		{"data.query", "read", false, true},
		{"data.get", "read", false, true},
		{"dataset.preview", "read", false, true},
		{"data.insert", "write", true, true},
		{"data.delete", "write", true, true},
		{"schema.create_table", "write", true, true},
		{"schema.drop_field", "write", true, true},
		{"some.unknown_tool", "write", true, true}, // conservative default
	}

	for _, tc := range cases {
		spec := ToolSpec{Name: tc.name}
		applyToolMetadata(&spec)
		if spec.Category != tc.wantCategory {
			t.Errorf("%s: category = %q, want %q", tc.name, spec.Category, tc.wantCategory)
		}
		if spec.RequiresApproval != tc.wantRequires {
			t.Errorf("%s: requiresApproval = %v, want %v", tc.name, spec.RequiresApproval, tc.wantRequires)
		}
		if tc.wantAuditNonZero && spec.AuditCategory == "" {
			t.Errorf("%s: auditCategory should not be empty", tc.name)
		}
		if spec.Risk == "" {
			t.Errorf("%s: risk should not be empty", tc.name)
		}
	}
}

func TestRunOptionsAuthorize(t *testing.T) {
	readTool := ToolSpec{Name: "data.query", Category: "read"}
	writeTool := ToolSpec{Name: "data.insert", Category: "write", RequiresApproval: true}

	// Read tools are always allowed within the project boundary.
	if ok, _ := (RunOptions{}).authorize(readTool); !ok {
		t.Fatal("read tool should be allowed by default")
	}

	// Write tools are denied by default (pending approval).
	if ok, reason := (RunOptions{}).authorize(writeTool); ok || reason == "" {
		t.Fatal("write tool should be denied by default with a reason")
	}

	// Global write authorization allows write tools.
	if ok, _ := (RunOptions{AllowWrites: true}).authorize(writeTool); !ok {
		t.Fatal("write tool should be allowed when AllowWrites is set")
	}

	// Fine-grained approval allows a specific write tool.
	if ok, _ := (RunOptions{ApprovedTools: []string{"data.insert"}}).authorize(writeTool); !ok {
		t.Fatal("write tool should be allowed when present in ApprovedTools")
	}

	// Fine-grained approval for a different tool does not leak.
	if ok, _ := (RunOptions{ApprovedTools: []string{"data.update"}}).authorize(writeTool); ok {
		t.Fatal("approval for a different tool must not authorize this tool")
	}
}

func TestAuditSinkRecordsPendingOnDeny(t *testing.T) {
	sink := &auditSink{session: "s1", project: "p1"}
	spec := ToolSpec{Name: "data.delete", Category: "write", Risk: "high", AuditCategory: "data", RequiresApproval: true}

	sink.record(spec, "deny", "needs approval", "pending", "", map[string]any{"project": "p1", "id": "x"})

	if len(sink.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(sink.entries))
	}
	if len(sink.pendings) != 1 {
		t.Fatalf("expected 1 pending approval, got %d", len(sink.pendings))
	}
	if _, ok := sink.pendings[0].Args["project"]; ok {
		t.Error("pending approval args should redact the injected project key")
	}
	if sink.pendings[0].Tool != "data.delete" {
		t.Errorf("pending tool = %q, want data.delete", sink.pendings[0].Tool)
	}
}

func TestAuditSinkDeduplicatesPendingApprovals(t *testing.T) {
	sink := &auditSink{session: "s1", project: "p1"}
	spec := ToolSpec{Name: "data.insert", Category: "write", Risk: "medium", AuditCategory: "data", RequiresApproval: true}
	args := map[string]any{"project": "p1", "collection": "posts"}

	sink.record(spec, "deny", "needs approval", "pending", "", args)
	sink.record(spec, "deny", "needs approval", "pending", "", args)

	if len(sink.entries) != 2 {
		t.Fatalf("expected both audit entries to be kept, got %d", len(sink.entries))
	}
	if len(sink.pendings) != 1 {
		t.Fatalf("expected duplicate pending approvals to be collapsed, got %d", len(sink.pendings))
	}
}

func TestSDKToolPendingApprovalReturnsStructuredOutput(t *testing.T) {
	sink := &auditSink{session: "s1", project: "p1"}
	tool := &sdkTool{
		spec: ToolSpec{
			Name:             "data.insert",
			Category:         "write",
			Risk:             "medium",
			AuditCategory:    "data",
			RequiresApproval: true,
		},
		project: "p1",
		opts:    RunOptions{},
		audit:   sink,
	}

	result, err := tool.Execute(context.Background(), map[string]any{"collection": "posts"})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("business-level pending approval should remain a structured tool result, not a transport error")
	}
	if !strings.Contains(result.Text, `"status":"pending_approval"`) {
		t.Fatalf("unexpected result text: %s", result.Text)
	}
	if len(sink.pendings) != 1 || sink.pendings[0].Tool != "data.insert" {
		t.Fatalf("expected pending approval to be recorded, got %#v", sink.pendings)
	}
}

func TestSDKToolBusinessErrorReturnsStructuredOutput(t *testing.T) {
	sink := &auditSink{session: "s1", project: "p1"}
	tool := &sdkTool{
		spec: ToolSpec{
			Name:          "data.query",
			Category:      "read",
			Risk:          "low",
			AuditCategory: "data",
		},
		exec: func(args map[string]any) (*ToolExecutionResult, error) {
			return &ToolExecutionResult{Status: "error", Message: "missing table"}, nil
		},
		project: "p1",
		audit:   sink,
	}

	result, err := tool.Execute(context.Background(), map[string]any{"collection": "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("business-level tool status=error should remain readable by the model")
	}
	if !strings.Contains(result.Text, `"status":"error"`) || !strings.Contains(result.Text, `"message":"missing table"`) {
		t.Fatalf("unexpected result text: %s", result.Text)
	}
	if len(sink.entries) != 1 || sink.entries[0].Status != "error" || sink.entries[0].Error != "missing table" {
		t.Fatalf("expected audit error to be kept, got %#v", sink.entries)
	}
}
