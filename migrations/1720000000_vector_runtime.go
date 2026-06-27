package migrations

import "github.com/zhenruyan/postgrebase/dbx"

func init() {
	AppMigrations.Register(func(db dbx.Builder) error {
		driver := db.DriverName()

		taskTable := `
			CREATE TABLE IF NOT EXISTS {{_pb_vector_tasks_}} (
				[[id]]            ` + vectorIdType(driver) + ` NOT NULL PRIMARY KEY,
				[[project_id]]    ` + vectorTextType(driver) + ` NOT NULL,
				[[source_type]]   ` + vectorTextType(driver) + ` NOT NULL,
				[[source_id]]     ` + vectorTextType(driver) + ` NOT NULL,
				[[source_field]]  ` + vectorTextType(driver) + ` NOT NULL,
				[[embedding_model]] ` + vectorTextType(driver) + ` NOT NULL,
				[[content_hash]]  ` + vectorTextType(driver) + ` NOT NULL,
				[[status]]        ` + vectorTextType(driver) + ` NOT NULL,
				[[attempt_count]] ` + vectorIntType(driver) + ` NOT NULL DEFAULT 0,
				[[payload]]       ` + vectorPayloadType(driver) + ` NOT NULL,
				[[created]]       ` + vectorCreatedType(driver) + ` NOT NULL,
				[[updated]]       ` + vectorUpdatedType(driver) + ` NOT NULL
			);`

		entryTable := `
			CREATE TABLE IF NOT EXISTS {{_pb_vector_entries_}} (
				[[id]]             ` + vectorIdType(driver) + ` NOT NULL PRIMARY KEY,
				[[project_id]]     ` + vectorTextType(driver) + ` NOT NULL,
				[[source_type]]    ` + vectorTextType(driver) + ` NOT NULL,
				[[source_id]]      ` + vectorTextType(driver) + ` NOT NULL,
				[[source_field]]   ` + vectorTextType(driver) + ` NOT NULL,
				[[embedding_model]] ` + vectorTextType(driver) + ` NOT NULL,
				[[vector]]         ` + vectorVectorType(driver) + ` NOT NULL,
				[[content_hash]]   ` + vectorTextType(driver) + ` NOT NULL,
				[[created]]        ` + vectorCreatedType(driver) + ` NOT NULL,
				[[updated]]        ` + vectorUpdatedType(driver) + ` NOT NULL
			);`

		for _, stmt := range []string{taskTable, entryTable} {
			if _, err := db.NewQuery(stmt).Execute(); err != nil {
				return err
			}
		}

		return nil
	}, func(db dbx.Builder) error {
		for _, table := range []string{"_pb_vector_tasks_", "_pb_vector_entries_"} {
			if _, err := db.NewQuery("DROP TABLE IF EXISTS {{" + table + "}}").Execute(); err != nil {
				return err
			}
		}
		return nil
	})
}

func vectorIdType(driver string) string {
	if driver == "mysql" {
		return "VARCHAR(36)"
	}
	return "text"
}

func vectorTextType(driver string) string {
	if driver == "mysql" {
		return "TEXT"
	}
	return "text"
}

func vectorIntType(driver string) string {
	if driver == "mysql" {
		return "INT"
	}
	if driver == "sqlite" || driver == "sqlite3" {
		return "INTEGER"
	}
	return "INT"
}

func vectorPayloadType(driver string) string {
	if driver == "mysql" {
		return "JSON"
	}
	return "text"
}

func vectorVectorType(driver string) string {
	if driver == "mysql" {
		return "JSON"
	}
	return "text"
}

func vectorCreatedType(driver string) string {
	if driver == "mysql" {
		return "DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3)"
	}
	if driver == "sqlite" || driver == "sqlite3" {
		return "TEXT DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now'))"
	}
	return "timestamp DEFAULT now()::TIMESTAMP"
}

func vectorUpdatedType(driver string) string {
	return vectorCreatedType(driver)
}
