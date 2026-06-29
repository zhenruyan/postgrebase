package vector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Heartbeat is exchanged between cluster members to share liveness and the
// current leadership view.
type Heartbeat struct {
	Term         uint64   `json:"term"`
	LeaderID     string   `json:"leaderId"`
	NodeID       string   `json:"nodeId"`
	Address      string   `json:"address"`
	Members      []string `json:"members"`
	SentAt       int64    `json:"sentAt"`
	LastLogIndex uint64   `json:"lastLogIndex,omitempty"`
}

// HeartbeatReply is returned by a member after processing a heartbeat.
type HeartbeatReply struct {
	Term         uint64 `json:"term"`
	NodeID       string `json:"nodeId"`
	Address      string `json:"address"`
	LeaderID     string `json:"leaderId"`
	Accepted     bool   `json:"accepted"`
	LastLogIndex uint64 `json:"lastLogIndex,omitempty"`
}

// PeerState describes the last known state of a cluster member.
type PeerState struct {
	Address      string    `json:"address"`
	NodeID       string    `json:"nodeId"`
	Alive        bool      `json:"alive"`
	IsLeader     bool      `json:"isLeader"`
	IsSelf       bool      `json:"isSelf"`
	Term         uint64    `json:"term"`
	LastLogIndex uint64    `json:"lastLogIndex"`
	LastSeen     time.Time `json:"lastSeen,omitempty"`
}

// ClusterView is a point-in-time snapshot of the cluster membership used by the
// monitoring panel.
type ClusterView struct {
	Mode     Mode        `json:"mode"`
	SelfAddr string      `json:"selfAddr"`
	NodeID   string      `json:"nodeId"`
	LeaderID string      `json:"leaderId"`
	IsLeader bool        `json:"isLeader"`
	Term     uint64      `json:"term"`
	Members  []PeerState `json:"members"`
}

// SnapshotApp defines the app methods required for snapshot and file replacements.
type SnapshotApp interface {
	CreateConsistentSnapshot(targetPath string) error
	ReloadDataDBWithReplacement(replacePath string) error
}

// Transport abstracts the peer-to-peer communication used by the coordinator so
// it can be backed by HTTP in production and an in-memory bus in tests.
type Transport interface {
	SendHeartbeat(ctx context.Context, peer string, hb Heartbeat) (HeartbeatReply, error)
	Replicate(ctx context.Context, peer string, op ReplicatedOperation) error
	Forward(ctx context.Context, peer string, op ReplicatedOperation) error
	SendSnapshot(ctx context.Context, peer string, snapshotReader io.Reader, lastLogIndex uint64, appliedLogIndex uint64, term uint64) error
	Join(ctx context.Context, leader string, selfAddr string) error
}

var ErrLeaderUnavailable = errors.New("cluster leader is unavailable")

// Coordinator implements a lightweight Raft-inspired coordination layer. It is
// intentionally small: leadership is derived deterministically from the set of
// currently-alive members (the lexicographically smallest reachable address
// wins) and the term is bumped whenever leadership changes. This is enough to
// guarantee a single writer for vector/cache state replication without pulling
// in a full consensus implementation.
type Coordinator struct {
	mu        sync.RWMutex
	manager   *Manager
	transport Transport
	apply     ApplyFunc
	app       SnapshotApp

	selfAddr string
	nodeID   string
	peers    []string

	term     uint64
	leaderID string
	states   map[string]*PeerState

	interval time.Duration
	timeout  time.Duration

	lastLogIndex    uint64
	appliedLogIndex uint64
	logs            []ReplicatedOperation
	peerNextIndex   map[string]uint64

	stop    chan struct{}
	stopped chan struct{}
	once    sync.Once
}

// CoordinatorConfig configures a new coordinator.
type CoordinatorConfig struct {
	SelfAddr  string
	NodeID    string
	Peers     []string
	Interval  time.Duration
	Timeout   time.Duration
	Transport Transport
	Apply     ApplyFunc
	App       SnapshotApp
}

