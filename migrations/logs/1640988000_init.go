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
				[[id]]        text NOT NULL DEFAULT uuid_generate_v4()::text,
				[[rowid]]     timestamp NOT NULL DEFAULT now()::TIMESTAMP,
				[[url]]       text DEFAULT '' NOT NULL,
				[[method]]    text DEFAULT 'get' NOT NULL,
				[[status]]    int DEFAULT 200 NOT NULL,
				[[auth]]      text DEFAULT 'guest' NOT NULL,
				[[remoteIp]]    text,
				[[userIp]]    text DEFAULT '127.0.0.1' NOT NULL,
				[[referer]]   text DEFAULT '' NOT NULL,
				[[userAgent]] text DEFAULT '' NOT NULL,
				[[meta]]      text DEFAULT '{}' NOT NULL,
				[[created]]   timestamp NOT NULL DEFAULT now()::TIMESTAMP,
				[[updated]]   timestamp NOT NULL DEFAULT now()::TIMESTAMP,
				CONSTRAINT "primary" PRIMARY KEY (id)
			);
		`).Execute()

		return err
	}, func(db dbx.Builder) error {
		_, err := db.DropTable("_requests").Execute()
		return err
	})
}
