package migrations

import "github.com/zhenruyan/postgrebase/dbx"

func init() {
	AppMigrations.Register(func(db dbx.Builder) error {
		// Vector tables are isolated in a dedicated local sqlite database (pb_data/vector.db)
		// to decouple heavy vector data and sqlite-vec indices from the main DB switching.
		return nil
	}, func(db dbx.Builder) error {
		return nil
	})
}