// NewCoordinator creates a coordinator for the given manager.
func NewCoordinator(manager *Manager, config CoordinatorConfig) *Coordinator {
	interval := config.Interval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = interval + time.Second
	}

	nodeID := config.NodeID
	if nodeID == "" && manager != nil {
		nodeID = manager.Status().NodeID
	}

	term := uint64(0)
	lastLogIndex := uint64(0)
	appliedLogIndex := uint64(0)
	if manager != nil {
		status := manager.Status()
		term = status.RaftTerm
		lastLogIndex = status.LastLogIndex
		appliedLogIndex = status.AppliedLogIndex
	}

	c := &Coordinator{
		manager:         manager,
		transport:       config.Transport,
		apply:           config.Apply,
		app:             config.App,
		selfAddr:        config.SelfAddr,
		nodeID:          nodeID,
		peers:           dedupePeers(config.SelfAddr, config.Peers),
		states:          make(map[string]*PeerState),
		interval:        interval,
		timeout:         timeout,
		term:            term,
		lastLogIndex:    lastLogIndex,
		appliedLogIndex: appliedLogIndex,
		logs:            make([]ReplicatedOperation, 0),
		peerNextIndex:   make(map[string]uint64),
		stop:            make(chan struct{}),
		stopped:         make(chan struct{}),
	}

	// seed self state
	c.states[c.selfAddr] = &PeerState{
		Address:  c.selfAddr,
		NodeID:   c.nodeID,
		Alive:    true,
		IsSelf:   true,
		LastSeen: time.Now().UTC(),
	}
	for _, peer := range c.peers {
		if _, ok := c.states[peer]; !ok {
			c.states[peer] = &PeerState{Address: peer}
		}
		c.peerNextIndex[peer] = lastLogIndex + 1
	}

	c.recomputeLeaderLocked()
	return c
}

// HasPeers reports whether the coordinator is configured for multi-instance
// operation.
func (c *Coordinator) HasPeers() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.peers) > 0
}

// Start launches the background heartbeat/election loop.
func (c *Coordinator) Start() {
	if c == nil {
		return
	}
	if !c.HasPeers() {
		// standalone: still publish topology so the panel reflects it
		c.syncTopology()
		return
	}

	go c.loop()
	go c.replayLoop()
}

// Stop terminates the background loop.
func (c *Coordinator) Stop() {
	if c == nil {
		return
	}
	c.once.Do(func() {
		close(c.stop)
	})
	if c.HasPeers() {
		select {
		case <-c.stopped:
		case <-time.After(c.timeout):
		}
	}
}

func (c *Coordinator) loop() {
	defer close(c.stopped)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	c.tick()
	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.tick()
		}
	}
}

func (c *Coordinator) replayLoop() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.retryFailedReplications()
		}
	}
}

