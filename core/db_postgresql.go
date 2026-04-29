package core

import (
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/free/postgresqlbaseapi/dbx"
)

func connectDB(dsn string) (*dbx.DB, error) {
	driver := "postgres"
	// Parse driver from DSN prefix
	if strings.HasPrefix(dsn, "mysql://") {
		driver = "mysql"
		dsn = strings.TrimPrefix(dsn, "mysql://")
	} else if strings.HasPrefix(dsn, "postgres://") {
		driver = "postgres"
	} else if strings.HasPrefix(dsn, "postgresql://") {
		driver = "postgres"
	}

	db, err := dbx.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}
