package logs

import (
	"github.com/pocketbase/dbx"
)

func init() {
	LogsMigrations.Register(func(db dbx.Builder) error {

		// rename ip -> remoteIp
		if _, err := db.RenameColumn("_requests", "ip", "remoteIp").Execute(); err != nil {
			return err
		}

		// add new userIp column
		if _, err := db.AddColumn("_requests", "userIp", `TEXT DEFAULT '127.0.0.1' NOT NULL`).Execute(); err != nil {
			return err
		}

		// add new indexes
		if _, err := db.CreateIndex("_requests", "_request_remote_ip_idx", "remoteIp").Execute(); err != nil {
			return err
		}
		if _, err := db.CreateIndex("_requests", "_request_user_ip_idx", "userIp").Execute(); err != nil {
			return err
		}

		return nil
	}, func(db dbx.Builder) error {

		// drop userIp column
		if _, err := db.DropColumn("_requests", "userIp").Execute(); err != nil {
			return err
		}

		// restore original remoteIp column name
		if _, err := db.RenameColumn("_requests", "remoteIp", "ip").Execute(); err != nil {
			return err
		}

		// restore original index
		if _, err := db.CreateIndex("_requests", "_request_ip_idx", "ip").Execute(); err != nil {
			return err
		}

		return nil
	})
}
