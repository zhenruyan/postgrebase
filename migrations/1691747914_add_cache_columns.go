package migrations

import (
	"github.com/zhenruyan/postgrebase/dbx"
)

func init() {
	AppMigrations.Register(func(db dbx.Builder) error {
		tables := []string{"_collections"}

		for _, table := range tables {
			if db.DriverName() == "mysql" {
				cols := map[string]string{
					"cache_enabled":        "TINYINT(1) DEFAULT 0 NOT NULL",
					"list_cache_enabled":   "TINYINT(1) DEFAULT 0 NOT NULL",
					"search_cache_enabled": "TINYINT(1) DEFAULT 0 NOT NULL",
					"cache_duration":       "INT DEFAULT 0 NOT NULL",
				}
				for col, def := range cols {
					// Check if column exists
					var count int
					checkSql := "SELECT COUNT(1) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = '" + table + "' AND column_name = '" + col + "'"
					if err := db.NewQuery(checkSql).Row(&count); err == nil && count == 0 {
						if _, err := db.NewQuery("ALTER TABLE " + table + " ADD COLUMN " + col + " " + def).Execute(); err != nil {
							return err
						}
					}
				}
			} else if db.DriverName() == "sqlite" || db.DriverName() == "sqlite3" {
				// SQLite doesn't support IF NOT EXISTS for ADD COLUMN
				// Use PRAGMA to check existing columns
				cols := map[string]string{
					"cache_enabled":        "INTEGER DEFAULT 0 NOT NULL",
					"list_cache_enabled":   "INTEGER DEFAULT 0 NOT NULL",
					"search_cache_enabled": "INTEGER DEFAULT 0 NOT NULL",
					"cache_duration":       "INTEGER DEFAULT 0 NOT NULL",
				}
				existingCols := map[string]bool{}
				rows, err := db.NewQuery("PRAGMA table_info({{" + table + "}})").Rows()
				if err != nil {
					return err
				}
				for rows.Next() {
					var cid int
					var name, colType string
					var notNull, pk int
					var dfltValue interface{}
					if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
						rows.Close()
						return err
					}
					existingCols[name] = true
				}
				rows.Close()
				for col, def := range cols {
					if !existingCols[col] {
						if _, err := db.NewQuery("ALTER TABLE {{" + table + "}} ADD COLUMN [[" + col + "]] " + def).Execute(); err != nil {
							return err
						}
					}
				}
			} else {
				// Postgres
				cols := map[string]string{
					"cache_enabled":        "BOOLEAN DEFAULT FALSE NOT NULL",
					"list_cache_enabled":   "BOOLEAN DEFAULT FALSE NOT NULL",
					"search_cache_enabled": "BOOLEAN DEFAULT FALSE NOT NULL",
					"cache_duration":       "INT DEFAULT 0 NOT NULL",
				}
				for col, def := range cols {
					if _, err := db.NewQuery("ALTER TABLE " + table + " ADD COLUMN IF NOT EXISTS " + col + " " + def).Execute(); err != nil {
						return err
					}
				}
			}
		}

		return nil
	}, func(db dbx.Builder) error {
		return nil
	})
}
