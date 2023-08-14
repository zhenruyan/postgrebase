package core

import (
	_ "github.com/lib/pq"
	"github.com/pocketbase/dbx"
)

func connectDB(dsn string) (*dbx.DB, error) {
	db, err := dbx.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}
