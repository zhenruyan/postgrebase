// Package migrations contains the system PocketBase DB migrations.
package migrations

import (
	"path/filepath"
	"runtime"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/models/settings"
	"github.com/pocketbase/pocketbase/tools/migrate"
	"github.com/pocketbase/pocketbase/tools/types"
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
		_, tablesErr := db.NewQuery(`
			CREATE TABLE {{_admins}} (
				[[id]]              string NOT NULL DEFAULT uuid_generate_v4()::string PRIMARY KEY,
				[[avatar]]          int DEFAULT 0 NOT NULL,
				[[email]]           string UNIQUE NOT NULL,
				[[tokenKey]]        string UNIQUE NOT NULL,
				[[passwordHash]]    string NOT NULL,
				[[lastResetSentAt]] string DEFAULT '' NOT NULL,
				[[created]]          timestamp NOT NULL DEFAULT now():::TIMESTAMP,
				[[updated]]          timestamp NOT NULL DEFAULT now():::TIMESTAMP
			);

			CREATE TABLE {{_collections}} (
				[[id]]         string NOT NULL DEFAULT uuid_generate_v4()::string PRIMARY KEY,
				[[system]]     BOOLEAN DEFAULT FALSE NOT NULL,
				[[type]]       string DEFAULT 'base' NOT NULL,
				[[name]]       string UNIQUE NOT NULL,
				[[schema]]     string DEFAULT '[]' NOT NULL,
				[[indexes]]    string DEFAULT '[]' NOT NULL,
				[[listRule]]   string DEFAULT NULL,
				[[viewRule]]   string DEFAULT NULL,
				[[createRule]] string DEFAULT NULL,
				[[updateRule]] string DEFAULT NULL,
				[[deleteRule]] string DEFAULT NULL,
				[[options]]    string DEFAULT '{}' NOT NULL,
				[[created]]     timestamp NOT NULL DEFAULT now():::TIMESTAMP,
				[[updated]]     timestamp NOT NULL DEFAULT now():::TIMESTAMP
			);

			CREATE TABLE {{_params}} (
				[[id]]      string NOT NULL DEFAULT uuid_generate_v4()::string PRIMARY KEY,
				[[key]]     string UNIQUE NOT NULL,
				[[value]]   string DEFAULT NULL,
				[[created]]  timestamp NOT NULL DEFAULT now():::TIMESTAMP,
				[[updated]]  timestamp NOT NULL DEFAULT now():::TIMESTAMP
			);

			CREATE TABLE {{_externalAuths}} (
				[[id]]           string NOT NULL DEFAULT uuid_generate_v4()::string  PRIMARY KEY,
				[[collectionId]] string NOT NULL,
				[[recordId]]     string NOT NULL,
				[[provider]]     string NOT NULL,
				[[providerId]]   string NOT NULL,
				[[created]]      timestamp NOT NULL DEFAULT now():::TIMESTAMP,
				[[updated]]       timestamp NOT NULL DEFAULT now():::TIMESTAMP,
				---
				FOREIGN KEY ([[collectionId]]) REFERENCES {{_collections}} ([[id]]) ON UPDATE CASCADE ON DELETE CASCADE
			);

		`).Execute()
		if tablesErr != nil {
			return tablesErr
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

		return dao.SaveCollection(usersCollection)
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
