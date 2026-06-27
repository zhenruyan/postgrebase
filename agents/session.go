package agents

import (
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/zhenruyan/postgrebase/tools/security"
	"github.com/zhenruyan/postgrebase/tools/types"
)

// Session represents a project-scoped agent session.
type Session struct {
	Id          string         `json:"id"`
	Project     string         `json:"project"`
	Name        string         `json:"name"`
	Model       string         `json:"model"`
	Provider    string         `json:"provider"`
	Created     types.DateTime `json:"created"`
	Updated     types.DateTime `json:"updated"`
	LastMessage string         `json:"lastMessage"`
}

// SessionMessage represents an in-memory conversation item.
type SessionMessage struct {
	Role    string         `json:"role"`
	Content string         `json:"content"`
	Created types.DateTime `json:"created"`
}

// SessionStore holds in-memory sessions for the embedded agent surface.
type SessionStore struct {
	mux      sync.RWMutex
	sessions map[string]*Session
	messages map[string][]SessionMessage
}

// NewSessionStore creates a new in-memory session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: map[string]*Session{},
		messages: map[string][]SessionMessage{},
	}
}

// Create creates a new session for a project.
func (s *SessionStore) Create(project, name, provider, model string) *Session {
	s.mux.Lock()
	defer s.mux.Unlock()

	now := types.NowDateTime()
	session := &Session{
		Id:       "as_" + security.RandomString(24),
		Project:  project,
		Name:     strings.TrimSpace(name),
		Provider: provider,
		Model:    model,
		Created:  now,
		Updated:  now,
	}
	if session.Name == "" {
		session.Name = "session-" + session.Id[len(session.Id)-6:]
	}

	s.sessions[session.Id] = session
	return session
}

// List returns sessions sorted by newest first.
func (s *SessionStore) List(project string) []*Session {
	s.mux.RLock()
	defer s.mux.RUnlock()

	result := make([]*Session, 0)
	for _, session := range s.sessions {
		if project != "" && session.Project != project {
			continue
		}
		cp := *session
		result = append(result, &cp)
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Updated.Time().After(result[j].Updated.Time())
	})

	return result
}

// Get returns a session by id.
func (s *SessionStore) Get(id string) (*Session, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil, errors.New("agent session not found")
	}

	cp := *session
	return &cp, nil
}

// Messages returns the stored messages for a session.
func (s *SessionStore) Messages(id string) ([]SessionMessage, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	if _, ok := s.sessions[id]; !ok {
		return nil, errors.New("agent session not found")
	}

	result := append([]SessionMessage(nil), s.messages[id]...)
	return result, nil
}

// AddMessage appends a message to a session.
func (s *SessionStore) AddMessage(id, role, content string) (*Session, []SessionMessage, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil, nil, errors.New("agent session not found")
	}

	msg := SessionMessage{
		Role:    role,
		Content: strings.TrimSpace(content),
		Created: types.NowDateTime(),
	}
	s.messages[id] = append(s.messages[id], msg)
	session.LastMessage = msg.Content
	session.Updated = types.NowDateTime()
	cp := *session
	msgs := append([]SessionMessage(nil), s.messages[id]...)
	return &cp, msgs, nil
}
