package vector

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zhenruyan/postgrebase/models"
)

// Mode describes the current vector runtime topology.
type Mode string

const (
	ModeStandalone Mode = "standalone"
	ModeCluster    Mode = "cluster"
)

// Status describes the current vector runtime state.
type Status struct {
	Enabled           bool      `json:"enabled"`
	Mode              Mode      `json:"mode"`
	NodeID            string    `json:"nodeId"`
	DataDriver        string    `json:"dataDriver,omitempty"`
	RedisEnabled      bool      `json:"redisEnabled"`
	Backend           string    `json:"backend"`
	EmbeddingModel    string    `json:"embeddingModel,omitempty"`
	EmbeddingReady    bool      `json:"embeddingReady"`
	StartedAt         time.Time `json:"startedAt,omitempty"`
	LastUpdatedAt     time.Time `json:"lastUpdatedAt,omitempty"`
	LeaderID          string    `json:"leaderId,omitempty"`
	RaftTerm          uint64    `json:"raftTerm"`
	Peers             []string  `json:"peers,omitempty"`
	PendingEmbeddings int       `json:"pendingEmbeddings"`
	CacheItems        int       `json:"cacheItems"`
}

// EmbeddingTask describes a queued embedding job.
type EmbeddingTask struct {
	Id           string    `json:"id"`
	ProjectID    string    `json:"projectId"`
	SourceType   string    `json:"sourceType"`
	SourceID     string    `json:"sourceId"`
	SourceField  string    `json:"sourceField"`
	Model        string    `json:"model"`
	ContentHash  string    `json:"contentHash"`
	Status       string    `json:"status"`
	QueuedAt     time.Time `json:"queuedAt"`
	AttemptCount int       `json:"attemptCount"`
	Payload      []byte    `json:"payload,omitempty"`
}

// Snapshot describes a runtime snapshot for restore and monitoring.
type Snapshot struct {
	Status Status          `json:"status"`
	Tasks  []EmbeddingTask `json:"tasks"`
}

// Config defines the initial vector runtime configuration.
type Config struct {
	DataDsn        string
	RedisDsn       string
	DataDir        string
	NodeID         string
	Peers          []string
	EmbeddingModel string
}

// Engine defines the storage backend for vector runtime state.
type Engine interface {
	Load() (Snapshot, bool, error)
	Save(Snapshot) error
}

// TaskStore persists the queued embedding tasks.
type TaskStore interface {
	Load() ([]EmbeddingTask, error)
	Replace([]EmbeddingTask) error
}

// EntryStore persists the computed vector entries.
type EntryStore interface {
	Load() ([]*models.VectorEntry, error)
	Replace([]*models.VectorEntry) error
}

// OperationType identifies a state transition that can be replayed by a
// replicated state machine.
type OperationType string

const (
	OperationTypeReplaceSnapshot OperationType = "replace_snapshot"
	OperationTypeSetTopology     OperationType = "set_topology"
	OperationTypeSetEmbedding    OperationType = "set_embedding_model"
	OperationTypeSetCounters     OperationType = "set_counters"
	OperationTypeEnqueueTask     OperationType = "enqueue_task"
	OperationTypeDequeueTask     OperationType = "dequeue_task"
)

// Operation is a replayable vector state transition.
type Operation struct {
	Type              OperationType  `json:"type"`
	Snapshot          *Snapshot      `json:"snapshot,omitempty"`
	Mode              Mode           `json:"mode,omitempty"`
	LeaderID          string         `json:"leaderId,omitempty"`
	RaftTerm          uint64         `json:"raftTerm,omitempty"`
	Peers             []string       `json:"peers,omitempty"`
	EmbeddingModel    string         `json:"embeddingModel,omitempty"`
	PendingEmbeddings int            `json:"pendingEmbeddings,omitempty"`
	CacheItems        int            `json:"cacheItems,omitempty"`
	Task              *EmbeddingTask `json:"task,omitempty"`
}

