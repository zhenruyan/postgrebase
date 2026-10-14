package vector

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// memoryBus connects multiple coordinators in-process for testing.
type memoryBus struct {
	mu    sync.RWMutex
	nodes map[string]*Coordinator
	down  map[string]bool
	delay map[string]time.Duration
}

func newMemoryBus() *memoryBus {
	return &memoryBus{
		nodes: make(map[string]*Coordinator),
		down:  make(map[string]bool),
		delay: make(map[string]time.Duration),
	}
}

func (b *memoryBus) register(addr string, c *Coordinator) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nodes[addr] = c
}

func (b *memoryBus) setDown(addr string, down bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.down[addr] = down
}

func (b *memoryBus) setDelay(addr string, delay time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.delay[addr] = delay
}

func (b *memoryBus) transportFor(self string) Transport {
	return &memoryTransport{bus: b, self: self}
}

type memoryTransport struct {
	bus  *memoryBus
	self string
}

func (t *memoryTransport) peer(addr string) (*Coordinator, error) {
	t.bus.mu.RLock()
	defer t.bus.mu.RUnlock()
	if t.bus.down[addr] {
		return nil, errors.New("peer down")
	}
	delay := t.bus.delay[addr]
	c, ok := t.bus.nodes[addr]
	if !ok {
		return nil, errors.New("peer not found")
	}
	if delay > 0 {
		t.bus.mu.RUnlock()
		time.Sleep(delay)
		t.bus.mu.RLock()
	}
	return c, nil
}

func (t *memoryTransport) SendHeartbeat(ctx context.Context, peer string, hb Heartbeat) (HeartbeatReply, error) {
	c, err := t.peer(peer)
	if err != nil {
		return HeartbeatReply{}, err
	}
	return c.ReceiveHeartbeat(hb), nil
}

func (t *memoryTransport) Replicate(ctx context.Context, peer string, op ReplicatedOperation) error {
	c, err := t.peer(peer)
	if err != nil {
		return err
	}
	_, err = c.ApplyReplicated(op)
	return err
}

func (t *memoryTransport) Forward(ctx context.Context, peer string, op ReplicatedOperation) error {
	c, err := t.peer(peer)
	if err != nil {
		return err
	}
	_, err = c.ProposeReplicated(op)
	return err
}

func (t *memoryTransport) SendSnapshot(ctx context.Context, peer string, snapshotReader io.Reader, lastLogIndex uint64, appliedLogIndex uint64, term uint64) error {
	c, err := t.peer(peer)
	if err != nil {
		return err
	}
	tempFile, err := os.CreateTemp("", "pb_test_snapshot_*.db")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, snapshotReader); err != nil {
		return err
	}
	tempFile.Close()

	return c.InstallSnapshot(tempFile.Name(), lastLogIndex, appliedLogIndex, term)
}

func (t *memoryTransport) Join(ctx context.Context, leader string, selfAddr string) error {
	c, err := t.peer(leader)
	if err != nil {
		return err
	}
	c.AddPeer(selfAddr)
	return nil
}

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	mgr := NewManager(Config{DataDir: t.TempDir(), EmbeddingModel: "embed-model"})
	if err := mgr.Load(); err != nil {
		t.Fatal(err)
	}
	return mgr
}