func (c *Coordinator) retryFailedReplications() {
	isLeader := c.IsLeader()
	c.mu.Lock()
	peers := append([]string(nil), c.peers...)
	lastIdx := c.appliedLogIndex
	self := c.selfAddr
	transport := c.transport
	timeout := c.timeout
	c.mu.Unlock()

	if !isLeader {
		return
	}

	if transport == nil || len(peers) == 0 {
		return
	}

	for _, peer := range peers {
		if peer == self {
			continue
		}

		c.mu.RLock()
		nextIdx := c.peerNextIndex[peer]
		if nextIdx == 0 {
			nextIdx = 1
		}

		// Adjust next index based on the follower's reported progress in heartbeats
		if state, ok := c.states[peer]; ok && state != nil && state.Alive {
			if state.LastLogIndex+1 < nextIdx {
				nextIdx = state.LastLogIndex + 1
			}
		}

		var lowestLogIndex uint64 = 0
		if len(c.logs) > 0 {
			lowestLogIndex = c.logs[0].LogIndex
		}

		hasLogGap := (nextIdx < lowestLogIndex && lowestLogIndex > 0) || (len(c.logs) == 0 && lastIdx > 0 && nextIdx <= lastIdx)
		termVal := c.term
		c.mu.RUnlock()

		if hasLogGap {
			if c.app != nil {
				go func(p string, lIdx uint64, appIdx uint64, tVal uint64) {
					tempPath := filepath.Join(os.TempDir(), fmt.Sprintf("pb_snapshot_%d.db", time.Now().UnixNano()))
					defer os.Remove(tempPath)

					if err := c.app.CreateConsistentSnapshot(tempPath); err != nil {
						return
					}

					file, err := os.Open(tempPath)
					if err != nil {
						return
					}
					defer file.Close()

					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
					err = transport.SendSnapshot(ctx, p, file, lIdx, appIdx, tVal)
					cancel()
					if err == nil {
						c.mu.Lock()
						c.peerNextIndex[p] = lIdx + 1
						c.mu.Unlock()
					}
				}(peer, lastIdx, lastIdx, termVal)
			} else {
			}
			continue
		}

		c.mu.RLock()
		var opsToReplay []ReplicatedOperation
		if nextIdx <= lastIdx {
			for _, logOp := range c.logs {
				if logOp.LogIndex >= nextIdx && logOp.LogIndex <= lastIdx {
					opsToReplay = append(opsToReplay, logOp)
				}
			}
		}
		c.mu.RUnlock()

		if len(opsToReplay) == 0 {
			continue
		}

		go func(p string, ops []ReplicatedOperation, startIdx uint64) {
			successCount := 0
			for _, op := range ops {
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				err := transport.Replicate(ctx, p, op)
				cancel()
				if err != nil {
					break
				}
				successCount++
			}

			if successCount > 0 {
				c.mu.Lock()
				if c.peerNextIndex[p] == startIdx {
					c.peerNextIndex[p] = startIdx + uint64(successCount)
				}
				c.mu.Unlock()
			}
		}(peer, opsToReplay, nextIdx)
	}
}

func (c *Coordinator) tick() {
	c.mu.RLock()
	peers := append([]string(nil), c.peers...)
	hb := c.heartbeatLocked()
	transport := c.transport
	timeout := c.timeout
	c.mu.RUnlock()

	if transport == nil {
		return
	}

	for _, peer := range peers {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		reply, err := transport.SendHeartbeat(ctx, peer, hb)
		cancel()

		c.mu.Lock()
		state := c.states[peer]
		if state == nil {
			state = &PeerState{Address: peer}
			c.states[peer] = state
		}
		if err != nil {
			state.Alive = false
		} else {
			state.Alive = true
			state.NodeID = reply.NodeID
			state.Term = reply.Term
			state.LastLogIndex = reply.LastLogIndex
			state.LastSeen = time.Now().UTC()
		}
		c.mu.Unlock()
	}

	c.mu.Lock()
	c.recomputeLeaderLocked()
	c.mu.Unlock()

	c.syncTopology()
}

// ReceiveHeartbeat processes an inbound heartbeat from a peer.
func (c *Coordinator) ReceiveHeartbeat(hb Heartbeat) HeartbeatReply {
	c.mu.Lock()
	defer c.mu.Unlock()

	if hb.Address != "" {
		state := c.states[hb.Address]
		if state == nil {
			state = &PeerState{Address: hb.Address}
			c.states[hb.Address] = state
		}
		state.Alive = true
		state.NodeID = hb.NodeID
		state.Term = hb.Term
		state.LastLogIndex = hb.LastLogIndex
		state.LastSeen = time.Now().UTC()

		// learn about new members announced by the peer
		if !containsString(c.peers, hb.Address) && hb.Address != c.selfAddr {
			c.peers = append(c.peers, hb.Address)
		}
	}

	if hb.Term > c.term {
		c.term = hb.Term
	}

	c.recomputeLeaderLocked()

	return HeartbeatReply{
		Term:         c.term,
		NodeID:       c.nodeID,
		Address:      c.selfAddr,
		LeaderID:     c.leaderID,
		Accepted:     true,
		LastLogIndex: c.lastLogIndex,
	}
}

