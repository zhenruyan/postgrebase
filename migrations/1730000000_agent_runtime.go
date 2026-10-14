package migrations

import (
	"fmt"
	"strings"

	"github.com/zhenruyan/postgrebase/dbx"
)

func init() {
	AppMigrations.Register(func(db dbx.Builder) error {
		driver := db.DriverName()

		sessionTable := `
			CREATE TABLE IF NOT EXISTS {{_pb_agent_sessions_}} (
				[[id]]           ` + agentIdType(driver) + ` NOT NULL PRIMARY KEY,
				[[project_id]]   ` + agentStringType(driver) + ` NOT NULL,
				[[name]]         ` + agentStringType(driver) + ` NOT NULL,
				[[provider]]     ` + agentStringType(driver) + ` NOT NULL,
				[[model]]        ` + agentStringType(driver) + ` NOT NULL,
				[[name_locked]]  ` + agentBoolType(driver) + ` NOT NULL DEFAULT ` + agentBoolDefault(driver) + `,
				[[last_message]] ` + agentLongTextType(driver) + ` NOT NULL,
				[[created]]      ` + agentTsType(driver) + ` NOT NULL,
				[[updated]]      ` + agentTsType(driver) + ` NOT NULL
			);`

		messageTable := `
			CREATE TABLE IF NOT EXISTS {{_pb_agent_messages_}} (
				[[id]]         ` + agentIdType(driver) + ` NOT NULL PRIMARY KEY,
				[[session_id]] ` + agentStringType(driver) + ` NOT NULL,
				[[role]]       ` + agentStringType(driver) + ` NOT NULL,
				[[content]]    ` + agentLongTextType(driver) + ` NOT NULL,
				[[images]]     ` + agentJsonType(driver) + `,
				[[created]]    ` + agentTsType(driver) + ` NOT NULL,
				[[updated]]    ` + agentTsType(driver) + ` NOT NULL
			);`

		auditTable := `
			CREATE TABLE IF NOT EXISTS {{_pb_agent_audit_}} (
				[[id]]             ` + agentIdType(driver) + ` NOT NULL PRIMARY KEY,
				[[session_id]]     ` + agentStringType(driver) + ` NOT NULL,
				[[project_id]]     ` + agentStringType(driver) + ` NOT NULL,
				[[actor]]          ` + agentStringType(driver) + ` NOT NULL DEFAULT '',
				[[tool]]           ` + agentStringType(driver) + ` NOT NULL,
				[[category]]       ` + agentStringType(driver) + ` NOT NULL DEFAULT '',
				[[risk]]           ` + agentStringType(driver) + ` NOT NULL DEFAULT '',
				[[audit_category]] ` + agentStringType(driver) + ` NOT NULL DEFAULT '',
				[[decision]]       ` + agentStringType(driver) + ` NOT NULL DEFAULT '',
				[[reason]]         ` + agentLongTextType(driver) + ` NOT NULL,
				[[status]]         ` + agentStringType(driver) + ` NOT NULL DEFAULT '',
				[[error_msg]]      ` + agentLongTextType(driver) + ` NOT NULL,
				[[created]]        ` + agentTsType(driver) + ` NOT NULL,
				[[updated]]        ` + agentTsType(driver) + ` NOT NULL
			);`

		stmts := []string{sessionTable, messageTable, auditTable}
		stmts = append(stmts,
			`CREATE TABLE IF NOT EXISTS {{_pb_agent_project_configs_}} (
				[[id]]                  `+agentIdType(driver)+` NOT NULL PRIMARY KEY,
				[[project_id]]          `+agentStringType(driver)+` NOT NULL,
				[[default_provider]]    `+agentStringType(driver)+` NOT NULL DEFAULT '',
				[[default_model]]       `+agentStringType(driver)+` NOT NULL DEFAULT '',
				[[allowed_tools]]       `+agentJsonType(driver)+`,
				[[allow_schema_change]] `+agentStringType(driver)+` NOT NULL DEFAULT 'inherit',
				[[approval_policy]]     `+agentStringType(driver)+` NOT NULL DEFAULT 'inherit',
				[[created]]             `+agentTsType(driver)+` NOT NULL,
				[[updated]]             `+agentTsType(driver)+` NOT NULL
			);`,
		)

		for _, stmt := range stmts {
			if _, err := db.NewQuery(stmt).Execute(); err != nil {
				return err
			}
		}

		indexes := []agentIndex{
			{name: "idx_agent_sessions_project", table: "_pb_agent_sessions_", columns: []string{"project_id"}},
			{name: "idx_agent_messages_session", table: "_pb_agent_messages_", columns: []string{"session_id"}},
			{name: "idx_agent_audit_session", table: "_pb_agent_audit_", columns: []string{"session_id"}},
			{name: "idx_agent_project_config", table: "_pb_agent_project_configs_", columns: []string{"project_id"}, unique: true},
		}
		for _, idx := range indexes {
			if err := createAgentIndex(db, idx); err != nil {
				return err
			}
		}

		// The tables above are created with IF NOT EXISTS, so installations that ran
		// an earlier revision of this migration keep the old TEXT columns. Align
		// them now instead of letting schemas silently drift.
		if err := alignAgentColumnTypes(db, driver); err != nil {
			return err
		}

		return nil
	}, func(db dbx.Builder) error {
		for _, table := range []string{"_pb_agent_project_configs_", "_pb_agent_audit_", "_pb_agent_messages_", "_pb_agent_sessions_"} {
			if _, err := db.NewQuery("DROP TABLE IF EXISTS {{" + table + "}}").Execute(); err != nil {
				return err
			}
		}
		return nil
	})
}

