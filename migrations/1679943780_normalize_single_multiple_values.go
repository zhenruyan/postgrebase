package migrations

import (
	"github.com/pocketbase/dbx"
)

// Normalizes old single and multiple values of MultiValuer fields (file, select, relation).
func init() {
	AppMigrations.Register(func(db dbx.Builder) error {
		return normalizeMultivaluerFields(db)
	}, func(db dbx.Builder) error {
		return nil
	})
}

func normalizeMultivaluerFields(db dbx.Builder) error {
	return nil
}
