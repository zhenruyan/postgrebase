// Package migrations contains the system PocketBase DB migrations.
package migrations

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/free/postgresqlbaseapi/dbx"
	"github.com/free/postgresqlbaseapi/daos"
	"github.com/free/postgresqlbaseapi/models"
	"github.com/free/postgresqlbaseapi/models/schema"
	"github.com/free/postgresqlbaseapi/models/settings"
	"github.com/free/postgresqlbaseapi/tools/migrate"
	"github.com/free/postgresqlbaseapi/tools/types"
)

var AppMigrations migrate.MigrationsList

// Register is a short alias for `AppMigrations.Register()`
// that is usually used in external/user defined migrations.
func Register(
	up func(db dbx.Builder) error,
	down func(db dbx.Builder) error,
	optFilename ...string,
) {
	var optFiles []string
	if len(optFilename) > 0 {
		optFiles = optFilename
	} else {
		_, path, _, _ := runtime.Caller(1)
		optFiles = append(optFiles, filepath.Base(path))
	}
	AppMigrations.Register(up, down, optFiles...)
}

func init() {
	AppMigrations.Register(func(db dbx.Builder) error {
		if db.DriverName() == "mysql" {
			statements := []string{
				`CREATE TABLE IF NOT EXISTS {{_admins}} (
					[[id]]              VARCHAR(150) NOT NULL PRIMARY KEY,
					[[avatar]]          INT DEFAULT 0 NOT NULL,
					[[email]]           VARCHAR(255) UNIQUE NOT NULL,
					[[tokenKey]]        VARCHAR(255) UNIQUE NOT NULL,
					[[passwordHash]]    VARCHAR(255) NOT NULL,
					[[lastResetSentAt]] VARCHAR(255) DEFAULT '' NOT NULL,
					[[created]]         DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) NOT NULL,
					[[updated]]         DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) NOT NULL
				);`,
				`CREATE TABLE IF NOT EXISTS {{_collections}} (
					[[id]]             VARCHAR(150) NOT NULL PRIMARY KEY,
					[[system]]         TINYINT(1) DEFAULT 0 NOT NULL,
					[[type]]           VARCHAR(255) DEFAULT 'base' NOT NULL,
					[[name]]           VARCHAR(255) UNIQUE NOT NULL,
					[[display_name]]   VARCHAR(255) DEFAULT NULL,
					[[project]]        VARCHAR(255) DEFAULT NULL,
					[[schema]]         JSON NOT NULL,
					[[indexes]]        JSON NOT NULL,
					[[listRule]]       TEXT DEFAULT NULL,
					[[viewRule]]       TEXT DEFAULT NULL,
					[[createRule]]     TEXT DEFAULT NULL,
					[[updateRule]]     TEXT DEFAULT NULL,
					[[deleteRule]]     TEXT DEFAULT NULL,
					[[options]]        JSON NOT NULL,
					[[created]]        DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) NOT NULL,
					[[updated]]        DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) NOT NULL
				);`,
				`CREATE TABLE IF NOT EXISTS {{_params}} (
					[[id]]      VARCHAR(150) NOT NULL PRIMARY KEY,
					[[key]]     VARCHAR(255) UNIQUE NOT NULL,
					[[value]]   JSON DEFAULT NULL,
					[[created]] DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) NOT NULL,
					[[updated]] DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) NOT NULL
				);`,
				`CREATE TABLE IF NOT EXISTS {{_externalAuths}} (
					[[id]]           VARCHAR(150) NOT NULL PRIMARY KEY,
					[[collectionId]] VARCHAR(150) NOT NULL,
					[[recordId]]     VARCHAR(150) NOT NULL,
					[[provider]]     VARCHAR(255) NOT NULL,
					[[providerId]]   VARCHAR(255) NOT NULL,
					[[created]]      DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) NOT NULL,
					[[updated]]      DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) NOT NULL,
					FOREIGN KEY ([[collectionId]]) REFERENCES {{_collections}} ([[id]]) ON UPDATE CASCADE ON DELETE CASCADE
				);`,
			}

			for _, s := range statements {
				if _, err := db.NewQuery(s).Execute(); err != nil {
					return err
				}
			}
		} else {
			statements := []string{
				`create extension IF NOT EXISTS "uuid-ossp";`,
				`CREATE TABLE IF NOT EXISTS {{_admins}} (
					[[id]]              text NOT NULL DEFAULT uuid_generate_v4()::text PRIMARY KEY,
					[[avatar]]          int DEFAULT 0 NOT NULL,
					[[email]]           text UNIQUE NOT NULL,
					[[tokenKey]]        text UNIQUE NOT NULL,
					[[passwordHash]]    text NOT NULL,
					[[lastResetSentAt]] text DEFAULT '' NOT NULL,
					[[created]]          timestamp NOT NULL DEFAULT now()::TIMESTAMP,
					[[updated]]          timestamp NOT NULL DEFAULT now()::TIMESTAMP
				);`,
				`CREATE TABLE IF NOT EXISTS {{_collections}} (
					[[id]]         text NOT NULL DEFAULT uuid_generate_v4()::text PRIMARY KEY,
					[[system]]     BOOLEAN DEFAULT FALSE NOT NULL,
					[[type]]       text DEFAULT 'base' NOT NULL,
					[[name]]       text UNIQUE NOT NULL,
					[[display_name]]       text DEFAULT  NULL,
					[[project]]   text DEFAULT NULL,
					[[schema]]     text DEFAULT '[]' NOT NULL,
					[[indexes]]    text DEFAULT '[]' NOT NULL,
					[[listRule]]   text DEFAULT NULL,
					[[viewRule]]   text DEFAULT NULL,
					[[createRule]] text DEFAULT NULL,
					[[updateRule]] text DEFAULT NULL,
					[[deleteRule]] text DEFAULT NULL,
					[[options]]    text DEFAULT '{}' NOT NULL,
					[[created]]     timestamp NOT NULL DEFAULT now()::TIMESTAMP,
					[[updated]]     timestamp NOT NULL DEFAULT now()::TIMESTAMP
				);`,
				`CREATE TABLE IF NOT EXISTS {{_params}} (
					[[id]]      text NOT NULL DEFAULT uuid_generate_v4()::text PRIMARY KEY,
					[[key]]     text UNIQUE NOT NULL,
					[[value]]   text DEFAULT NULL,
					[[created]]  timestamp NOT NULL DEFAULT now()::TIMESTAMP,
					[[updated]]  timestamp NOT NULL DEFAULT now()::TIMESTAMP
				);`,
				`CREATE TABLE IF NOT EXISTS {{_externalAuths}} (
					[[id]]           text NOT NULL DEFAULT uuid_generate_v4()::text  PRIMARY KEY,
					[[collectionId]] text NOT NULL,
					[[recordId]]     text NOT NULL,
					[[provider]]     text NOT NULL,
					[[providerId]]   text NOT NULL,
					[[created]]      timestamp NOT NULL DEFAULT now()::TIMESTAMP,
					[[updated]]       timestamp NOT NULL DEFAULT now()::TIMESTAMP,
					FOREIGN KEY ([[collectionId]]) REFERENCES {{_collections}} ([[id]]) ON UPDATE CASCADE ON DELETE CASCADE
				);`,
			}

			for _, s := range statements {
				if _, err := db.NewQuery(s).Execute(); err != nil {
					return err
				}
			}
		}

		if db.DriverName() == "mysql" {
			// Ensure _migrations table exists
			db.NewQuery(`CREATE TABLE IF NOT EXISTS {{_migrations}} (
				[[file]] VARCHAR(255) PRIMARY KEY,
				[[applied]] BIGINT NOT NULL
			);`).Execute()
			statements := []string{
				`CREATE UNIQUE INDEX _externalAuths_record_provider_idx ON {{_externalAuths}} ([[recordId]], [[provider]]);`,
				`CREATE UNIQUE INDEX _externalAuths_collection_provider_idx ON {{_externalAuths}} ([[collectionId]], [[provider]], [[providerId]]);`,
			}
			for _, s := range statements {
				// check if index exists first for MySQL as it doesn't support IF NOT EXISTS for CREATE INDEX
				indexName := ""
				if strings.Contains(s, "_externalAuths_record_provider_idx") {
					indexName = "_externalAuths_record_provider_idx"
				} else {
					indexName = "_externalAuths_collection_provider_idx"
				}

				var count int
				checkSql := "SELECT COUNT(1) FROM information_schema.statistics WHERE table_schema = DATABASE() AND index_name = '" + indexName + "'"
				if err := db.NewQuery(checkSql).Row(&count); err == nil && count > 0 {
					continue
				}

				if _, err := db.NewQuery(s).Execute(); err != nil {
					return err
				}
			}
		}

		dao := daos.New(db)

		// inserts default settings
		// -----------------------------------------------------------
		defaultSettings := settings.New()
		if err := dao.SaveSettings(defaultSettings); err != nil {
			return err
		}

		// inserts the system profiles collection
		// -----------------------------------------------------------
		existingUsers, _ := dao.FindCollectionByNameOrId("_pb_users_auth_")
		if existingUsers == nil {
			usersCollection := &models.Collection{}
			usersCollection.MarkAsNew()
			usersCollection.Id = "_pb_users_auth_"
			usersCollection.Name = "users"
			usersCollection.Type = models.CollectionTypeAuth
			usersCollection.ListRule = types.Pointer("id = @request.auth.id")
			usersCollection.ViewRule = types.Pointer("id = @request.auth.id")
			usersCollection.CreateRule = types.Pointer("")
			usersCollection.UpdateRule = types.Pointer("id = @request.auth.id")
			usersCollection.DeleteRule = types.Pointer("id = @request.auth.id")

			// set auth options
			usersCollection.SetOptions(models.CollectionAuthOptions{
				ManageRule:        nil,
				AllowOAuth2Auth:   true,
				AllowUsernameAuth: true,
				AllowEmailAuth:    true,
				MinPasswordLength: 8,
				RequireEmail:      false,
			})

			// set optional default fields
			usersCollection.Schema = schema.NewSchema(
				&schema.SchemaField{
					Id:      "users_name",
					Type:    schema.FieldTypeText,
					Name:    "name",
					Options: &schema.TextOptions{},
				},
				&schema.SchemaField{
					Id:   "users_avatar",
					Type: schema.FieldTypeFile,
					Name: "avatar",
					Options: &schema.FileOptions{
						MaxSelect: 1,
						MaxSize:   5242880,
						MimeTypes: []string{
							"image/jpeg",
							"image/png",
							"image/svg+xml",
							"image/gif",
							"image/webp",
						},
					},
				},
			)

			err := dao.SaveCollection(usersCollection)
			if err != nil {
				return err
			}
		}

		//
		existingProject, _ := dao.FindCollectionByNameOrId("_pb_project_")
		if existingProject == nil {
			ProjectCollection := &models.Collection{}
			ProjectCollection.MarkAsNew()
			ProjectCollection.Id = "_pb_project_"
			ProjectCollection.Name = "project"
			ProjectCollection.Type = models.CollectionTypeBase
			ProjectCollection.ListRule = types.Pointer("id = @request.auth.id")
			ProjectCollection.ViewRule = types.Pointer("id = @request.auth.id")
			ProjectCollection.CreateRule = types.Pointer("")
			ProjectCollection.UpdateRule = types.Pointer("id = @request.auth.id")
			ProjectCollection.DeleteRule = types.Pointer("id = @request.auth.id")

			// set optional default fields
			ProjectCollection.Schema = schema.NewSchema(
				&schema.SchemaField{
					Id:      "project_name",
					Type:    schema.FieldTypeText,
					Name:    "name",
					Options: &schema.TextOptions{},
				},
			)

			err := dao.SaveCollection(ProjectCollection)
			if err != nil {
				return err
			}
		}
		return nil
	}, func(db dbx.Builder) error {
		tables := []string{
			"users",
			"_externalAuths",
			"_params",
			"_collections",
			"_admins",
		}

		for _, name := range tables {
			if _, err := db.DropTable(name).Execute(); err != nil {
				return err
			}
		}

		return nil
	})
}
