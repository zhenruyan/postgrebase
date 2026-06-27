package agents

import "testing"

func TestSessionStore(t *testing.T) {
	store := NewSessionStore()
	session := store.Create("project-1", "", "openai-main", "gpt-4.1")
	if session == nil {
		t.Fatal("expected session")
	}
	if session.Project != "project-1" {
		t.Fatalf("unexpected project: %s", session.Project)
	}
	if session.Name == "" {
		t.Fatal("expected generated name")
	}

	updated, msgs, err := store.AddMessage(session.Id, "user", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if updated.LastMessage != "hello" {
		t.Fatalf("unexpected last message: %s", updated.LastMessage)
	}
	if len(msgs) != 1 || msgs[0].Content != "hello" {
		t.Fatalf("unexpected messages: %#v", msgs)
	}

	loaded, err := store.Messages(session.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].Content != "hello" {
		t.Fatalf("unexpected loaded messages: %#v", loaded)
	}
}

func TestToolRegistry(t *testing.T) {
	reg := NewToolRegistry()
	tools := reg.List()
	if len(tools) == 0 {
		t.Fatal("expected tools")
	}
	if _, ok := reg.Get("data.query"); !ok {
		t.Fatal("expected data.query tool")
	}
}
