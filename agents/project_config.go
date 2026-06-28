package agents

import (
	"encoding/json"
	"strings"

	"github.com/zhenruyan/postgrebase/models"
)

// ProjectConfig is the API view of a per-project agent configuration (§9.1).
type ProjectConfig struct {
	Project           string   `json:"project"`
	DefaultProvider   string   `json:"defaultProvider"`
	DefaultModel      string   `json:"defaultModel"`
	AllowedTools      []string `json:"allowedTools"`
	AllowSchemaChange string   `json:"allowSchemaChange"` // inherit|allow|deny
	ApprovalPolicy    string   `json:"approvalPolicy"`    // inherit|manual|auto
}

// projectPolicy is the resolved effective policy for a run, after overlaying
// the per-project config over the global agent settings.
type projectPolicy struct {
	defaultProvider   string
	defaultModel      string
	allowedTools      []string
	allowSchemaChange bool
	autoApprove       bool
}

// GetProjectConfig returns the stored per-project config or an inherit default.
func (s *Service) GetProjectConfig(project string) ProjectConfig {
	cfg := ProjectConfig{
		Project:           project,
		AllowSchemaChange: "inherit",
		ApprovalPolicy:    "inherit",
		AllowedTools:      []string{},
	}
	if s == nil || s.app == nil {
		return cfg
	}

	record, err := s.app.Dao().FindAgentProjectConfig(project)
	if err != nil || record == nil {
		return cfg
	}

	cfg.DefaultProvider = record.DefaultProvider
	cfg.DefaultModel = record.DefaultModel
	if record.AllowSchemaChange != "" {
		cfg.AllowSchemaChange = record.AllowSchemaChange
	}
	if record.ApprovalPolicy != "" {
		cfg.ApprovalPolicy = record.ApprovalPolicy
	}
	if len(record.AllowedTools) > 0 {
		var tools []string
		if err := json.Unmarshal(record.AllowedTools, &tools); err == nil {
			cfg.AllowedTools = tools
		}
	}
	return cfg
}

// SaveProjectConfig persists a per-project agent config (upsert by project).
func (s *Service) SaveProjectConfig(in ProjectConfig) (ProjectConfig, error) {
	record, err := s.app.Dao().FindAgentProjectConfig(in.Project)
	if err != nil || record == nil {
		record = &models.AgentProjectConfig{ProjectID: in.Project}
	}

	record.DefaultProvider = strings.TrimSpace(in.DefaultProvider)
	record.DefaultModel = strings.TrimSpace(in.DefaultModel)
	record.AllowSchemaChange = normalizeTriState(in.AllowSchemaChange, "inherit", "allow", "deny")
	record.ApprovalPolicy = normalizeTriState(in.ApprovalPolicy, "inherit", "manual", "auto")
	if raw, mErr := json.Marshal(in.AllowedTools); mErr == nil {
		record.AllowedTools = raw
	}

	if err := s.app.Dao().SaveAgentProjectConfig(record); err != nil {
		return ProjectConfig{}, err
	}
	return s.GetProjectConfig(in.Project), nil
}

// normalizeTriState returns value if it is one of the allowed options, else def.
func normalizeTriState(value, def string, allowed ...string) string {
	value = strings.TrimSpace(value)
	for _, a := range allowed {
		if value == a {
			return value
		}
	}
	return def
}

// resolvePolicy overlays the per-project config over global agent settings.
func (s *Service) resolvePolicy(project string) projectPolicy {
	global := s.app.Settings().Agents
	cfg := s.GetProjectConfig(project)

	policy := projectPolicy{
		defaultProvider:   firstNonEmpty(cfg.DefaultProvider, global.DefaultProvider),
		defaultModel:      firstNonEmpty(cfg.DefaultModel, global.DefaultModel),
		allowedTools:      global.AllowedTools,
		allowSchemaChange: global.AllowSchemaChange,
	}

	if len(cfg.AllowedTools) > 0 {
		policy.allowedTools = cfg.AllowedTools
	}
	switch cfg.AllowSchemaChange {
	case "allow":
		policy.allowSchemaChange = true
	case "deny":
		policy.allowSchemaChange = false
	}
	if cfg.ApprovalPolicy == "auto" {
		policy.autoApprove = true
	}

	return policy
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