// Replayable converts the current status into a replay operation.
func (m *Manager) Replayable() Operation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := m.snapshotLocked()
	return Operation{
		Type:              OperationTypeReplaceSnapshot,
		Snapshot:          &snapshot,
		PendingEmbeddings: m.status.PendingEmbeddings,
		CacheItems:        m.status.CacheItems,
	}
}

// Manager provides a lightweight runtime model for the embedded vector stack.
//
// The Raft transport and sqlite-vec integration will be added in follow-up
// steps. For now it provides topology detection, status reporting and
// lifecycle hooks for the rest of the application.
type Manager struct {
	mu          sync.RWMutex
	config      Config
	status      Status
	tasks       []EmbeddingTask
	entries     []*models.VectorEntry
	engine      Engine
	store       TaskStore
	entryStore  EntryStore
	coordinator *Coordinator
}

// NewManager creates a new vector manager.
func NewManager(config Config) *Manager {
	if config.NodeID == "" {
		config.NodeID = newNodeID()
	}

	status := Status{
		Enabled:        true,
		Mode:           ModeStandalone,
		NodeID:         config.NodeID,
		DataDriver:     detectDataDriver(config.DataDsn),
		RedisEnabled:   config.RedisDsn != "",
		Backend:        "sqlite-vec",
		EmbeddingModel: config.EmbeddingModel,
		EmbeddingReady: config.EmbeddingModel != "",
		Peers:          append([]string(nil), config.Peers...),
		StartedAt:      time.Now().UTC(),
	}

	if len(config.Peers) > 0 {
		status.Mode = ModeCluster
	}

	return &Manager{
		config:  config,
		status:  status,
		tasks:   make([]EmbeddingTask, 0),
		entries: make([]*models.VectorEntry, 0),
		engine:  NewFileEngine(config.DataDir),
	}
}

// SetTaskStore replaces the current task persistence backend.
func (m *Manager) SetTaskStore(store TaskStore) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.store = store
}

// SetEntryStore replaces the current vector entry persistence backend.
func (m *Manager) SetEntryStore(store EntryStore) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entryStore = store
}

// Load restores the manager state from the configured snapshot file.
func (m *Manager) Load() error {
	if m.engine == nil {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.store != nil {
			tasks, err := m.store.Load()
			if err != nil {
				return err
			}
			m.tasks = append([]EmbeddingTask(nil), tasks...)
			m.status.PendingEmbeddings = len(m.tasks)
			m.applyDefaultsLocked()
		}
		return nil
	}

	snapshot, ok, err := m.engine.Load()
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if ok {
		m.status = snapshot.Status
		m.tasks = append([]EmbeddingTask(nil), snapshot.Tasks...)
	}
	if m.store != nil {
		tasks, loadErr := m.store.Load()
		if loadErr != nil {
			return loadErr
		}
		if !ok && len(tasks) == 0 {
			tasks = append([]EmbeddingTask(nil), m.tasks...)
		}
		if len(tasks) == 0 && len(m.tasks) > 0 {
			tasks = append([]EmbeddingTask(nil), m.tasks...)
			_ = m.store.Replace(tasks)
		}
		m.tasks = append([]EmbeddingTask(nil), tasks...)
	}
	if m.entryStore != nil {
		entries, loadErr := m.entryStore.Load()
		if loadErr != nil {
			return loadErr
		}
		m.entries = append([]*models.VectorEntry(nil), entries...)
	}
	m.status.PendingEmbeddings = len(m.tasks)
	m.applyDefaultsLocked()
	return nil
}

// Persist writes the manager state to the configured snapshot file.
func (m *Manager) Persist() error {
	if m.engine == nil {
		return nil
	}

	m.mu.RLock()
	snapshot := m.snapshotLocked()
	m.mu.RUnlock()
	return m.engine.Save(snapshot)
}