func agentIdType(driver string) string {
	if driver == "mysql" {
		return "VARCHAR(36)"
	}
	return "text"
}

func agentStringType(driver string) string {
	if driver == "mysql" {
		return "VARCHAR(255)"
	}
	return "text"
}

func agentLongTextType(driver string) string {
	if driver == "mysql" {
		return "LONGTEXT"
	}
	return "text"
}

func agentJsonType(driver string) string {
	if driver == "mysql" {
		return "JSON"
	}
	return "text"
}

// alignAgentColumnTypes narrows legacy TEXT columns to the current types on
// MySQL. It is a no-op on other drivers (SQLite/Postgres use text throughout)
// and idempotent: columns already matching the target definition are skipped by
// comparing information_schema.columns.COLUMN_TYPE.
func alignAgentColumnTypes(db dbx.Builder, driver string) error {
	if driver != "mysql" {
		return nil
	}

	columns := []struct {
		table  string
		column string
		def    string // full MODIFY definition (type + nullability + default)
	}{
		{"_pb_agent_sessions_", "project_id", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_sessions_", "name", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_sessions_", "provider", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_sessions_", "model", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_sessions_", "last_message", agentLongTextType(driver) + " NOT NULL"},
		{"_pb_agent_messages_", "session_id", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_messages_", "role", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_messages_", "content", agentLongTextType(driver) + " NOT NULL"},
		{"_pb_agent_audit_", "session_id", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_audit_", "project_id", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_audit_", "actor", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_audit_", "tool", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_audit_", "category", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_audit_", "risk", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_audit_", "audit_category", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_audit_", "decision", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_audit_", "reason", agentLongTextType(driver) + " NOT NULL"},
		{"_pb_agent_audit_", "status", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_audit_", "error_msg", agentLongTextType(driver) + " NOT NULL"},
		{"_pb_agent_project_configs_", "project_id", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_project_configs_", "default_provider", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_project_configs_", "default_model", agentStringType(driver) + " NOT NULL"},
		{"_pb_agent_project_configs_", "allow_schema_change", agentStringType(driver) + " NOT NULL DEFAULT 'inherit'"},
		{"_pb_agent_project_configs_", "approval_policy", agentStringType(driver) + " NOT NULL DEFAULT 'inherit'"},
	}

	for _, col := range columns {
		var current string
		err := db.NewQuery(`
			SELECT COLUMN_TYPE
			FROM information_schema.columns
			WHERE table_schema = DATABASE()
				AND table_name = {:table}
				AND column_name = {:column}
		`).Bind(dbx.Params{
			"table":  col.table,
			"column": col.column,
		}).Row(&current)
		if err != nil {
			// table/column not present (fresh install already has the right type)
			continue
		}

		// COLUMN_TYPE reports just the type (e.g. "varchar(255)", "longtext"),
		// while col.def carries type + nullability + default, so compare on the
		// leading type token only.
		desired := strings.ToLower(strings.Fields(col.def)[0])
		if strings.EqualFold(current, desired) {
			continue
		}

		stmt := fmt.Sprintf("ALTER TABLE {{%s}} MODIFY [[%s]] %s", col.table, col.column, col.def)
		if _, err := db.NewQuery(stmt).Execute(); err != nil {
			return fmt.Errorf("failed to align %s.%s to %s: %w", col.table, col.column, col.def, err)
		}
	}

	return nil
}

func agentBoolType(driver string) string {
	switch driver {
	case "mysql":
		return "TINYINT(1)"
	case "sqlite", "sqlite3":
		return "INTEGER"
	default:
		return "BOOLEAN"
	}
}

func agentBoolDefault(driver string) string {
	switch driver {
	case "mysql", "sqlite", "sqlite3":
		return "0"
	default:
		return "false"
	}
}

func agentTsType(driver string) string {
	if driver == "mysql" {
		return "DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3)"
	}
	if driver == "sqlite" || driver == "sqlite3" {
		return "TEXT DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now'))"
	}
	return "timestamp DEFAULT now()::TIMESTAMP"
}

type agentIndex struct {
	name    string
	table   string
	columns []string
	unique  bool
}

func createAgentIndex(db dbx.Builder, idx agentIndex) error {
	if db.DriverName() == "mysql" {
		var count int
		err := db.NewQuery(`
			SELECT COUNT(1)
			FROM information_schema.statistics
			WHERE table_schema = DATABASE()
				AND table_name = {:table}
				AND index_name = {:index}
		`).Bind(dbx.Params{
			"table": idx.table,
			"index": idx.name,
		}).Row(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
	}

	optional := " IF NOT EXISTS"
	if db.DriverName() == "mysql" {
		optional = ""
	}
	unique := ""
	if idx.unique {
		unique = "UNIQUE "
	}
	cols := ""
	for i, col := range idx.columns {
		if i > 0 {
			cols += ", "
		}
		cols += "[[" + col + "]]"
	}

	stmt := "CREATE " + unique + "INDEX" + optional + " [[" + idx.name + "]] ON {{" + idx.table + "}} (" + cols + ")"
	_, err := db.NewQuery(stmt).Execute()
	return err
}
