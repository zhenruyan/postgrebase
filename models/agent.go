package models

import "github.com/zhenruyan/postgrebase/tools/types"

var _ Model = (*AgentSession)(nil)

// AgentSession stores a persisted project-scoped agent session (proposal §9).
type AgentSession struct {
	BaseModel

	ProjectID   string `db:"project_id" json:"project"`
	Name        string `db:"name" json:"name"`
	Provider    string `db:"provider" json:"provider"`
	Model       string `db:"model" json:"model"`
	NameLocked  bool   `db:"name_locked" json:"-"`
	LastMessage string `db:"last_message" json:"lastMessage"`
}

// TableName returns the agent session SQL table name.
func (m *AgentSession) TableName() string {
	return "_pb_agent_sessions_"
}

var _ Model = (*AgentMessage)(nil)

// AgentMessage stores a single persisted conversation item.
type AgentMessage struct {
	BaseModel

	SessionID string        `db:"session_id" json:"sessionId"`
	Role      string        `db:"role" json:"role"`
	Content   string        `db:"content" json:"content"`
	Images    types.JsonRaw `db:"images" json:"images"`
}

// TableName returns the agent message SQL table name.
func (m *AgentMessage) TableName() string {
	return "_pb_agent_messages_"
}

var _ Model = (*AgentAuditRecord)(nil)

// AgentAuditRecord stores a persisted tool authorization/execution audit entry
// (proposal §8.2). Persisted records also enable audit replay (§8.4).
type AgentAuditRecord struct {
	BaseModel

	SessionID     string `db:"session_id" json:"sessionId"`
	ProjectID     string `db:"project_id" json:"project"`
	Actor         string `db:"actor" json:"actor"`
	Tool          string `db:"tool" json:"tool"`
	Category      string `db:"category" json:"category"`
	Risk          string `db:"risk" json:"risk"`
	AuditCategory string `db:"audit_category" json:"auditCategory"`
	Decision      string `db:"decision" json:"decision"`
	Reason        string `db:"reason" json:"reason"`
	Status        string `db:"status" json:"status"`
	ErrorMsg      string `db:"error_msg" json:"error"`
}

// TableName returns the agent audit SQL table name.
func (m *AgentAuditRecord) TableName() string {
	return "_pb_agent_audit_"
}

var _ Model = (*AgentProjectConfig)(nil)

// AgentProjectConfig stores per-project agent overrides (proposal §9.1).
// Empty/inherit values fall back to the global AgentConfig in settings.
type AgentProjectConfig struct {
	BaseModel

	ProjectID       string        `db:"project_id" json:"project"`
	DefaultProvider string        `db:"default_provider" json:"defaultProvider"`
	DefaultModel    string        `db:"default_model" json:"defaultModel"`
	AllowedTools    types.JsonRaw `db:"allowed_tools" json:"allowedTools"`
	// AllowSchemaChange is one of: inherit, allow, deny.
	AllowSchemaChange string `db:"allow_schema_change" json:"allowSchemaChange"`
	// ApprovalPolicy is one of: inherit, manual, auto.
	ApprovalPolicy string `db:"approval_policy" json:"approvalPolicy"`
}

// TableName returns the agent project config SQL table name.
func (m *AgentProjectConfig) TableName() string {
	return "_pb_agent_project_configs_"
}
