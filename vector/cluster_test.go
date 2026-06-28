package vector

import (
	"context"
	"errors"
	"sync"
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
