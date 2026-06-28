package migrations

import "github.com/zhenruyan/postgrebase/dbx"

func init() {
	AppMigrations.Register(func(db dbx.Builder) error {
		driver := db.DriverName()

		sessionTable := `
			CREATE TABLE IF NOT EXISTS {{_pb_agent_sessions_}} (
				[[id]]           ` + agentIdType(driver) + ` NOT NULL PRIMARY KEY,
				[[project_id]]   ` + agentTextType(driver) + ` NOT NULL,
				[[name]]         ` + agentTextType(driver) + ` NOT NULL,
				[[provider]]     ` + agentTextType(driver) + ` NOT NULL,
				[[model]]        ` + agentTextType(driver) + ` NOT NULL,
				[[name_locked]]  ` + agentBoolType(driver) + ` NOT NULL DEFAULT ` + agentBoolDefault(driver) + `,
				[[last_message]] ` + agentTextType(driver) + ` NOT NULL DEFAULT '',
				[[created]]      ` + agentTsType(driver) + ` NOT NULL,
				[[updated]]      ` + agentTsType(driver) + ` NOT NULL
			);`

		messageTable := `
			CREATE TABLE IF NOT EXISTS {{_pb_agent_messages_}} (
				[[id]]         ` + agentIdType(driver) + ` NOT NULL PRIMARY KEY,
				[[session_id]] ` + agentTextType(driver) + ` NOT NULL,
				[[role]]       ` + agentTextType(driver) + ` NOT NULL,
				[[content]]    ` + agentTextType(driver) + ` NOT NULL DEFAULT '',
				[[images]]     ` + agentJsonType(driver) + `,
				[[created]]    ` + agentTsType(driver) + ` NOT NULL,
				[[updated]]    ` + agentTsType(driver) + ` NOT NULL
			);`

		auditTable := `
			CREATE TABLE IF NOT EXISTS {{_pb_agent_audit_}} (
				[[id]]             ` + agentIdType(driver) + ` NOT NULL PRIMARY KEY,
				[[session_id]]     ` + agentTextType(driver) + ` NOT NULL,
				[[project_id]]     ` + agentTextType(driver) + ` NOT NULL,
				[[actor]]          ` + agentTextType(driver) + ` NOT NULL DEFAULT '',
				[[tool]]           ` + agentTextType(driver) + ` NOT NULL,
				[[category]]       ` + agentTextType(driver) + ` NOT NULL DEFAULT '',
				[[risk]]           ` + agentTextType(driver) + ` NOT NULL DEFAULT '',
				[[audit_category]] ` + agentTextType(driver) + ` NOT NULL DEFAULT '',
				[[decision]]       ` + agentTextType(driver) + ` NOT NULL DEFAULT '',
				[[reason]]         ` + agentTextType(driver) + ` NOT NULL DEFAULT '',
				[[status]]         ` + agentTextType(driver) + ` NOT NULL DEFAULT '',
				[[error_msg]]      ` + agentTextType(driver) + ` NOT NULL DEFAULT '',
				[[created]]        ` + agentTsType(driver) + ` NOT NULL,
				[[updated]]        ` + agentTsType(driver) + ` NOT NULL
			);`

		stmts := []string{sessionTable, messageTable, auditTable}
		stmts = append(stmts,
			`CREATE TABLE IF NOT EXISTS {{_pb_agent_project_configs_}} (
				[[id]]                  `+agentIdType(driver)+` NOT NULL PRIMARY KEY,
				[[project_id]]          `+agentTextType(driver)+` NOT NULL,
				[[default_provider]]    `+agentTextType(driver)+` NOT NULL DEFAULT '',
				[[default_model]]       `+agentTextType(driver)+` NOT NULL DEFAULT '',
				[[allowed_tools]]       `+agentJsonType(driver)+`,
				[[allow_schema_change]] `+agentTextType(driver)+` NOT NULL DEFAULT 'inherit',
				[[approval_policy]]     `+agentTextType(driver)+` NOT NULL DEFAULT 'inherit',
				[[created]]             `+agentTsType(driver)+` NOT NULL,
				[[updated]]             `+agentTsType(driver)+` NOT NULL
			);`,
			"CREATE INDEX IF NOT EXISTS [[idx_agent_sessions_project]] ON {{_pb_agent_sessions_}} ([[project_id]])",
			"CREATE INDEX IF NOT EXISTS [[idx_agent_messages_session]] ON {{_pb_agent_messages_}} ([[session_id]])",
			"CREATE INDEX IF NOT EXISTS [[idx_agent_audit_session]] ON {{_pb_agent_audit_}} ([[session_id]])",
			"CREATE UNIQUE INDEX IF NOT EXISTS [[idx_agent_project_config]] ON {{_pb_agent_project_configs_}} ([[project_id]])",
		)

		for _, stmt := range stmts {
			if _, err := db.NewQuery(stmt).Execute(); err != nil {
				return err
			}
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

func agentTextType(driver string) string {
	if driver == "mysql" {
		return "TEXT"
	}
	return "text"
}

func agentJsonType(driver string) string {
	if driver == "mysql" {
		return "JSON"
	}
	return "text"
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
