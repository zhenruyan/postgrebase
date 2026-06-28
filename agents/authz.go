package agents

import (
	"encoding/json"
	"log"
	"reflect"
	"strings"

	"github.com/zhenruyan/postgrebase/tools/list"
	"github.com/zhenruyan/postgrebase/tools/types"
)

// RunOptions carries per-run authorization context (proposal §8.3).
//
// Read operations are always permitted within the project boundary. Write
// operations are denied by default and must be explicitly authorized either
// globally for the run (AllowWrites) or per tool (ApprovedTools).
type RunOptions struct {
	// AllowWrites authorizes every write/schema tool for this run.
	AllowWrites bool `json:"allowWrites"`
	// ApprovedTools is a fine-grained allow-list of dotted tool names that may
	// perform write operations for this run.
	ApprovedTools []string `json:"approvedTools"`
	// Actor identifies who initiated the run (for audit). Usually an admin id.
	Actor string `json:"actor"`
}

// authorize decides whether a tool may execute under the run options.
// It returns the decision and a human-readable reason when denied.
func (o RunOptions) authorize(spec ToolSpec) (bool, string) {
	// Read/low-risk tools are allowed by default within the project boundary.
	if spec.Category == "read" && !spec.RequiresApproval {
		return true, ""
	}

	if o.AllowWrites {
		return true, ""
	}
	if list.ExistInSlice(spec.Name, o.ApprovedTools) {
		return true, ""
	}

	return false, "write operation requires approval; re-run with allowWrites=true or include \"" + spec.Name + "\" in approvedTools"
}

// AgentAuditEntry is a structured audit record for a single tool decision
// (proposal §8.2 step 6 / §8.1 audit category).
type AgentAuditEntry struct {
	Time     types.DateTime `json:"time"`
	Session  string         `json:"session"`
	Project  string         `json:"project"`
	Actor    string         `json:"actor,omitempty"`
	Tool     string         `json:"tool"`
	Category string         `json:"category"`
	Risk     string         `json:"risk"`
	Audit    string         `json:"auditCategory"`
	Decision string         `json:"decision"` // "allow" | "deny"
	Reason   string         `json:"reason,omitempty"`
	Status   string         `json:"status,omitempty"`
	Error    string         `json:"error,omitempty"`
}

// PendingApproval describes a write operation that was blocked pending
// explicit authorization (proposal §8.3 pending state).
type PendingApproval struct {
	Tool   string         `json:"tool"`
	Risk   string         `json:"risk"`
	Args   map[string]any `json:"args,omitempty"`
	Reason string         `json:"reason"`
}

// auditSink collects audit entries and pending approvals during a run.
type auditSink struct {
	session  string
	project  string
	actor    string
	entries  []AgentAuditEntry
	pendings []PendingApproval
}

// record appends an audit entry and emits it to the process log.
func (a *auditSink) record(spec ToolSpec, decision, reason, status, errMsg string, args map[string]any) {
	if a == nil {
		return
	}
	entry := AgentAuditEntry{
		Time:     types.NowDateTime(),
		Session:  a.session,
		Project:  a.project,
		Actor:    a.actor,
		Tool:     spec.Name,
		Category: spec.Category,
		Risk:     spec.Risk,
		Audit:    spec.AuditCategory,
		Decision: decision,
		Reason:   reason,
		Status:   status,
		Error:    errMsg,
	}
	a.entries = append(a.entries, entry)

	if decision == "deny" {
		pending := PendingApproval{
			Tool:   spec.Name,
			Risk:   spec.Risk,
			Args:   redactArgs(args),
			Reason: reason,
		}
		if !a.hasPending(pending) {
			a.pendings = append(a.pendings, pending)
		}
	}

	if encoded, err := json.Marshal(entry); err == nil {
		log.Printf("AGENT_AUDIT %s", string(encoded))
	}
}

func (a *auditSink) hasPending(next PendingApproval) bool {
	for _, existing := range a.pendings {
		if existing.Tool == next.Tool && existing.Reason == next.Reason && reflect.DeepEqual(existing.Args, next.Args) {
			return true
		}
	}
	return false
}

// redactArgs strips the always-injected project key for cleaner audit output.
func redactArgs(args map[string]any) map[string]any {
	if len(args) == 0 {
		return nil
	}
	out := make(map[string]any, len(args))
	for k, v := range args {
		if k == "project" {
			continue
		}
		out[k] = v
	}
	return out
}

// trimReason keeps audit reasons compact for logs.
func trimReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if len(reason) > 240 {
		return reason[:240]
	}
	return reason
}
