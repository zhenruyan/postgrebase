package vector

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Heartbeat is exchanged between cluster members to share liveness and the
// current leadership view.
type Heartbeat struct {
	Term     uint64   `json:"term"`
	LeaderID string   `json:"leaderId"`
	NodeID   string   `json:"nodeId"`
	Address  string   `json:"address"`
	Members  []string `json:"members"`
	SentAt   int64    `json:"sentAt"`
}

// HeartbeatReply is returned by a member after processing a heartbeat.
type HeartbeatReply struct {
	Term     uint64 `json:"term"`
	NodeID   string `json:"nodeId"`
	Address  string `json:"address"`
	LeaderID string `json:"leaderId"`
	Accepted bool   `json:"accepted"`
}

// PeerState describes the last known state of a cluster member.
type PeerState struct {
	Address  string    `json:"address"`
	NodeID   string    `json:"nodeId"`
	Alive    bool      `json:"alive"`
	IsLeader bool      `json:"isLeader"`
	IsSelf   bool      `json:"isSelf"`
	Term     uint64    `json:"term"`
	LastSeen time.Time `json:"lastSeen,omitempty"`
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

// Transport abstracts the peer-to-peer communication used by the coordinator so
// it can be backed by HTTP in production and an in-memory bus in tests.
type Transport interface {
	SendHeartbeat(ctx context.Context, peer string, hb Heartbeat) (HeartbeatReply, error)
	Replicate(ctx context.Context, peer string, op ReplicatedOperation) error
	Forward(ctx context.Context, peer string, op ReplicatedOperation) error
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

	selfAddr string
	nodeID   string
	peers    []string

	term     uint64
	leaderID string
	states   map[string]*PeerState

	interval time.Duration
	timeout  time.Duration

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

	c := &Coordinator{
		manager:   manager,
		transport: config.Transport,
		apply:     config.Apply,
		selfAddr:  config.SelfAddr,
		nodeID:    nodeID,
		peers:     dedupePeers(config.SelfAddr, config.Peers),
		states:    make(map[string]*PeerState),
		interval:  interval,
		timeout:   timeout,
		stop:      make(chan struct{}),
		stopped:   make(chan struct{}),
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
		Term:     c.term,
		NodeID:   c.nodeID,
		Address:  c.selfAddr,
		LeaderID: c.leaderID,
		Accepted: true,
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
	c.mu.RUnlock()

	hasPeers := c.HasPeers()
	if op.Strict && hasPeers && transport == nil {
		return Snapshot{}, ErrLeaderUnavailable
	}

	// standalone or leader: apply locally
	if !hasPeers || isLeader {
		snapshot, err := c.applyOperation(op)
		if err != nil {
			return Snapshot{}, err
		}
		if hasPeers && transport != nil {
			if op.Strict {
				if err := c.replicateSync(peers, op); err != nil {
					return Snapshot{}, err
				}
			} else {
				c.replicateAsync(peers, op)
			}
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
	return c.applyOperation(op)
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
		go func(peer string) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			_ = transport.Replicate(ctx, peer, op)
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

func (c *Coordinator) applyOperation(op ReplicatedOperation) (Snapshot, error) {
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

func (c *Coordinator) heartbeatLocked() Heartbeat {
	members := make([]string, 0, len(c.states))
	for addr := range c.states {
		members = append(members, addr)
	}
	sort.Strings(members)

	return Heartbeat{
		Term:     c.term,
		LeaderID: c.leaderID,
		NodeID:   c.nodeID,
		Address:  c.selfAddr,
		Members:  members,
		SentAt:   time.Now().UTC().Unix(),
	}
}

// recomputeLeaderLocked derives the leader from the set of alive members.
func (c *Coordinator) recomputeLeaderLocked() {
	candidates := make([]string, 0, len(c.states))
	for addr, state := range c.states {
		if addr == c.selfAddr || state.Alive {
			candidates = append(candidates, addr)
		}
	}
	sort.Strings(candidates)

	newLeader := ""
	if len(candidates) > 0 {
		newLeader = candidates[0]
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
