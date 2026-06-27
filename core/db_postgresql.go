package core

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/zhenruyan/postgrebase/dbx"
	_ "modernc.org/sqlite"
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

	if driver == "sqlite" {
		if err := ensureSQLitePath(dsn); err != nil {
			return nil, err
		}
	}

	db, err := dbx.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func ensureSQLitePath(dsn string) error {
	if dsn == "" {
		return nil
	}
	if dsn == ":memory:" || strings.HasPrefix(dsn, "file::memory:") {
		return nil
	}

	path := dsn
	if strings.HasPrefix(path, "file:") {
		if u, err := url.Parse(path); err == nil && u.Path != "" {
			path = u.Path
		}
	}

	dir := filepath.Dir(path)
	if dir == "." || dir == string(filepath.Separator) {
		return nil
	}

	return os.MkdirAll(dir, os.ModePerm)
}
