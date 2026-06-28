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
	// NameLocked is true when the name was explicitly set by the user (on
	// create or via rename) and must not be auto-generated (proposal §9.2).
	NameLocked bool `json:"-"`
}

// SessionImage is an image attachment carried by a user message (proposal §6).
type SessionImage struct {
	MimeType string `json:"mimeType"`
	// Data is the base64-encoded image payload.
	Data string `json:"data"`
}

// SessionMessage represents an in-memory conversation item.
type SessionMessage struct {
	Role    string         `json:"role"`
	Content string         `json:"content"`
	Images  []SessionImage `json:"images,omitempty"`
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
	} else {
		// A user-provided name is locked and never auto-generated.
		session.NameLocked = true
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

// AddMessage appends a text message to a session.
func (s *SessionStore) AddMessage(id, role, content string) (*Session, []SessionMessage, error) {
	return s.AddMessageWithImages(id, role, content, nil)
}

// AddMessageWithImages appends a message that may carry image attachments.
func (s *SessionStore) AddMessageWithImages(id, role, content string, images []SessionImage) (*Session, []SessionMessage, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil, nil, errors.New("agent session not found")
	}

	msg := SessionMessage{
		Role:    role,
		Content: strings.TrimSpace(content),
		Images:  images,
		Created: types.NowDateTime(),
	}
	s.messages[id] = append(s.messages[id], msg)
	session.LastMessage = msg.Content
	session.Updated = types.NowDateTime()
	cp := *session
	msgs := append([]SessionMessage(nil), s.messages[id]...)
	return &cp, msgs, nil
}

// isPlaceholderName reports whether a session name is still the auto-generated
// placeholder (i.e. has not been named by the user or LLM yet).
func isPlaceholderName(name string) bool {
	return strings.HasPrefix(name, "session-")
}

// NeedsAutoName reports whether the session should receive an LLM-generated
// name: the name is not user-locked, is still a placeholder, and the session
// already has at least one user message (proposal §9.2).
func (s *SessionStore) NeedsAutoName(id string) bool {
	s.mux.RLock()
	defer s.mux.RUnlock()

	session, ok := s.sessions[id]
	if !ok || session.NameLocked || !isPlaceholderName(session.Name) {
		return false
	}
	for _, m := range s.messages[id] {
		if m.Role == "user" {
			return true
		}
	}
	return false
}

// SetGeneratedName sets an LLM-generated name once. It is a no-op if the name
// is already locked or no longer a placeholder, guaranteeing single generation.
func (s *SessionStore) SetGeneratedName(id, name string) (*Session, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil, errors.New("agent session not found")
	}
	if session.NameLocked || !isPlaceholderName(session.Name) {
		cp := *session
		return &cp, nil
	}
	name = strings.TrimSpace(name)
	if name != "" {
		session.Name = name
		session.Updated = types.NowDateTime()
	}
	cp := *session
	return &cp, nil
}

// Rename explicitly sets a user-provided name and locks it against future
// auto-generation (proposal §9.2 "除非用户显式重命名").
func (s *SessionStore) Rename(id, name string) (*Session, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil, errors.New("agent session not found")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	session.Name = name
	session.NameLocked = true
	session.Updated = types.NowDateTime()
	cp := *session
	return &cp, nil
}
