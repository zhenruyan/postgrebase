package core

import (
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
	"github.com/zhenruyan/postgrebase/dbx"
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
	} else if strings.HasPrefix(dsn, "sqlite://") {
		driver = "sqlite"
		dsn = strings.TrimPrefix(dsn, "sqlite://")
	} else if strings.HasPrefix(dsn, "sqlite3://") {
		driver = "sqlite"
		dsn = strings.TrimPrefix(dsn, "sqlite3://")
	} else if strings.HasSuffix(dsn, ".db") {
		// Assume SQLite if the DSN ends with .db
		driver = "sqlite"
	}

	db, err := dbx.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}
