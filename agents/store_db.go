package agents

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/zhenruyan/postgrebase/core"
	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/tools/security"
	"github.com/zhenruyan/postgrebase/tools/types"
)

// sessionBackend is the storage surface used by the agent service. It is
// implemented by both the in-memory SessionStore (used in tests) and the
// DB-backed dbSessionStore (used in production).
type sessionBackend interface {
	Create(project, name, provider, model string) *Session
	List(project string) []*Session
	Get(id string) (*Session, error)
	Messages(id string) ([]SessionMessage, error)
	AddMessage(id, role, content string) (*Session, []SessionMessage, error)
	AddMessageWithImages(id, role, content string, images []SessionImage) (*Session, []SessionMessage, error)
	NeedsAutoName(id string) bool
	SetGeneratedName(id, name string) (*Session, error)
	Rename(id, name string) (*Session, error)
}

var _ sessionBackend = (*SessionStore)(nil)
var _ sessionBackend = (*dbSessionStore)(nil)

// dbSessionStore is a database-backed session store (proposal §9 persistence).
type dbSessionStore struct {
	app core.App
}

// NewDBSessionStore creates a DB-backed session store.
func NewDBSessionStore(app core.App) *dbSessionStore {
	return &dbSessionStore{app: app}
}

// newSessionId returns a new opaque session id.
func newSessionId() string {
	return "as_" + security.RandomString(24)
}

// modelToSession maps a persisted model to the API session view.
func modelToSession(m *models.AgentSession) *Session {
	return &Session{
		Id:          m.Id,
		Project:     m.ProjectID,
		Name:        m.Name,
		Provider:    m.Provider,
		Model:       m.Model,
		Created:     m.Created,
		Updated:     m.Updated,
		LastMessage: m.LastMessage,
		NameLocked:  m.NameLocked,
	}
}

func (s *dbSessionStore) Create(project, name, provider, model string) *Session {
	id := newSessionId()
	trimmed := strings.TrimSpace(name)
	locked := trimmed != ""
	if trimmed == "" {
		trimmed = "session-" + id[len(id)-6:]
	}

	record := &models.AgentSession{
		ProjectID:   project,
		Name:        trimmed,
		Provider:    provider,
		Model:       model,
		NameLocked:  locked,
		LastMessage: "",
	}
	record.SetId(id)

	if err := s.app.Dao().SaveAgentSession(record); err != nil {
		log.Printf("agents: failed to persist session: %v", err)
	}
	return modelToSession(record)
}

func (s *dbSessionStore) List(project string) []*Session {
	records, err := s.app.Dao().FindAgentSessionsByProject(project)
	if err != nil {
		log.Printf("agents: failed to list sessions: %v", err)
		return nil
	}
	result := make([]*Session, 0, len(records))
	for _, r := range records {
		result = append(result, modelToSession(r))
	}
	return result
}

func (s *dbSessionStore) Get(id string) (*Session, error) {
	record, err := s.app.Dao().FindAgentSessionById(id)
	if err != nil {
		return nil, errors.New("agent session not found")
	}
	return modelToSession(record), nil
}

func (s *dbSessionStore) Messages(id string) ([]SessionMessage, error) {
	if _, err := s.app.Dao().FindAgentSessionById(id); err != nil {
		return nil, errors.New("agent session not found")
	}
	records, err := s.app.Dao().FindAgentMessagesBySession(id)
	if err != nil {
		return nil, err
	}
	result := make([]SessionMessage, 0, len(records))
	for _, r := range records {
		result = append(result, modelToMessage(r))
	}
	return result, nil
}

// modelToMessage maps a persisted message to the API view.
func modelToMessage(m *models.AgentMessage) SessionMessage {
	msg := SessionMessage{
		Role:    m.Role,
		Content: m.Content,
		Created: m.Created,
	}
	if len(m.Images) > 0 {
		var images []SessionImage
		if err := json.Unmarshal(m.Images, &images); err == nil {
			msg.Images = images
		}
	}
	return msg
}

func (s *dbSessionStore) AddMessage(id, role, content string) (*Session, []SessionMessage, error) {
	return s.AddMessageWithImages(id, role, content, nil)
}

func (s *dbSessionStore) AddMessageWithImages(id, role, content string, images []SessionImage) (*Session, []SessionMessage, error) {
	record, err := s.app.Dao().FindAgentSessionById(id)
	if err != nil {
		return nil, nil, errors.New("agent session not found")
	}

	trimmed := strings.TrimSpace(content)
	msg := &models.AgentMessage{
		SessionID: id,
		Role:      role,
		Content:   trimmed,
	}
	if len(images) > 0 {
		if raw, mErr := json.Marshal(images); mErr == nil {
			msg.Images = types.JsonRaw(raw)
		}
	}
	if err := s.app.Dao().SaveAgentMessage(msg); err != nil {
		return nil, nil, err
	}

	record.LastMessage = trimmed
	record.RefreshUpdated()
	if err := s.app.Dao().SaveAgentSession(record); err != nil {
		return nil, nil, err
	}

	msgs, err := s.Messages(id)
	if err != nil {
		return nil, nil, err
	}
	return modelToSession(record), msgs, nil
}

func (s *dbSessionStore) NeedsAutoName(id string) bool {
	record, err := s.app.Dao().FindAgentSessionById(id)
	if err != nil || record.NameLocked || !isPlaceholderName(record.Name) {
		return false
	}
	messages, err := s.app.Dao().FindAgentMessagesBySession(id)
	if err != nil {
		return false
	}
	for _, m := range messages {
		if m.Role == "user" {
			return true
		}
	}
	return false
}

func (s *dbSessionStore) SetGeneratedName(id, name string) (*Session, error) {
	record, err := s.app.Dao().FindAgentSessionById(id)
	if err != nil {
		return nil, errors.New("agent session not found")
	}
	if record.NameLocked || !isPlaceholderName(record.Name) {
		return modelToSession(record), nil
	}
	name = strings.TrimSpace(name)
	if name != "" {
		record.Name = name
		record.RefreshUpdated()
		if err := s.app.Dao().SaveAgentSession(record); err != nil {
			return nil, err
		}
	}
	return modelToSession(record), nil
}

func (s *dbSessionStore) Rename(id, name string) (*Session, error) {
	record, err := s.app.Dao().FindAgentSessionById(id)
	if err != nil {
		return nil, errors.New("agent session not found")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	record.Name = name
	record.NameLocked = true
	record.RefreshUpdated()
	if err := s.app.Dao().SaveAgentSession(record); err != nil {
		return nil, err
	}
	return modelToSession(record), nil
}
