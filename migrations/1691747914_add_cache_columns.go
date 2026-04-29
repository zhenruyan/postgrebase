package migrations

import (
	"github.com/free/postgresqlbaseapi/dbx"
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
