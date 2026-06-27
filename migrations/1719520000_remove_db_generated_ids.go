package migrations

import (
	"strings"

	"github.com/zhenruyan/postgrebase/dbx"
	"github.com/zhenruyan/postgrebase/daos"
	"github.com/zhenruyan/postgrebase/models"
)

func init() {
	AppMigrations.Register(func(db dbx.Builder) error {
		driver := db.DriverName()

		if driver == "sqlite" || driver == "sqlite3" {
			return nil
		}

		dao := daos.New(db)

		tables := []string{
			"_admins",
			"_collections",
			"_params",
			"_externalAuths",
			"_pb_mcp_tokens_",
		}

		for _, table := range tables {
			if _, err := db.NewQuery("ALTER TABLE {{"+table+"}} ALTER COLUMN [[id]] DROP DEFAULT").Execute(); err != nil {
				if driver == "mysql" {
					continue
				}
				return err
			}
		}

		collections := []*models.Collection{}
		if err := dao.CollectionQuery().All(&collections); err != nil {
			return err
		}
		for _, collection := range collections {
			if collection.IsView() || strings.HasPrefix(collection.Name, "_") {
				continue
			}
			if _, err := db.NewQuery("ALTER TABLE {{"+collection.Name+"}} ALTER COLUMN [[id]] DROP DEFAULT").Execute(); err != nil {
				if driver == "mysql" {
					continue
				}
				return err
			}
		}

		if driver == "postgres" || driver == "pgx" {
			_, _ = db.NewQuery(`DROP EXTENSION IF EXISTS "uuid-ossp"`).Execute()
		}

		return nil
	}, func(db dbx.Builder) error {
		return nil
	})
}
