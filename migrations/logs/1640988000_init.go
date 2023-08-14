package logs

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tools/migrate"
)

var LogsMigrations migrate.MigrationsList

func init() {
	LogsMigrations.Register(func(db dbx.Builder) (err error) {
		_, err = db.NewQuery(`
			CREATE TABLE {{_requests}} (
				[[id]]        string NOT NULL DEFAULT uuid_generate_v4()::string,
				[[rowid]]     timestamp NOT NULL DEFAULT now():::TIMESTAMP,
				[[url]]       string DEFAULT '' NOT NULL,
				[[method]]    string DEFAULT 'get' NOT NULL,
				[[status]]    int DEFAULT 200 NOT NULL,
				[[auth]]      string DEFAULT 'guest' NOT NULL,
				[[remoteIp]]    string,
				[[userIp]]    string DEFAULT '127.0.0.1' NOT NULL,
				[[referer]]   string DEFAULT '' NOT NULL,
				[[userAgent]] string DEFAULT '' NOT NULL,
				[[meta]]      string DEFAULT '{}' NOT NULL,
				[[created]]   timestamp NOT NULL DEFAULT now():::TIMESTAMP,
				[[updated]]   timestamp NOT NULL DEFAULT now():::TIMESTAMP,
				CONSTRAINT "primary" PRIMARY KEY (id)
			);
		`).Execute()

		return err
	}, func(db dbx.Builder) error {
		_, err := db.DropTable("_requests").Execute()
		return err
	})
}
