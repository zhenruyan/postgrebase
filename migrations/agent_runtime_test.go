package migrations

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/zhenruyan/postgrebase/dbx"
	_ "modernc.org/sqlite"
)

func TestAgentRuntimeMySQLTypes(t *testing.T) {
	if got := agentStringType("mysql"); got != "VARCHAR(255)" {
		t.Fatalf("expected mysql agent string type VARCHAR(255), got %q", got)
	}
	if got := agentLongTextType("mysql"); got != "LONGTEXT" {
		t.Fatalf("expected mysql agent long text type LONGTEXT, got %q", got)
	}
	if got := agentJsonType("mysql"); got != "JSON" {
		t.Fatalf("expected mysql agent json type JSON, got %q", got)
	}
}

// agentRuntimeMigration returns the compiled migration for the agent runtime tables.
func agentRuntimeMigration(t *testing.T) func(dbx.Builder) error {
	t.Helper()

	for _, m := range AppMigrations.Items() {
		if m.File == "1730000000_agent_runtime.go" {
			return m.Up
		}
	}
	t.Fatal("1730000000_agent_runtime.go migration is not registered")
	return nil
}

func newMigrationSQLite(t *testing.T) dbx.Builder {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	return dbx.NewFromDB(sqlDB, "sqlite")
}

// TestAgentRuntimeMigrationExecutes runs the real migration against SQLite so
// that DDL and index typos are caught, not only the type helpers.
func TestAgentRuntimeMigrationExecutes(t *testing.T) {
	db := newMigrationSQLite(t)

	if err := agentRuntimeMigration(t)(db); err != nil {
		t.Fatalf("migration up failed: %v", err)
	}

	tables := []string{
		"_pb_agent_sessions_",
		"_pb_agent_messages_",
		"_pb_agent_audit_",
		"_pb_agent_project_configs_",
	}
	for _, table := range tables {
		var count int
		if err := db.NewQuery(
			"SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = {:name}",
		).Bind(dbx.Params{"name": table}).Row(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("expected table %s to exist", table)
		}
	}

	indexes := map[string]bool{
		"idx_agent_sessions_project": false,
		"idx_agent_messages_session": false,
		"idx_agent_audit_session":    false,
		"idx_agent_project_config":   true,
	}
	for name, unique := range indexes {
		var sqlText string
		if err := db.NewQuery(
			"SELECT sql FROM sqlite_master WHERE type = 'index' AND name = {:name}",
		).Bind(dbx.Params{"name": name}).Row(&sqlText); err != nil {
			t.Fatalf("expected index %s to exist: %v", name, err)
		}
		if unique && !strings.Contains(strings.ToUpper(sqlText), "UNIQUE") {
			t.Fatalf("expected index %s to be UNIQUE, got %q", name, sqlText)
		}
	}

	// Long-text columns must accept content beyond a narrow varchar limit.
	if _, err := db.NewQuery(
		"INSERT INTO {{_pb_agent_messages_}} ([[id]], [[session_id]], [[role]], [[content]], [[created]], [[updated]]) " +
			"VALUES ('m1', 's1', 'assistant', {:content}, '2026-01-01 00:00:00.000', '2026-01-01 00:00:00.000')",
	).Bind(dbx.Params{"content": longText()}).Execute(); err != nil {
		t.Fatalf("failed to insert long message content: %v", err)
	}

	// Re-running must be idempotent (IF NOT EXISTS + guarded index creation).
	if err := agentRuntimeMigration(t)(db); err != nil {
		t.Fatalf("migration up is not idempotent: %v", err)
	}
}

func longText() string {
	return strings.Repeat("a", 70000)
}

// TestAlignAgentColumnTypesIsNoOpForNonMySQL documents that the legacy-column
// alignment pass only applies to MySQL.
func TestAlignAgentColumnTypesIsNoOpForNonMySQL(t *testing.T) {
	db := newMigrationSQLite(t)
	if err := alignAgentColumnTypes(db, "sqlite"); err != nil {
		t.Fatalf("expected no-op for sqlite, got %v", err)
	}
	if err := alignAgentColumnTypes(db, "postgres"); err != nil {
		t.Fatalf("expected no-op for postgres, got %v", err)
	}
}
