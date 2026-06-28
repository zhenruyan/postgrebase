package agents

import (
	"errors"
	"os"
	"strings"

	// Registers the vibecoding agent builder + provider resolution hooks.
	_ "github.com/startvibecoding/vibecoding/bootstrap"
	"github.com/zhenruyan/postgrebase/core"
	"github.com/zhenruyan/postgrebase/models"
)

// resolveApiKey resolves an api key value, supporting the "env:NAME" form to
// read the key from the process environment.
func resolveApiKey(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "env:") {
		return strings.TrimSpace(os.Getenv(strings.TrimPrefix(value, "env:")))
	}
	return value
}

// Service exposes embedded agent runtime views derived from app settings.
type Service struct {
	app      core.App
	registry *Registry
	sessions sessionBackend
	tools    *ToolRegistry
}

// NewService creates a new agent service.
func NewService(app core.App) *Service {
	svc := &Service{
		app:      app,
		registry: NewRegistry(app.Settings().Agents),
		sessions: NewDBSessionStore(app),
		tools:    NewToolRegistry(),
	}

	svc.RegisterExecutors()

	return svc
}

// Refresh reloads the service registry from current app settings.
func (s *Service) Refresh() {
	s.registry = NewRegistry(s.app.Settings().Agents)
}

// Runtime returns the current runtime snapshot.
func (s *Service) Runtime() Runtime {
	if s == nil || s.registry == nil {
		return Runtime{}
	}

	return s.registry.Snapshot()
}

// Providers returns all configured providers.
func (s *Service) Providers() []Provider {
	return s.Runtime().Providers
}

// Models returns all configured models across providers.
func (s *Service) Models() []Model {
	runtime := s.Runtime()
	result := make([]Model, 0)
	for _, provider := range runtime.Providers {
		result = append(result, provider.Models...)
	}
	return result
}

// Tools returns the static tool registry.
func (s *Service) Tools() []ToolSpec {
	if s == nil || s.tools == nil {
		return nil
	}
	return s.tools.List()
}

// CreateSession creates a new in-memory project session.
func (s *Service) CreateSession(project, name, provider, model string) *Session {
	if s == nil || s.sessions == nil {
		return nil
	}
	return s.sessions.Create(project, name, provider, model)
}

// ListSessions returns all sessions for a project.
func (s *Service) ListSessions(project string) []*Session {
	if s == nil || s.sessions == nil {
		return nil
	}
	return s.sessions.List(project)
}

// GetSession returns a session by id.
func (s *Service) GetSession(id string) (*Session, error) {
	if s == nil || s.sessions == nil {
		return nil, errors.New("agent sessions are not available")
	}
	return s.sessions.Get(id)
}

// RenameSession explicitly renames a session and locks the name (proposal §9.2).
func (s *Service) RenameSession(id, name string) (*Session, error) {
	if s == nil || s.sessions == nil {
		return nil, errors.New("agent sessions are not available")
	}
	return s.sessions.Rename(id, name)
}

// SessionAudit returns the persisted audit trail for a session (proposal §8.2/§8.4).
func (s *Service) SessionAudit(id string) ([]*models.AgentAuditRecord, error) {
	if s == nil || s.app == nil {
		return nil, errors.New("agent audit is not available")
	}
	return s.app.Dao().FindAgentAuditBySession(id)
}

// AppendMessage adds a message to a session.
func (s *Service) AppendMessage(id, role, content string) (*Session, []SessionMessage, error) {
	if s == nil || s.sessions == nil {
		return nil, nil, errors.New("agent sessions are not available")
	}
	return s.sessions.AddMessage(id, role, content)
}

// SessionMessages returns all stored messages for a session.
func (s *Service) SessionMessages(id string) ([]SessionMessage, error) {
	if s == nil || s.sessions == nil {
		return nil, errors.New("agent sessions are not available")
	}
	return s.sessions.Messages(id)
}

// ExecuteTool runs a tool call using the configured registry.
func (s *Service) ExecuteTool(name string, args map[string]any) (*ToolExecutionResult, error) {
	if s == nil || s.tools == nil {
		return nil, errors.New("agent tools are not available")
	}
	return s.tools.Execute(name, args)
}

// RegisterExecutors wires context-aware tool executors.
func (s *Service) RegisterExecutors() {
	if s == nil || s.tools == nil || s.app == nil {
		return
	}

	s.tools.SetExecutor("data.query", NewQueryExecutor(s.app))
	s.tools.SetExecutor("data.insert", NewInsertRecordExecutor(s.app))
	s.tools.SetExecutor("data.update", NewUpdateRecordExecutor(s.app))
	s.tools.SetExecutor("data.delete", NewDeleteRecordExecutor(s.app))
	s.tools.SetExecutor("dataset.preview", NewDatasetPreviewExecutor(s.app))
	s.tools.SetExecutor("schema.create_table", NewCreateTableExecutor(s.app))
	s.tools.SetExecutor("schema.create_index", NewCreateIndexExecutor(s.app))
	s.tools.SetExecutor("schema.add_field", NewAddFieldExecutor(s.app))
	s.tools.SetExecutor("schema.update_field", NewUpdateFieldExecutor(s.app))
	s.tools.SetExecutor("schema.drop_field", NewDropFieldExecutor(s.app))
	s.tools.SetExecutor("schema.set_relation", NewSetRelationExecutor(s.app))
}

// ExecuteToolInSession runs a tool call and stores a trace in the session history.
func (s *Service) ExecuteToolInSession(sessionID, name string, args map[string]any) (*ToolExecutionResult, error) {
	if s == nil || s.tools == nil || s.sessions == nil {
		return nil, errors.New("agent tools are not available")
	}

	result, err := s.tools.Execute(name, args)
	if err != nil {
		return nil, err
	}

	if sessionID != "" {
		_, _, _ = s.sessions.AddMessage(sessionID, "tool", name)
	}

	return result, nil
}
