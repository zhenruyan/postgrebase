package agents

import (
	"path/filepath"
	"testing"

	"github.com/zhenruyan/postgrebase/core"
)

func newTestApp(t *testing.T) *core.BaseApp {
	t.Helper()
	dataDir := filepath.Join(t.TempDir(), "pb_data")
	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir:       dataDir,
		DataDsn:       "sqlite://" + filepath.Join(dataDir, "test.db"),
		DisableVector: true,
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.ResetBootstrapState() })

	if err := runMigrationsForTest(app); err != nil {
		t.Fatal(err)
	}
	if err := app.RefreshSettings(); err != nil {
		t.Fatal(err)
	}
	return app
}

func TestDBSessionStorePersistence(t *testing.T) {
	app := newTestApp(t)
	store := NewDBSessionStore(app)

	// create a session and ensure it is retrievable after a fresh store
	sess := store.Create("proj-1", "", "openai-main", "gpt-4o")
	if sess.Id == "" || !isPlaceholderName(sess.Name) {
		t.Fatalf("unexpected created session: %+v", sess)
	}

	store2 := NewDBSessionStore(app)
	got, err := store2.Get(sess.Id)
	if err != nil {
		t.Fatalf("session not persisted across stores: %v", err)
	}
	if got.Project != "proj-1" || got.Provider != "openai-main" {
		t.Fatalf("session fields not persisted: %+v", got)
	}

	// add messages (text + image) and read them back
	if _, _, err := store2.AddMessage(sess.Id, "user", "build a sales table"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store2.AddMessageWithImages(sess.Id, "user", "see image", []SessionImage{{MimeType: "image/png", Data: "QUJD"}}); err != nil {
		t.Fatal(err)
	}

	msgs, err := NewDBSessionStore(app).Messages(sess.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 persisted messages, got %d", len(msgs))
	}
	if len(msgs[1].Images) != 1 || msgs[1].Images[0].Data != "QUJD" {
		t.Fatalf("image attachment not persisted: %+v", msgs[1].Images)
	}

	// auto-name lifecycle across stores
	if !NewDBSessionStore(app).NeedsAutoName(sess.Id) {
		t.Fatal("session should need auto-name after first user message")
	}
	if _, err := store2.SetGeneratedName(sess.Id, "Sales Table Setup"); err != nil {
		t.Fatal(err)
	}
	if NewDBSessionStore(app).NeedsAutoName(sess.Id) {
		t.Fatal("session should not need auto-name after generation")
	}

	// rename locks the name
	renamed, err := store2.Rename(sess.Id, "Custom")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Name != "Custom" || !renamed.NameLocked {
		t.Fatalf("rename not persisted/locked: %+v", renamed)
	}

	// listing returns the session for its project
	list := NewDBSessionStore(app).List("proj-1")
	if len(list) != 1 || list[0].Id != sess.Id {
		t.Fatalf("session list mismatch: %+v", list)
	}
}

func TestAgentAuditPersistence(t *testing.T) {
	app := newTestApp(t)
	svc := NewService(app)

	sess := svc.CreateSession("proj-1", "", "openai-main", "gpt-4o")
	if sess == nil {
		t.Fatal("failed to create session")
	}

	entries := []AgentAuditEntry{
		{Session: sess.Id, Project: "proj-1", Tool: "data.query", Category: "read", Decision: "allow", Status: "ok"},
		{Session: sess.Id, Project: "proj-1", Tool: "data.delete", Category: "write", Decision: "deny", Status: "pending", Reason: "needs approval"},
	}
	svc.persistAudit(sess.Id, "proj-1", entries)

	records, err := svc.SessionAudit(sess.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 audit records, got %d", len(records))
	}

	var sawDeny bool
	for _, r := range records {
		if r.Tool == "data.delete" && r.Decision == "deny" && r.Status == "pending" {
			sawDeny = true
		}
	}
	if !sawDeny {
		t.Fatal("deny/pending audit record not persisted")
	}
}