// Propose applies an operation through the cluster. On the leader the operation
// is applied locally and replicated to the followers. On a follower it is
// forwarded to the leader; if forwarding fails it is applied locally as a
// best-effort fallback so single writes never get lost.
func (c *Coordinator) Propose(op Operation) (Snapshot, error) {
	wrapped, err := WrapVectorOperation(op)
	if err != nil {
		return Snapshot{}, err
	}
	if _, err := c.ProposeReplicated(wrapped); err != nil {
		return Snapshot{}, err
	}
	if c.manager == nil {
		return Snapshot{}, nil
	}
	return c.manager.Snapshot(), nil
}

// ProposeReplicated applies a common replicated operation through the cluster.
// Strict operations are never applied locally on a follower if leader forwarding
// fails; this is required for SQLite primary database consistency.
func (c *Coordinator) ProposeReplicated(op ReplicatedOperation) (Snapshot, error) {
	if c == nil || c.manager == nil {
		return Snapshot{}, nil
	}

	c.mu.RLock()
	isLeader := c.isLeaderLocked()
	leader := c.leaderID
	leaderAddr := c.leaderAddrLocked()
	peers := append([]string(nil), c.peers...)
	transport := c.transport
	timeout := c.timeout
	term := c.term
	c.mu.RUnlock()

	hasPeers := c.HasPeers()
	if op.Strict && hasPeers && transport == nil {
		return Snapshot{}, ErrLeaderUnavailable
	}

	// standalone or leader: apply locally
	if !hasPeers || isLeader {
		c.mu.Lock()
		c.lastLogIndex++
		op.LogIndex = c.lastLogIndex
		op.RaftTerm = term
		c.logs = append(c.logs, op)
		c.mu.Unlock()

		if hasPeers && transport != nil && op.Strict {
			if err := c.replicateSyncQuorum(peers, op); err != nil {
				c.mu.Lock()
				if len(c.logs) > 0 && c.logs[len(c.logs)-1].LogIndex == op.LogIndex {
					c.logs = c.logs[:len(c.logs)-1]
				}
				if c.lastLogIndex == op.LogIndex {
					c.lastLogIndex--
				}
				c.mu.Unlock()
				return Snapshot{}, err
			}
		}

		snapshot, err := c.applyOperation(op)
		if err != nil {
			if hasPeers && transport != nil && op.Strict {
				c.mu.Lock()
				if len(c.logs) > 0 && c.logs[len(c.logs)-1].LogIndex == op.LogIndex {
					c.logs = c.logs[:len(c.logs)-1]
				}
				if c.lastLogIndex == op.LogIndex {
					c.lastLogIndex--
				}
				c.mu.Unlock()
			}
			return Snapshot{}, err
		}

		c.mu.Lock()
		if op.LogIndex > c.appliedLogIndex {
			c.appliedLogIndex = op.LogIndex
		}
		c.mu.Unlock()

		if c.manager != nil {
			_ = c.manager.Persist()
		}

		if hasPeers && transport != nil && !op.Strict {
			c.replicateAsync(peers, op)
		}
		return snapshot, nil
	}

	if op.Strict && (leader == "" || transport == nil) {
		return Snapshot{}, ErrLeaderUnavailable
	}

	if leader == "" || transport == nil {
		return c.applyOperation(op)
	}

	// follower: forward to leader
	if leaderAddr != "" && leaderAddr != c.selfAddr {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		err := transport.Forward(ctx, leaderAddr, op)
		cancel()
		if err == nil {
			return c.manager.Snapshot(), nil
		}
	}

	if op.Strict {
		return Snapshot{}, ErrLeaderUnavailable
	}

	// fallback: apply locally
	return c.applyOperation(op)
}

