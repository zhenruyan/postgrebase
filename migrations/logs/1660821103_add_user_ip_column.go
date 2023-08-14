package logs

import (
	"github.com/pocketbase/dbx"
)

func init() {
	LogsMigrations.Register(func(db dbx.Builder) error {
		// add new indexes
		if _, err := db.CreateIndex("_requests", "_request_remote_ip_idx", "remoteIp").Execute(); err != nil {
			return err
		}
		if _, err := db.CreateIndex("_requests", "_request_user_ip_idx", "userIp").Execute(); err != nil {
			return err
		}

		return nil
	}, func(db dbx.Builder) error {
		return nil
	})
}