// ApplySnapshot replaces the current runtime state with the provided snapshot.
func (m *Manager) ApplySnapshot(snapshot Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status = snapshot.Status
	m.tasks = append([]EmbeddingTask(nil), snapshot.Tasks...)
	m.status.PendingEmbeddings = len(m.tasks)
	m.status.LastUpdatedAt = time.Now().UTC()
	m.applyDefaultsLocked()
	_ = m.persistTasksLocked()
	return m.persistLocked(m.snapshotLocked())
}

// ApplyOperation replays a state transition and persists the resulting state.
func (m *Manager) ApplyOperation(operation Operation) (Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch operation.Type {
	case OperationTypeReplaceSnapshot:
		if operation.Snapshot == nil {
			return Snapshot{}, errors.New("operation snapshot is required")
		}
		m.status = operation.Snapshot.Status
		m.tasks = append([]EmbeddingTask(nil), operation.Snapshot.Tasks...)
		m.status.PendingEmbeddings = len(m.tasks)
		m.applyDefaultsLocked()
	case OperationTypeSetTopology:
		m.status.Mode = operation.Mode
		m.status.LeaderID = operation.LeaderID
		m.status.RaftTerm = operation.RaftTerm
		m.status.Peers = append([]string(nil), operation.Peers...)
	case OperationTypeSetEmbedding:
		m.config.EmbeddingModel = operation.EmbeddingModel
		m.status.EmbeddingModel = operation.EmbeddingModel
		m.status.EmbeddingReady = operation.EmbeddingModel != ""
	case OperationTypeSetCounters:
		m.status.PendingEmbeddings = operation.PendingEmbeddings
		m.status.CacheItems = operation.CacheItems
	case OperationTypeEnqueueTask:
		if operation.Task == nil {
			return Snapshot{}, errors.New("operation task is required")
		}
		task := *operation.Task
		if task.Id == "" {
			task.Id = newNodeID()
		}
		if task.QueuedAt.IsZero() {
			task.QueuedAt = time.Now().UTC()
		}
		if task.Model == "" {
			task.Model = m.status.EmbeddingModel
		}
		m.tasks = append(m.tasks, task)
		m.status.PendingEmbeddings = len(m.tasks)
		_ = m.persistTasksLocked()
	case OperationTypeDequeueTask:
		if len(m.tasks) == 0 {
			return m.snapshotLocked(), nil
		}
		m.tasks = m.tasks[1:]
		m.status.PendingEmbeddings = len(m.tasks)
		_ = m.persistTasksLocked()
	default:
		return Snapshot{}, errors.New("unknown operation type")
	}

	m.status.LastUpdatedAt = time.Now().UTC()
	snapshot := m.snapshotLocked()
	if err := m.persistLocked(snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

// Start marks the vector runtime as active.
func (m *Manager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	m.status.Enabled = true
	m.status.LastUpdatedAt = now
	if m.status.StartedAt.IsZero() {
		m.status.StartedAt = now
	}
	_ = m.persistLocked(m.snapshotLocked())
}

// Stop marks the vector runtime as inactive.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status.Enabled = false
	m.status.LastUpdatedAt = time.Now().UTC()
	_ = m.persistLocked(m.snapshotLocked())
}

// Status returns a copy of the current vector runtime status.
func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := m.status
	status.Peers = append([]string(nil), status.Peers...)
	return status
}

// Tasks returns a copy of the current queued embedding tasks.
func (m *Manager) Tasks() []EmbeddingTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return append([]EmbeddingTask(nil), m.tasks...)
}

// Entries returns a copy of the current vector entries.
func (m *Manager) Entries() []*models.VectorEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return append([]*models.VectorEntry(nil), m.entries...)
}

// Snapshot returns the runtime status and queued embedding tasks.
func (m *Manager) Snapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.snapshotLocked()
}

// UpdateTopology refreshes the runtime status with the provided cluster hints.
func (m *Manager) UpdateTopology(mode Mode, leaderID string, raftTerm uint64, peers []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status.Mode = mode
	m.status.LeaderID = leaderID
	m.status.RaftTerm = raftTerm
	m.status.Peers = append([]string(nil), peers...)
	m.status.LastUpdatedAt = time.Now().UTC()
	_ = m.persistLocked(m.snapshotLocked())
}

