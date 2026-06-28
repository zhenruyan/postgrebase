package agents

import "testing"

func TestSessionAutoNameLifecycle(t *testing.T) {
	store := NewSessionStore()

	// session created without a name => placeholder, eligible for auto-naming
	sess := store.Create("p1", "", "openai-main", "gpt-4o")
	if !isPlaceholderName(sess.Name) {
		t.Fatalf("expected placeholder name, got %q", sess.Name)
	}
	if sess.NameLocked {
		t.Fatal("auto-created session should not be name-locked")
	}

	// no user message yet => not eligible
	if store.NeedsAutoName(sess.Id) {
		t.Fatal("should not need auto-name before any user message")
	}

	// add a user message => eligible
	if _, _, err := store.AddMessage(sess.Id, "user", "build me a sales table"); err != nil {
		t.Fatal(err)
	}
	if !store.NeedsAutoName(sess.Id) {
		t.Fatal("should need auto-name after first user message")
	}

	// generate once
	updated, err := store.SetGeneratedName(sess.Id, "Sales Table Setup")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Sales Table Setup" {
		t.Fatalf("name not set, got %q", updated.Name)
	}

	// no longer eligible (single generation guarantee)
	if store.NeedsAutoName(sess.Id) {
		t.Fatal("should not need auto-name after generation")
	}

	// a second generation attempt is a no-op
	again, _ := store.SetGeneratedName(sess.Id, "Different Name")
	if again.Name != "Sales Table Setup" {
		t.Fatalf("name should not change on second generation, got %q", again.Name)
	}
}

func TestSessionRenameLocks(t *testing.T) {
	store := NewSessionStore()
	sess := store.Create("p1", "", "", "")
	_, _, _ = store.AddMessage(sess.Id, "user", "hi")

	renamed, err := store.Rename(sess.Id, "My Custom Name")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Name != "My Custom Name" || !renamed.NameLocked {
		t.Fatalf("rename should set and lock name, got %q locked=%v", renamed.Name, renamed.NameLocked)
	}
	if store.NeedsAutoName(sess.Id) {
		t.Fatal("renamed session must not be auto-named")
	}

	// user-provided name on create is locked
	named := store.Create("p1", "Initial", "", "")
	if !named.NameLocked {
		t.Fatal("user-named session should be locked on create")
	}
}

func TestSanitizeTitle(t *testing.T) {
	cases := map[string]string{
		"  Sales Report Dashboard  ": "Sales Report Dashboard",
		"\"Quoted Title\"":           "Quoted Title",
		"Title line\nsecond line":    "Title line",
		"Ends with period.":          "Ends with period",
		"":                           "",
	}
	for in, want := range cases {
		if got := sanitizeTitle(in); got != want {
			t.Errorf("sanitizeTitle(%q) = %q, want %q", in, got, want)
		}
	}
}