func TestCoordinatorStandaloneIsLeader(t *testing.T) {
	mgr := newTestManager(t)
	c := NewCoordinator(mgr, CoordinatorConfig{SelfAddr: "http://node-a"})

	if c.HasPeers() {
		t.Fatal("expected no peers in standalone mode")
	}
	if !c.IsLeader() {
		t.Fatal("standalone node must be leader")
	}

	snapshot, err := c.Propose(Operation{
		Type: OperationTypeEnqueueTask,
		Task: &EmbeddingTask{ProjectID: "p1", SourceType: "record", SourceID: "r1", SourceField: "body"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Status.PendingEmbeddings != 1 {
		t.Fatalf("expected 1 pending embedding, got %d", snapshot.Status.PendingEmbeddings)
	}
}

func TestCoordinatorElectsSmallestAddress(t *testing.T) {
	bus := newMemoryBus()

	addrs := []string{"http://node-a", "http://node-b", "http://node-c"}
	coords := make(map[string]*Coordinator)
	for _, addr := range addrs {
		mgr := newTestManager(t)
		peers := make([]string, 0, len(addrs)-1)
		for _, other := range addrs {
			if other != addr {
				peers = append(peers, other)
			}
		}
		c := NewCoordinator(mgr, CoordinatorConfig{
			SelfAddr:  addr,
			Peers:     peers,
			Interval:  20 * time.Millisecond,
			Timeout:   200 * time.Millisecond,
			Transport: bus.transportFor(addr),
		})
		bus.register(addr, c)
		coords[addr] = c
	}

	for _, c := range coords {
		c.Start()
	}
	defer func() {
		for _, c := range coords {
			c.Stop()
		}
	}()

	waitFor(t, func() bool {
		return coords["http://node-a"].IsLeader() &&
			!coords["http://node-b"].IsLeader() &&
			!coords["http://node-c"].IsLeader()
	})

	if got := coords["http://node-b"].View().LeaderID; got != "http://node-a" {
		t.Fatalf("expected leader http://node-a, got %q", got)
	}
}

func TestCoordinatorReplicatesFromLeader(t *testing.T) {
	bus := newMemoryBus()

	mgrA := newTestManager(t)
	mgrB := newTestManager(t)

	ca := NewCoordinator(mgrA, CoordinatorConfig{
		SelfAddr:  "http://node-a",
		Peers:     []string{"http://node-b"},
		Interval:  20 * time.Millisecond,
		Timeout:   200 * time.Millisecond,
		Transport: bus.transportFor("http://node-a"),
	})
	cb := NewCoordinator(mgrB, CoordinatorConfig{
		SelfAddr:  "http://node-b",
		Peers:     []string{"http://node-a"},
		Interval:  20 * time.Millisecond,
		Timeout:   200 * time.Millisecond,
		Transport: bus.transportFor("http://node-b"),
	})
	mgrA.AttachCoordinator(ca)
	mgrB.AttachCoordinator(cb)
	bus.register("http://node-a", ca)
	bus.register("http://node-b", cb)

	ca.Start()
	cb.Start()
	defer ca.Stop()
	defer cb.Stop()

	waitFor(t, func() bool { return ca.IsLeader() && !cb.IsLeader() })

	// enqueue on the leader: should replicate to the follower
	if _, err := ca.Propose(Operation{
		Type: OperationTypeEnqueueTask,
		Task: &EmbeddingTask{ProjectID: "p1", SourceType: "record", SourceID: "r1", SourceField: "body"},
	}); err != nil {
		t.Fatal(err)
	}

	waitFor(t, func() bool {
		return mgrB.Status().PendingEmbeddings == 1
	})

	// enqueue via the follower: should be forwarded to leader and replicated back
	if _, err := cb.Propose(Operation{
		Type: OperationTypeEnqueueTask,
		Task: &EmbeddingTask{ProjectID: "p2", SourceType: "record", SourceID: "r2", SourceField: "body"},
	}); err != nil {
		t.Fatal(err)
	}

	waitFor(t, func() bool {
		return mgrA.Status().PendingEmbeddings == 2 && mgrB.Status().PendingEmbeddings == 2
	})
}

func TestCoordinatorFailoverReelectsLeader(t *testing.T) {
	bus := newMemoryBus()

	addrs := []string{"http://node-a", "http://node-b"}
	coords := make(map[string]*Coordinator)
	for _, addr := range addrs {
		mgr := newTestManager(t)
		peers := make([]string, 0)
		for _, other := range addrs {
			if other != addr {
				peers = append(peers, other)
			}
		}
		c := NewCoordinator(mgr, CoordinatorConfig{
			SelfAddr:  addr,
			Peers:     peers,
			Interval:  20 * time.Millisecond,
			Timeout:   200 * time.Millisecond,
			Transport: bus.transportFor(addr),
		})
		bus.register(addr, c)
		coords[addr] = c
		c.Start()
	}
	defer func() {
		for _, c := range coords {
			c.Stop()
		}
	}()

	waitFor(t, func() bool { return coords["http://node-a"].IsLeader() })

	// take node-a down; node-b should take over
	bus.setDown("http://node-a", true)

	waitFor(t, func() bool {
		return coords["http://node-b"].IsLeader()
	})
}

func TestCoordinatorStrictOperationDoesNotFallbackOnFollower(t *testing.T) {
	bus := newMemoryBus()

	applied := 0
	mgrA := newTestManager(t)
	mgrB := newTestManager(t)

	ca := NewCoordinator(mgrA, CoordinatorConfig{
		SelfAddr:  "http://node-a",
		Peers:     []string{"http://node-b"},
		Interval:  20 * time.Millisecond,
		Timeout:   200 * time.Millisecond,
		Transport: bus.transportFor("http://node-a"),
		Apply: func(op ReplicatedOperation) error {
			applied++
			return nil
		},
	})
	cb := NewCoordinator(mgrB, CoordinatorConfig{
		SelfAddr:  "http://node-b",
		Peers:     []string{"http://node-a"},
		Interval:  20 * time.Millisecond,
		Timeout:   200 * time.Millisecond,
		Transport: bus.transportFor("http://node-b"),
		Apply: func(op ReplicatedOperation) error {
			applied++
			return nil
		},
	})
	bus.register("http://node-a", ca)
	bus.register("http://node-b", cb)

	ca.Start()
	cb.Start()
	defer ca.Stop()
	defer cb.Stop()

	waitFor(t, func() bool { return ca.IsLeader() && !cb.IsLeader() })
	bus.setDown("http://node-a", true)

	_, err := cb.ProposeReplicated(ReplicatedOperation{
		Kind:   ReplicatedOperationKindSQLite,
		Type:   "schema.collection_upsert",
		Strict: true,
	})
	if !errors.Is(err, ErrLeaderUnavailable) {
		t.Fatalf("expected ErrLeaderUnavailable, got %v", err)
	}
	if applied != 0 {
		t.Fatalf("expected strict operation not to fallback apply, got %d applies", applied)
	}
}

func TestCoordinatorStrictOperationRequiresLeaderTransport(t *testing.T) {
	applied := 0
	mgr := newTestManager(t)
	c := NewCoordinator(mgr, CoordinatorConfig{
		SelfAddr: "http://node-b",
		Peers:    []string{"http://node-a"},
		Apply: func(op ReplicatedOperation) error {
			applied++
			return nil
		},
	})

	_, err := c.ProposeReplicated(ReplicatedOperation{
		Kind:   ReplicatedOperationKindSQLite,
		Type:   "schema.collection_upsert",
		Strict: true,
	})
	if !errors.Is(err, ErrLeaderUnavailable) {
		t.Fatalf("expected ErrLeaderUnavailable, got %v", err)
	}
	if applied != 0 {
		t.Fatalf("expected strict operation not to apply without leader transport, got %d applies", applied)
	}
}

func TestCoordinatorStrictOperationWaitsForReplication(t *testing.T) {
	bus := newMemoryBus()

	mgrA := newTestManager(t)
	mgrB := newTestManager(t)

	applyStarted := make(chan struct{})
	releaseApply := make(chan struct{})
	applyDone := make(chan struct{})

	ca := NewCoordinator(mgrA, CoordinatorConfig{
		SelfAddr:  "http://node-a",
		Peers:     []string{"http://node-b"},
		Interval:  20 * time.Millisecond,
		Timeout:   time.Second,
		Transport: bus.transportFor("http://node-a"),
		Apply: func(op ReplicatedOperation) error {
			return nil
		},
	})
	cb := NewCoordinator(mgrB, CoordinatorConfig{
		SelfAddr:  "http://node-b",
		Peers:     []string{"http://node-a"},
		Interval:  20 * time.Millisecond,
		Timeout:   time.Second,
		Transport: bus.transportFor("http://node-b"),
		Apply: func(op ReplicatedOperation) error {
			close(applyStarted)
			<-releaseApply
			close(applyDone)
			return nil
		},
	})
	bus.register("http://node-a", ca)
	bus.register("http://node-b", cb)

	ca.Start()
	cb.Start()
	defer ca.Stop()
	defer cb.Stop()

	waitFor(t, func() bool { return ca.IsLeader() && !cb.IsLeader() })

	proposeDone := make(chan error, 1)
	go func() {
		_, err := ca.ProposeReplicated(ReplicatedOperation{
			Kind:   ReplicatedOperationKindSQLite,
			Type:   "record.create",
			Strict: true,
		})
		proposeDone <- err
	}()

	select {
	case <-applyStarted:
	case <-time.After(time.Second):
		t.Fatal("expected follower apply to start")
	}

	select {
	case err := <-proposeDone:
		t.Fatalf("strict propose returned before follower apply completed: %v", err)
	default:
	}

	close(releaseApply)

	select {
	case <-applyDone:
	case <-time.After(time.Second):
		t.Fatal("expected follower apply to complete")
	}

	select {
	case err := <-proposeDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("expected strict propose to return after follower apply")
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

func TestCoordinatorQuorumStrictCommit(t *testing.T) {
	bus := newMemoryBus()

	mgrA := newTestManager(t)
	mgrB := newTestManager(t)
	mgrC := newTestManager(t)

	applyCount := 0
	applyMu := sync.Mutex{}
	applyFn := func(op ReplicatedOperation) error {
		applyMu.Lock()
		applyCount++
		applyMu.Unlock()
		return nil
	}

	ca := NewCoordinator(mgrA, CoordinatorConfig{
		SelfAddr:  "http://node-a",
		Peers:     []string{"http://node-b", "http://node-c"},
		Interval:  20 * time.Millisecond,
		Timeout:   200 * time.Millisecond,
		Transport: bus.transportFor("http://node-a"),
		Apply:     applyFn,
	})
	cb := NewCoordinator(mgrB, CoordinatorConfig{
		SelfAddr:  "http://node-b",
		Peers:     []string{"http://node-a", "http://node-c"},
		Interval:  20 * time.Millisecond,
		Timeout:   200 * time.Millisecond,
		Transport: bus.transportFor("http://node-b"),
		Apply:     applyFn,
	})
	cc := NewCoordinator(mgrC, CoordinatorConfig{
		SelfAddr:  "http://node-c",
		Peers:     []string{"http://node-a", "http://node-b"},
		Interval:  20 * time.Millisecond,
		Timeout:   200 * time.Millisecond,
		Transport: bus.transportFor("http://node-c"),
		Apply:     applyFn,
	})

	bus.register("http://node-a", ca)
	bus.register("http://node-b", cb)
	bus.register("http://node-c", cc)

	ca.Start()
	cb.Start()
	cc.Start()
	defer ca.Stop()
	defer cb.Stop()
	defer cc.Stop()

	waitFor(t, func() bool { return ca.IsLeader() })

	// Case 1: All nodes online -> Propose should succeed.
	_, err := ca.ProposeReplicated(ReplicatedOperation{
		Kind:   ReplicatedOperationKindSQLite,
		Type:   "schema.collection_upsert",
		Strict: true,
	})
	if err != nil {
		t.Fatalf("expected proposal to succeed with all nodes online, got %v", err)
	}

	// Case 2: One follower offline (node-b) -> Majority (2 out of 3) is still reached.
	bus.setDown("http://node-b", true)
	_, err = ca.ProposeReplicated(ReplicatedOperation{
		Kind:   ReplicatedOperationKindSQLite,
		Type:   "record.create",
		Strict: true,
	})
	if err != nil {
		t.Fatalf("expected proposal to succeed with one follower offline, got %v", err)
	}

	// Case 3: Two followers offline (node-b and node-c) -> Quorum not met, must fail.
	bus.setDown("http://node-c", true)
	_, err = ca.ProposeReplicated(ReplicatedOperation{
		Kind:   ReplicatedOperationKindSQLite,
		Type:   "record.update",
		Strict: true,
	})
	if err == nil {
		t.Fatal("expected proposal to fail when quorum is not met")
	}
}

func TestCoordinatorLogIndexAndGapDetection(t *testing.T) {
	mgr := newTestManager(t)
	c := NewCoordinator(mgr, CoordinatorConfig{
		SelfAddr: "http://node-a",
		Apply: func(op ReplicatedOperation) error {
			return nil
		},
	})

	// Case 1: First operation: index 1.
	_, err := c.ApplyReplicated(ReplicatedOperation{
		LogIndex: 1,
		Kind:     ReplicatedOperationKindSQLite,
		Type:     "record.create",
		Strict:   true,
	})
	if err != nil {
		t.Fatalf("expected apply of log index 1 to succeed, got %v", err)
	}

	// Case 2: Duplicate operation: index 1. Should be ignored and return success.
	_, err = c.ApplyReplicated(ReplicatedOperation{
		LogIndex: 1,
		Kind:     ReplicatedOperationKindSQLite,
		Type:     "record.create",
		Strict:   true,
	})
	if err != nil {
		t.Fatalf("expected duplicate apply to be ignored and succeed, got %v", err)
	}

	// Case 3: Gap detected: index 3 (expected 2 for strict). Should return error.
	_, err = c.ApplyReplicated(ReplicatedOperation{
		LogIndex: 3,
		Kind:     ReplicatedOperationKindSQLite,
		Type:     "record.create",
		Strict:   true,
	})
	if err == nil {
		t.Fatal("expected error due to log index gap detection")
	}

	// Case 4: Correct index: index 2. Should succeed.
	_, err = c.ApplyReplicated(ReplicatedOperation{
		LogIndex: 2,
		Kind:     ReplicatedOperationKindSQLite,
		Type:     "record.create",
		Strict:   true,
	})
	if err != nil {
		t.Fatalf("expected apply of correct log index 2 to succeed, got %v", err)
	}
}

func TestCoordinatorFailureReplay(t *testing.T) {
	bus := newMemoryBus()

	mgrA := newTestManager(t)
	mgrB := newTestManager(t)

	applyCountB := 0
	applyMuB := sync.Mutex{}

	ca := NewCoordinator(mgrA, CoordinatorConfig{
		SelfAddr:  "http://node-a",
		Peers:     []string{"http://node-b"},
		Interval:  20 * time.Millisecond,
		Timeout:   100 * time.Millisecond,
		Transport: bus.transportFor("http://node-a"),
		Apply: func(op ReplicatedOperation) error {
			return nil
		},
	})
	cb := NewCoordinator(mgrB, CoordinatorConfig{
		SelfAddr:  "http://node-b",
		Peers:     []string{"http://node-a"},
		Interval:  20 * time.Millisecond,
		Timeout:   100 * time.Millisecond,
		Transport: bus.transportFor("http://node-b"),
		Apply: func(op ReplicatedOperation) error {
			applyMuB.Lock()
			applyCountB++
			applyMuB.Unlock()
			return nil
		},
	})

	bus.register("http://node-a", ca)
	bus.register("http://node-b", cb)

	ca.Start()
	cb.Start()
	defer ca.Stop()
	defer cb.Stop()

	waitFor(t, func() bool { return ca.IsLeader() })

	// Take node-b down.
	bus.setDown("http://node-b", true)

	// Propose operations on the leader.
	_, err := ca.ProposeReplicated(ReplicatedOperation{
		Kind:   ReplicatedOperationKindSQLite,
		Type:   "schema.collection_upsert",
		Strict: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = ca.ProposeReplicated(ReplicatedOperation{
		Kind:   ReplicatedOperationKindSQLite,
		Type:   "record.create",
		Strict: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Give asynchronous replications some time to execute and fail while node-b is down.
	time.Sleep(50 * time.Millisecond)

	applyMuB.Lock()
	initialBCount := applyCountB
	applyMuB.Unlock()
	if initialBCount != 0 {
		t.Fatalf("expected node-b to have 0 applies since it was down, got %d", initialBCount)
	}

	// Bring node-b back online.
	bus.setDown("http://node-b", false)

	// Wait for background replay loop to automatically detect and replicate the missed logs.
	waitFor(t, func() bool {
		applyMuB.Lock()
		count := applyCountB
		applyMuB.Unlock()
		return count == 2
	})
}

func TestCoordinatorOutdatedLeaderCannotReclaim(t *testing.T) {
	bus := newMemoryBus()

	mgrA := newTestManager(t)
	mgrB := newTestManager(t)

	ca := NewCoordinator(mgrA, CoordinatorConfig{
		SelfAddr:  "http://node-a",
		Peers:     []string{"http://node-b"},
		Interval:  20 * time.Millisecond,
		Timeout:   100 * time.Millisecond,
		Transport: bus.transportFor("http://node-a"),
	})
	cb := NewCoordinator(mgrB, CoordinatorConfig{
		SelfAddr:  "http://node-b",
		Peers:     []string{"http://node-a"},
		Interval:  20 * time.Millisecond,
		Timeout:   100 * time.Millisecond,
		Transport: bus.transportFor("http://node-b"),
	})
	mgrA.AttachCoordinator(ca)
	mgrB.AttachCoordinator(cb)
	bus.register("http://node-a", ca)
	bus.register("http://node-b", cb)

	ca.Start()
	cb.Start()
	defer ca.Stop()
	defer cb.Stop()

	// Initially, http://node-a should be the leader (smaller address)
	waitFor(t, func() bool { return ca.IsLeader() && !cb.IsLeader() })

	// Bring node-a down
	bus.setDown("http://node-a", true)

	// Wait for node-b to detect failure and elect itself leader
	waitFor(t, func() bool { return cb.IsLeader() })

	// Propose 2 operations on node-b so its index increments to 2
	_, err := cb.ProposeReplicated(ReplicatedOperation{
		Kind:   ReplicatedOperationKindSQLite,
		Type:   "record.create",
		Strict: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = cb.ProposeReplicated(ReplicatedOperation{
		Kind:   ReplicatedOperationKindSQLite,
		Type:   "record.create",
		Strict: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	if cb.LastLogIndex() != 2 {
		t.Fatalf("expected node-b to have last log index 2, got %d", cb.LastLogIndex())
	}

	// Bring node-a back online but with delayed transport to node-a
	// so that we can deterministically observe node-a's state when it is outdated
	bus.setDelay("http://node-a", 200*time.Millisecond)
	bus.setDown("http://node-a", false)

	// Wait until node-a receives a heartbeat from node-b (with last index 2),
	// but before node-a catches up (since replication to node-a is delayed).
	waitFor(t, func() bool {
		ca.mu.RLock()
		peerBState := ca.states["http://node-b"]
		knownBIndex := uint64(0)
		if peerBState != nil && peerBState.Alive {
			knownBIndex = peerBState.LastLogIndex
		}
		ownIndex := ca.lastLogIndex
		ca.mu.RUnlock()
		return knownBIndex == 2 && ownIndex == 0
	})

	if ca.IsLeader() {
		t.Fatal("outdated node-a should not have reclaimed leadership")
	}
	if !cb.IsLeader() {
		t.Fatal("node-b should have remained leader because it is the up-to-date node")
	}

	// Now clear the delay to let node-a catch up
	bus.setDelay("http://node-a", 0)

	// Wait for the background replays to bring node-a up-to-date
	waitFor(t, func() bool {
		return ca.LastLogIndex() == 2
	})

	// Once node-a is up-to-date (index 2), it can safely reclaim leadership!
	waitFor(t, func() bool {
		return ca.IsLeader() && !cb.IsLeader()
	})
}

func TestCoordinatorHeartbeatLearnsMembersAndProgress(t *testing.T) {
	mgr := newTestManager(t)
	c := NewCoordinator(mgr, CoordinatorConfig{
		SelfAddr: "http://node-a",
		Peers:    []string{"http://node-b"},
	})

	c.mu.Lock()
	c.lastLogIndex = 5
	c.appliedLogIndex = 5
	c.peerNextIndex["http://node-b"] = 6
	c.mu.Unlock()

	c.ReceiveHeartbeat(Heartbeat{
		Term:         1,
		NodeID:       "node-b",
		Address:      "http://node-b",
		Members:      []string{"http://node-a", "http://node-b", "http://node-c"},
		LastLogIndex: 2,
	})

	c.mu.RLock()
	defer c.mu.RUnlock()

	if !containsString(c.peers, "http://node-c") {
		t.Fatal("expected coordinator to learn http://node-c from heartbeat members")
	}
	if c.states["http://node-c"] == nil {
		t.Fatal("expected learned member state to be initialized")
	}
	if got := c.peerNextIndex["http://node-b"]; got != 3 {
		t.Fatalf("expected node-b next index to track heartbeat progress, got %d", got)
	}
	if got := c.peerNextIndex["http://node-c"]; got != 1 {
		t.Fatalf("expected new member catch-up to start at index 1, got %d", got)
	}
}

func TestCoordinatorAddPeerStartsCatchupFromBeginning(t *testing.T) {
	mgr := newTestManager(t)
	c := NewCoordinator(mgr, CoordinatorConfig{SelfAddr: "http://node-a"})

	c.mu.Lock()
	c.lastLogIndex = 10
	c.appliedLogIndex = 10
	c.mu.Unlock()

	c.AddPeer("http://node-b")

	c.mu.RLock()
	defer c.mu.RUnlock()

	if !containsString(c.peers, "http://node-b") {
		t.Fatal("expected peer to be added")
	}
	if c.states["http://node-b"] == nil {
		t.Fatal("expected peer state to be initialized")
	}
	if got := c.peerNextIndex["http://node-b"]; got != 1 {
		t.Fatalf("expected new peer to start catch-up at index 1, got %d", got)
	}
}

// mockSnapshotApp is accessed from the coordinator's background snapshot
// goroutine as well as from the test goroutine, so its counters are atomic.
type mockSnapshotApp struct {
	createSnapshotCount atomic.Int64
	reloadDbCount       atomic.Int64

	mu           sync.Mutex
	installedIdx uint64
	installedSet bool
}

func (m *mockSnapshotApp) CreateConsistentSnapshot(targetPath string) error {
	m.createSnapshotCount.Add(1)
	// Create a dummy file
	return os.WriteFile(targetPath, []byte("sqlite-snapshot-data"), 0644)
}

func (m *mockSnapshotApp) ReloadDataDBWithReplacement(replacePath string) error {
	m.reloadDbCount.Add(1)
	return nil
}

func (m *mockSnapshotApp) createSnapshotCalled() bool {
	return m.createSnapshotCount.Load() > 0
}

func (m *mockSnapshotApp) reloadDbCalled() bool {
	return m.reloadDbCount.Load() > 0
}

// recordInstall mirrors the watermark bookkeeping InstallSnapshot performs so
// tests can assert on the resulting state machine position.
func (m *mockSnapshotApp) recordInstall(appliedIdx uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.installedIdx = appliedIdx
	m.installedSet = true
}

func TestCoordinatorSnapshotGapAndJoin(t *testing.T) {
	bus := newMemoryBus()

	mgrA := newTestManager(t)
	mgrB := newTestManager(t)

	appA := &mockSnapshotApp{}
	appB := &mockSnapshotApp{}

	ca := NewCoordinator(mgrA, CoordinatorConfig{
		SelfAddr:  "http://node-a",
		Peers:     []string{"http://node-b"},
		Interval:  20 * time.Millisecond,
		Timeout:   100 * time.Millisecond,
		Transport: bus.transportFor("http://node-a"),
		App:       appA,
	})
	cb := NewCoordinator(mgrB, CoordinatorConfig{
		SelfAddr:  "http://node-b",
		Peers:     []string{"http://node-a"},
		Interval:  20 * time.Millisecond,
		Timeout:   100 * time.Millisecond,
		Transport: bus.transportFor("http://node-b"),
		App:       appB,
	})

	bus.register("http://node-a", ca)
	bus.register("http://node-b", cb)

	ca.Start()
	cb.Start()
	defer ca.Stop()
	defer cb.Stop()

	waitFor(t, func() bool { return ca.IsLeader() })

	// Case 1: Dynamic Join
	ca.AddPeer("http://node-c")
	ca.mu.RLock()
	hasNodeC := false
	for _, p := range ca.peers {
		if p == "http://node-c" {
			hasNodeC = true
		}
	}
	ca.mu.RUnlock()
	if !hasNodeC {
		t.Fatal("expected http://node-c to be added dynamically")
	}

	// Case 2: Log Gap Snapshot Trigger
	ca.mu.Lock()
	ca.peerNextIndex["http://node-b"] = 1
	ca.appliedLogIndex = 10
	ca.lastLogIndex = 10
	ca.logs = nil // Empty logs forces snapshot trigger!
	ca.mu.Unlock()

	// Wait for background retry failed replications loop to detect the gap and send the snapshot!
	waitFor(t, func() bool {
		return appA.createSnapshotCalled()
	})

	// Also verify that on the other side, the snapshot gets loaded
	waitFor(t, func() bool {
		return appB.reloadDbCalled()
	})

	// And the follower's next index should be updated to lastLogIndex + 1 = 11
	waitFor(t, func() bool {
		ca.mu.RLock()
		nextIdx := ca.peerNextIndex["http://node-b"]
		ca.mu.RUnlock()
		return nextIdx == 11
	})
}

func TestCoordinatorRejectsStaleSnapshot(t *testing.T) {
	mgr := newTestManager(t)
	app := &mockSnapshotApp{}
	c := NewCoordinator(mgr, CoordinatorConfig{
		SelfAddr: "http://node-a",
		App:      app,
		Apply: func(op ReplicatedOperation) error {
			return nil
		},
	})

	for i := uint64(1); i <= 2; i++ {
		if _, err := c.ApplyReplicated(ReplicatedOperation{
			LogIndex: i,
			Kind:     ReplicatedOperationKindSQLite,
			Type:     "record.create",
			Strict:   true,
		}); err != nil {
			t.Fatal(err)
		}
	}

	err := c.InstallSnapshot("ignored.db", 1, 1, c.RaftTerm())
	if err == nil {
		t.Fatal("expected stale snapshot to be rejected")
	}
	if app.reloadDbCalled() {
		t.Fatal("stale snapshot must not replace the database")
	}
	if got := c.AppliedLogIndex(); got != 2 {
		t.Fatalf("expected applied log index to remain 2, got %d", got)
	}
}

func TestCoordinatorSnapshotNeverLowersWatermarks(t *testing.T) {
	mgr := newTestManager(t)
	app := &mockSnapshotApp{}
	c := NewCoordinator(mgr, CoordinatorConfig{
		SelfAddr: "http://node-a",
		App:      app,
		Apply: func(op ReplicatedOperation) error {
			return nil
		},
	})

	for i := uint64(1); i <= 5; i++ {
		if _, err := c.ApplyReplicated(ReplicatedOperation{
			LogIndex: i,
			Kind:     ReplicatedOperationKindSQLite,
			Type:     "record.create",
			Strict:   true,
		}); err != nil {
			t.Fatal(err)
		}
	}

	startTerm := c.RaftTerm()

	// A snapshot whose applied index moves forward but whose last-log watermark is
	// lower than ours must not drag lastLogIndex backwards, otherwise already-held
	// indexes become re-proposable.
	if err := c.InstallSnapshot("snap.db", 6, 6, startTerm+1); err != nil {
		t.Fatalf("expected newer snapshot to be accepted: %v", err)
	}
	if !app.reloadDbCalled() {
		t.Fatal("accepted snapshot must replace the database")
	}
	if got := c.AppliedLogIndex(); got != 6 {
		t.Fatalf("expected applied log index 6, got %d", got)
	}
	if got := c.LastLogIndex(); got != 6 {
		t.Fatalf("expected last log index 6, got %d", got)
	}
	if got := c.RaftTerm(); got != startTerm+1 {
		t.Fatalf("expected term %d, got %d", startTerm+1, got)
	}

	// An older term that carries no log beyond ours is rejected outright.
	before := app.reloadDbCount.Load()
	if err := c.InstallSnapshot("old.db", 6, 6, startTerm); err == nil {
		t.Fatal("expected same-position snapshot from an older term to be rejected")
	}
	if got := app.reloadDbCount.Load(); got != before {
		t.Fatal("rejected snapshot must not replace the database")
	}

	// But per raft an older term is still installed when it fills in committed
	// entries we are missing, as long as it does not rewind the state machine.
	if err := c.InstallSnapshot("fill.db", 9, 7, startTerm); err != nil {
		t.Fatalf("expected older-term snapshot with newer log to be accepted: %v", err)
	}
	if got := c.AppliedLogIndex(); got != 7 {
		t.Fatalf("expected applied log index 7, got %d", got)
	}
	if got := c.LastLogIndex(); got != 9 {
		t.Fatalf("expected last log index 9, got %d", got)
	}
	if got := c.RaftTerm(); got != startTerm+1 {
		t.Fatalf("expected term to stay at %d, got %d", startTerm+1, got)
	}

	// Malformed watermarks (last < applied) are rejected before any I/O.
	reloadBefore := app.reloadDbCount.Load()
	if err := c.InstallSnapshot("bad.db", 2, 8, startTerm+2); err == nil {
		t.Fatal("expected snapshot with lastLogIndex < appliedLogIndex to be rejected")
	}
	if got := app.reloadDbCount.Load(); got != reloadBefore {
		t.Fatal("malformed snapshot must not replace the database")
	}
	if got := c.AppliedLogIndex(); got != 7 {
		t.Fatalf("expected applied log index to remain 7, got %d", got)
	}
}