// ApplyReplicated applies an operation received from the leader.
func (c *Coordinator) ApplyReplicated(op ReplicatedOperation) (Snapshot, error) {
	if c == nil || c.manager == nil {
		return Snapshot{}, nil
	}

	c.mu.Lock()
	if op.LogIndex > 0 {
		if op.LogIndex <= c.appliedLogIndex {
			c.mu.Unlock()
			return c.manager.Snapshot(), nil
		}
		if op.Strict && op.LogIndex > c.appliedLogIndex+1 {
			c.mu.Unlock()
			return Snapshot{}, fmt.Errorf("log index gap detected: incoming %d, applied %d", op.LogIndex, c.appliedLogIndex)
		}
	}
	c.mu.Unlock()

	snap, err := c.applyOperation(op)
	if err != nil {
		return Snapshot{}, err
	}

	c.mu.Lock()
	if op.LogIndex > c.appliedLogIndex {
		c.appliedLogIndex = op.LogIndex
	}
	if op.LogIndex > c.lastLogIndex {
		c.lastLogIndex = op.LogIndex
	}
	c.logs = append(c.logs, op)
	c.mu.Unlock()

	if c.manager != nil {
		_ = c.manager.Persist()
	}

	return snap, nil
}

func (c *Coordinator) replicateAsync(peers []string, op ReplicatedOperation) {
	c.mu.RLock()
	transport := c.transport
	timeout := c.timeout
	self := c.selfAddr
	c.mu.RUnlock()

	if transport == nil {
		return
	}

	for _, peer := range peers {
		if peer == self {
			continue
		}
		go func(p string) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			err := transport.Replicate(ctx, p, op)
			cancel()
			c.mu.Lock()
			if err == nil {
				if c.peerNextIndex[p] <= op.LogIndex {
					c.peerNextIndex[p] = op.LogIndex + 1
				}
			} else {
				if c.peerNextIndex[p] == 0 || c.peerNextIndex[p] > op.LogIndex {
					c.peerNextIndex[p] = op.LogIndex
				}
			}
			c.mu.Unlock()
		}(peer)
	}
}

