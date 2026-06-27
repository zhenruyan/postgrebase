package vector

import (
	"path/filepath"
	"testing"
)

func TestFileEngineRoundTrip(t *testing.T) {
	dir := t.TempDir()
	engine := NewFileEngine(dir)

	snapshot := Snapshot{
		Status: Status{
			NodeID:         "node-1",
			EmbeddingModel: "embed-model",
		},
		Tasks: []EmbeddingTask{
			{
				Id:          "task-1",
				ProjectID:   "project-1",
				SourceType:  "record",
				SourceID:    "record-1",
				SourceField: "body",
				Model:       "embed-model",
			},
		},
	}

	if err := engine.Save(snapshot); err != nil {
		t.Fatal(err)
	}

	loaded, ok, err := engine.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected snapshot to exist")
	}

	if loaded.Status.NodeID != snapshot.Status.NodeID {
		t.Fatalf("expected node id %q, got %q", snapshot.Status.NodeID, loaded.Status.NodeID)
	}
	if len(loaded.Tasks) != 1 || loaded.Tasks[0].Id != "task-1" {
		t.Fatalf("unexpected loaded tasks: %#v", loaded.Tasks)
	}

	if _, err := filepath.Glob(filepath.Join(dir, "vector-state.json")); err != nil {
		t.Fatal(err)
	}
}

func TestApplyOperationEnqueue(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(Config{DataDir: dir, EmbeddingModel: "embed-model"})
	if err := mgr.Load(); err != nil {
		t.Fatal(err)
	}

	snapshot, err := mgr.ApplyOperation(Operation{
		Type: OperationTypeEnqueueTask,
		Task: &EmbeddingTask{
			ProjectID:   "project-1",
			SourceType:  "record",
			SourceID:    "record-1",
			SourceField: "body",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if snapshot.Status.PendingEmbeddings != 1 {
		t.Fatalf("expected 1 pending embedding, got %d", snapshot.Status.PendingEmbeddings)
	}
	if len(snapshot.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(snapshot.Tasks))
	}
	if snapshot.Tasks[0].Model != "embed-model" {
		t.Fatalf("expected model embed-model, got %q", snapshot.Tasks[0].Model)
	}
}

func TestLoadKeepsDefaultsWithoutSnapshot(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(Config{DataDir: dir, EmbeddingModel: "embed-model"})
	if err := mgr.Load(); err != nil {
		t.Fatal(err)
	}

	if got := mgr.Status().EmbeddingModel; got != "embed-model" {
		t.Fatalf("expected embedding model to stay set, got %q", got)
	}
}