// UpdateEmbeddingModel refreshes the embedding model tracking info.
func (m *Manager) UpdateEmbeddingModel(model string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config.EmbeddingModel = model
	m.status.EmbeddingModel = model
	m.status.EmbeddingReady = model != ""
	m.status.LastUpdatedAt = time.Now().UTC()
	_ = m.persistLocked(m.snapshotLocked())
}

// UpdateCounters refreshes the runtime counters used by the admin panel.
func (m *Manager) UpdateCounters(pendingEmbeddings, cacheItems int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status.PendingEmbeddings = pendingEmbeddings
	m.status.CacheItems = cacheItems
	m.status.LastUpdatedAt = time.Now().UTC()
	_ = m.persistLocked(m.snapshotLocked())
}

// ReplaceEntries replaces all vector entries and persists them.
func (m *Manager) ReplaceEntries(entries []*models.VectorEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries = append([]*models.VectorEntry(nil), entries...)
	return m.persistEntriesLocked()
}

// UpsertEntry inserts or replaces a single vector entry by its composite key.
func (m *Manager) UpsertEntry(entry *models.VectorEntry) error {
	if entry == nil {
		return errors.New("entry is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	replaced := false
	for i, current := range m.entries {
		if current == nil {
			continue
		}
		if current.ProjectID == entry.ProjectID &&
			current.SourceType == entry.SourceType &&
			current.SourceID == entry.SourceID &&
			current.SourceField == entry.SourceField &&
			current.EmbeddingModel == entry.EmbeddingModel {
			m.entries[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		m.entries = append(m.entries, entry)
	}

	return m.persistEntriesLocked()
}

// DeleteEntry removes a vector entry by its composite key.
func (m *Manager) DeleteEntry(entry *models.VectorEntry) error {
	if entry == nil {
		return errors.New("entry is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	filtered := m.entries[:0]
	for _, current := range m.entries {
		if current == nil {
			continue
		}
		if current.ProjectID == entry.ProjectID &&
			current.SourceType == entry.SourceType &&
			current.SourceID == entry.SourceID &&
			current.SourceField == entry.SourceField &&
			current.EmbeddingModel == entry.EmbeddingModel {
			continue
		}
		filtered = append(filtered, current)
	}
	m.entries = append([]*models.VectorEntry(nil), filtered...)

	return m.persistEntriesLocked()
}

// EnqueueEmbedding records a new embedding task and returns its id. When a
// cluster coordinator is attached the enqueue is proposed through it so the
// task queue stays consistent across instances.
func (m *Manager) EnqueueEmbedding(task EmbeddingTask) string {
	if task.Id == "" {
		task.Id = newNodeID()
	}
	if task.QueuedAt.IsZero() {
		task.QueuedAt = time.Now().UTC()
	}

	m.mu.RLock()
	coordinator := m.coordinator
	if task.Model == "" {
		task.Model = m.status.EmbeddingModel
	}
	m.mu.RUnlock()

	if coordinator != nil && coordinator.HasPeers() {
		if _, err := coordinator.Propose(Operation{
			Type: OperationTypeEnqueueTask,
			Task: &task,
		}); err == nil {
			return task.Id
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.tasks = append(m.tasks, task)
	m.status.PendingEmbeddings = len(m.tasks)
	m.status.LastUpdatedAt = time.Now().UTC()
	_ = m.persistTasksLocked()
	_ = m.persistLocked(m.snapshotLocked())
	return task.Id
}

// DequeueEmbedding removes the oldest queued embedding task.
func (m *Manager) DequeueEmbedding() (EmbeddingTask, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.tasks) == 0 {
		return EmbeddingTask{}, false
	}

	task := m.tasks[0]
	m.tasks = m.tasks[1:]
	m.status.PendingEmbeddings = len(m.tasks)
	m.status.LastUpdatedAt = time.Now().UTC()
	_ = m.persistTasksLocked()
	_ = m.persistLocked(m.snapshotLocked())
	return task, true
}

func (m *Manager) snapshotLocked() Snapshot {
	return Snapshot{
		Status: m.status,
		Tasks:  append([]EmbeddingTask(nil), m.tasks...),
	}
}

func (m *Manager) applyDefaultsLocked() {
	if m.status.NodeID == "" {
		m.status.NodeID = m.config.NodeID
	}
	if m.status.DataDriver == "" {
		m.status.DataDriver = detectDataDriver(m.config.DataDsn)
	}
	if m.status.Backend == "" {
		m.status.Backend = "sqlite-vec"
	}
	if m.status.EmbeddingModel == "" {
		m.status.EmbeddingModel = m.config.EmbeddingModel
	}
	if !m.status.EmbeddingReady {
		m.status.EmbeddingReady = m.status.EmbeddingModel != ""
	}
	if m.status.StartedAt.IsZero() {
		m.status.StartedAt = time.Now().UTC()
	}
}

func (m *Manager) persistLocked(snapshot Snapshot) error {
	if m.engine == nil {
		return nil
	}
	return m.engine.Save(snapshot)
}

func (m *Manager) persistTasksLocked() error {
	if m.store == nil {
		return nil
	}
	return m.store.Replace(append([]EmbeddingTask(nil), m.tasks...))
}

func (m *Manager) persistEntriesLocked() error {
	if m.entryStore == nil {
		return nil
	}
	return m.entryStore.Replace(append([]*models.VectorEntry(nil), m.entries...))
}

// AttachCoordinator wires a cluster coordinator to the manager so write
// operations can be replicated across instances.
func (m *Manager) AttachCoordinator(coordinator *Coordinator) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.coordinator = coordinator
}

// Coordinator returns the attached cluster coordinator (may be nil).
func (m *Manager) Coordinator() *Coordinator {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.coordinator
}

// IsLeader reports whether this node may execute coordinated write tasks. In
// standalone mode it always returns true.
func (m *Manager) IsLeader() bool {
	m.mu.RLock()
	coordinator := m.coordinator
	m.mu.RUnlock()
	if coordinator == nil {
		return true
	}
	return coordinator.IsLeader()
}

// SetEngine replaces the current storage backend.
func (m *Manager) SetEngine(engine Engine) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.engine = engine
}

// FileEngine is a file-backed state machine for vector snapshots.
type FileEngine struct {
	baseDir string
}

// NewFileEngine creates a file-backed engine.
func NewFileEngine(baseDir string) *FileEngine {
	return &FileEngine{baseDir: baseDir}
}

func (e *FileEngine) Load() (Snapshot, bool, error) {
	data, err := os.ReadFile(e.path())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Snapshot{}, false, nil
		}
		return Snapshot{}, false, err
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, false, err
	}

	return snapshot, true, nil
}

func (e *FileEngine) Save(snapshot Snapshot) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	path := e.path()
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

func (e *FileEngine) path() string {
	if e.baseDir == "" {
		return filepath.Join(os.TempDir(), "postgrebase-vector-state.json")
	}
	return filepath.Join(e.baseDir, "vector-state.json")
}

func newNodeID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "node-" + time.Now().UTC().Format("20060102150405")
	}

	return "node-" + hex.EncodeToString(buf[:])
}

func detectDataDriver(dataDsn string) string {
	switch {
	case strings.HasPrefix(dataDsn, "postgres://"):
		return "postgres"
	case strings.HasPrefix(dataDsn, "postgresql://"):
		return "postgres"
	case strings.HasPrefix(dataDsn, "mysql://"):
		return "mysql"
	case strings.HasPrefix(dataDsn, "sqlite://"):
		return "sqlite"
	case strings.HasPrefix(dataDsn, "sqlite3://"):
		return "sqlite"
	default:
		return "postgres"
	}
}