func (c *Coordinator) replicateSync(peers []string, op ReplicatedOperation) error {
	c.mu.RLock()
	transport := c.transport
	timeout := c.timeout
	self := c.selfAddr
	c.mu.RUnlock()

	if transport == nil {
		return ErrLeaderUnavailable
	}

	for _, peer := range peers {
		if peer == self {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		err := transport.Replicate(ctx, peer, op)
		cancel()
		if err != nil {
			return fmt.Errorf("replicate to %s: %w", peer, err)
		}
	}
	return nil
}

func (c *Coordinator) replicateSyncQuorum(peers []string, op ReplicatedOperation) error {
	c.mu.RLock()
	transport := c.transport
	timeout := c.timeout
	self := c.selfAddr
	c.mu.RUnlock()

	if transport == nil {
		return ErrLeaderUnavailable
	}

	totalNodes := len(peers) + 1
	quorum := (totalNodes / 2) + 1
	neededPeers := quorum - 1

	if neededPeers <= 0 {
		return nil
	}

	type peerResult struct {
		peer string
		err  error
	}

	resultCh := make(chan peerResult, len(peers))
	var wg sync.WaitGroup

	for _, peer := range peers {
		if peer == self {
			continue
		}
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			err := transport.Replicate(ctx, p, op)
			cancel()
			resultCh <- peerResult{peer: p, err: err}
		}(peer)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	successCount := 0
	var firstErr error

	for res := range resultCh {
		if res.err == nil {
			successCount++
			c.mu.Lock()
			if c.peerNextIndex[res.peer] <= op.LogIndex {
				c.peerNextIndex[res.peer] = op.LogIndex + 1
			}
			c.mu.Unlock()
		} else {
			if firstErr == nil {
				firstErr = res.err
			}
			c.mu.Lock()
			if c.peerNextIndex[res.peer] == 0 || c.peerNextIndex[res.peer] > op.LogIndex {
				c.peerNextIndex[res.peer] = op.LogIndex
			}
			c.mu.Unlock()
		}
	}

	if successCount >= neededPeers {
		return nil
	}

	if firstErr != nil {
		return fmt.Errorf("quorum failed (got %d/%d successes), first error: %w", successCount, neededPeers, firstErr)
	}
	return fmt.Errorf("quorum failed (got %d/%d successes)", successCount, neededPeers)
}

func (c *Coordinator) applyOperation(op ReplicatedOperation) (Snapshot, error) {
	if op.Type == "cluster.join" {
		var payload struct {
			Addr string `json:"addr"`
		}
		if err := json.Unmarshal(op.Payload, &payload); err == nil && payload.Addr != "" {
			c.AddPeer(payload.Addr)
		}
		return c.manager.Snapshot(), nil
	}

	if op.Kind == "" || op.Kind == ReplicatedOperationKindVector {
		vectorOp, err := UnwrapVectorOperation(op)
		if err != nil {
			return Snapshot{}, err
		}
		return c.manager.ApplyOperation(vectorOp)
	}

	if c.apply == nil {
		return c.manager.Snapshot(), nil
	}
	if err := c.apply(op); err != nil {
		return Snapshot{}, err
	}
	return c.manager.Snapshot(), nil
}

// View returns the current cluster view for monitoring.
func (c *Coordinator) View() ClusterView {
	c.mu.RLock()
	defer c.mu.RUnlock()

	members := make([]PeerState, 0, len(c.states))
	for addr, state := range c.states {
		ps := *state
		ps.IsSelf = addr == c.selfAddr
		ps.IsLeader = addr == c.leaderID
		members = append(members, ps)
	}
	sort.Slice(members, func(i, j int) bool {
		return members[i].Address < members[j].Address
	})

	mode := ModeStandalone
	if len(c.peers) > 0 {
		mode = ModeCluster
	}

	return ClusterView{
		Mode:     mode,
		SelfAddr: c.selfAddr,
		NodeID:   c.nodeID,
		LeaderID: c.leaderID,
		IsLeader: c.isLeaderLocked(),
		Term:     c.term,
		Members:  members,
	}
}

// IsLeader reports whether this node is currently the cluster leader.
func (c *Coordinator) IsLeader() bool {
	if c == nil {
		return true
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isLeaderLocked()
}

// Transport returns the coordinator's communication transport.
func (c *Coordinator) Transport() Transport {
	if c == nil {
		return nil
	}
	return c.transport
}

// AddPeer dynamically adds a new peer address to the coordinator's list.
func (c *Coordinator) AddPeer(peer string) {
	if c == nil || peer == "" || peer == c.selfAddr {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, p := range c.peers {
		if p == peer {
			return
		}
	}

	c.peers = append(c.peers, peer)
	if c.peerNextIndex != nil {
		c.peerNextIndex[peer] = c.lastLogIndex + 1
	}
}

// AdvanceIndexLocally manually increments the cluster logs indices on local save.
func (c *Coordinator) AdvanceIndexLocally() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.lastLogIndex++
	c.appliedLogIndex++
	c.mu.Unlock()
}

// InstallSnapshot is invoked on a follower to replace its physical SQLite state machine.
func (c *Coordinator) InstallSnapshot(replacePath string, lastLogIndex uint64, appliedLogIndex uint64, term uint64) error {
	if c == nil || c.app == nil {
		return errors.New("cannot install snapshot: app not configured")
	}

	// 1. Atomically replace database file using the app's database lifecycle hook
	if err := c.app.ReloadDataDBWithReplacement(replacePath); err != nil {
		return fmt.Errorf("failed to reload database with snapshot: %w", err)
	}

	// 2. Synchronize coordinator log sequence watermarks
	c.mu.Lock()
	c.appliedLogIndex = appliedLogIndex
	c.lastLogIndex = lastLogIndex
	c.term = term
	c.logs = nil // Clear local logs as we are now fully aligned with snapshot
	c.mu.Unlock()

	if c.manager != nil {
		_ = c.manager.Persist()
	}

	return nil
}

func (c *Coordinator) heartbeatLocked() Heartbeat {
	members := make([]string, 0, len(c.states))
	for addr := range c.states {
		members = append(members, addr)
	}
	sort.Strings(members)

	return Heartbeat{
		Term:         c.term,
		LeaderID:     c.leaderID,
		NodeID:       c.nodeID,
		Address:      c.selfAddr,
		Members:      members,
		SentAt:       time.Now().UTC().Unix(),
		LastLogIndex: c.lastLogIndex,
	}
}

// recomputeLeaderLocked derives the leader from the set of alive members.
func (c *Coordinator) recomputeLeaderLocked() {
	maxIndex := c.lastLogIndex
	for addr, state := range c.states {
		if addr != c.selfAddr && state.Alive && state.LastLogIndex > maxIndex {
			maxIndex = state.LastLogIndex
		}
	}

	candidates := make([]string, 0, len(c.states))
	for addr, state := range c.states {
		if addr == c.selfAddr {
			if c.lastLogIndex >= maxIndex {
				candidates = append(candidates, addr)
			}
		} else if state.Alive {
			if state.LastLogIndex >= maxIndex {
				candidates = append(candidates, addr)
			}
		}
	}
	sort.Strings(candidates)

	newLeader := ""
	if len(candidates) > 0 {
		newLeader = candidates[0]
	} else {
		// Fallback to absolute lexicographically smallest among alive
		aliveNodes := make([]string, 0)
		for addr, state := range c.states {
			if addr == c.selfAddr || state.Alive {
				aliveNodes = append(aliveNodes, addr)
			}
		}
		sort.Strings(aliveNodes)
		if len(aliveNodes) > 0 {
			newLeader = aliveNodes[0]
		}
	}

	if newLeader != c.leaderID {
		c.leaderID = newLeader
		c.term++
	}
}

func (c *Coordinator) isLeaderLocked() bool {
	if len(c.peers) == 0 {
		return true
	}
	return c.leaderID == c.selfAddr
}

func (c *Coordinator) leaderAddrLocked() string {
	return c.leaderID
}

func (c *Coordinator) syncTopology() {
	if c.manager == nil {
		return
	}

	c.mu.RLock()
	mode := ModeStandalone
	if len(c.peers) > 0 {
		mode = ModeCluster
	}
	leader := c.leaderID
	term := c.term
	peers := make([]string, 0, len(c.states))
	for addr := range c.states {
		if addr != c.selfAddr {
			peers = append(peers, addr)
		}
	}
	sort.Strings(peers)
	c.mu.RUnlock()

	c.manager.UpdateTopology(mode, leader, term, peers)
}

func dedupePeers(self string, peers []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(peers))
	for _, peer := range peers {
		if peer == "" || peer == self {
			continue
		}
		if _, ok := seen[peer]; ok {
			continue
		}
		seen[peer] = struct{}{}
		result = append(result, peer)
	}
	sort.Strings(result)
	return result
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

// AppliedLogIndex returns the coordinator's applied log index.
func (c *Coordinator) AppliedLogIndex() uint64 {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.appliedLogIndex
}

// LastLogIndex returns the coordinator's last log index.
func (c *Coordinator) LastLogIndex() uint64 {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastLogIndex
}

// RaftTerm returns the coordinator's term.
func (c *Coordinator) RaftTerm() uint64 {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.term
}
