package daos

import (
	"fmt"

	"github.com/zhenruyan/postgrebase/dbx"
	"github.com/zhenruyan/postgrebase/models"
)

// HasTable checks if a table (or view) with the provided name exists (case insensitive).
func (dao *Dao) HasTable(tableName string) bool {
	var exists bool

	switch dao.DB().DriverName() {
	case "sqlite", "sqlite3":
		err := dao.DB().Select("count(*)").
			From("sqlite_master").
			AndWhere(dbx.HashExp{"type": []any{"table", "view"}}).
			AndWhere(dbx.NewExp("LOWER([[name]])=LOWER({:tableName})", dbx.Params{"tableName": tableName})).
			Limit(1).
			Row(&exists)
		return err == nil && exists
	case "mysql":
		err := dao.DB().Select("count(*)").
			From("information_schema.tables").
			AndWhere(dbx.HashExp{"table_type": []any{"BASE TABLE", "VIEW"}}).
			AndWhere(dbx.NewExp("LOWER([[table_name]])=LOWER({:tableName})", dbx.Params{"tableName": tableName})).
			AndWhere(dbx.NewExp("table_schema = DATABASE()")).
			Limit(1).
			Row(&exists)
		return err == nil && exists
	default:
		// postgres
		err := dao.DB().Select("count(*)").
			From("information_schema.tables").
			AndWhere(dbx.HashExp{"table_type": []any{"BASE TABLE", "VIEW"}}).
			AndWhere(dbx.NewExp("LOWER([[table_name]])=LOWER({:tableName})", dbx.Params{"tableName": tableName})).
			Limit(1).
			Row(&exists)
		return err == nil && exists
	}
}

// TableColumns returns all column names of a single table by its name.
func (dao *Dao) TableColumns(tableName string) ([]string, error) {
	columns := []string{}

	switch dao.DB().DriverName() {
	case "sqlite", "sqlite3":
		type pragmaRow struct {
			Name string `db:"name"`
		}
		rows := []pragmaRow{}
		err := dao.DB().NewQuery("PRAGMA table_info({{" + tableName + "}})").All(&rows)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			columns = append(columns, r.Name)
		}
		return columns, nil
	case "mysql":
		err := dao.DB().NewQuery(`SELECT column_name as name
		FROM information_schema.columns
		WHERE table_name ={:tableName} AND table_schema = DATABASE()`).
			Bind(dbx.Params{"tableName": tableName}).
			Column(&columns)
		return columns, err
	default:
		// postgres
		err := dao.DB().NewQuery(`SELECT column_name as name
		FROM information_schema.columns
		WHERE table_name ={:tableName}`).
			Bind(dbx.Params{"tableName": tableName}).
			Column(&columns)
		return columns, err
	}
}

// TableInfo returns the `table_info` pragma result for the specified table.
func (dao *Dao) TableInfo(tableName string) ([]*models.TableInfoRow, error) {
	info := []*models.TableInfoRow{}

	switch dao.DB().DriverName() {
	case "sqlite", "sqlite3":
		sql := `SELECT cid, name, type, notnull, dflt_value, pk FROM pragma_table_info({:tableName})`
		err := dao.DB().NewQuery(sql).
			Bind(dbx.Params{"tableName": tableName}).
			All(&info)
		if err != nil {
			return nil, err
		}
	case "mysql":
		sql := `SELECT ordinal_position as cid, column_name as name, column_type as type,
			CASE WHEN is_nullable='NO' THEN 1 ELSE 0 END as notnull,
			column_default as dflt_value,
			CASE WHEN column_key='PRI' THEN 1 ELSE 0 END as pk
			FROM information_schema.columns
			WHERE table_name = {:tableName} AND table_schema = DATABASE()
			ORDER BY ordinal_position`
		err := dao.DB().NewQuery(sql).
			Bind(dbx.Params{"tableName": tableName}).
			All(&info)
		if err != nil {
			return nil, err
		}
	default:
		// postgres
		sql := `
		SELECT a.attnum as cid, a.attname AS name, t.typname AS type
		FROM pg_class c, pg_attribute a
			LEFT JOIN pg_description b
			ON a.attrelid = b.objoid
				AND a.attnum = b.objsubid, pg_type t
		WHERE c.relname = {:tableName}
			AND a.attnum > 0
			AND a.attrelid = c.oid
			AND a.atttypid = t.oid
		ORDER BY a.attnum
		`
		err := dao.DB().NewQuery(sql).
			Bind(dbx.Params{"tableName": tableName}).
			All(&info)
		if err != nil {
			return nil, err
		}
	}

	// sqlite doesn't throw an error on invalid or missing table
	// so we additionally have to check whether the loaded info result is nonempty
	if len(info) == 0 {
		return nil, fmt.Errorf("empty table info probably due to invalid or missing table %s", tableName)
	}

	return info, nil
}

// TableIndexes returns a name grouped map with all non empty index of the specified table.
//
// Note: This method doesn't return an error on nonexisting table.
func (dao *Dao) TableIndexes(tableName string) (map[string]string, error) {
	result := make(map[string]string)

	switch dao.DB().DriverName() {
	case "sqlite", "sqlite3":
		indexes := []struct {
			Name string `db:"name"`
			Sql  string `db:"sql"`
		}{}
		err := dao.DB().Select("name", "sql").
			From("sqlite_master").
			AndWhere(dbx.HashExp{"type": "index"}).
			AndWhere(dbx.NewExp("tbl_name={:tableName}", dbx.Params{"tableName": tableName})).
			AndWhere(dbx.NewExp("sql IS NOT NULL")).
			All(&indexes)
		if err != nil {
			return nil, err
		}
		for _, idx := range indexes {
			result[idx.Name] = idx.Sql
		}
	case "mysql":
		indexes := []struct {
			Name string `db:"name"`
			Sql  string `db:"sql"`
		}{}
		err := dao.DB().NewQuery(`
			SELECT index_name as name,
				   GROUP_CONCAT(column_name ORDER BY seq_in_index) as sql
			FROM information_schema.statistics
			WHERE table_name = {:tableName} AND table_schema = DATABASE()
			GROUP BY index_name
		`).Bind(dbx.Params{"tableName": tableName}).All(&indexes)
		if err != nil {
			return nil, err
		}
		for _, idx := range indexes {
			result[idx.Name] = idx.Sql
		}
	default:
		// postgres
		indexes := []struct {
			Name string `db:"name"`
			Sql  string `db:"sql"`
		}{}
		err := dao.DB().Select("indexname as name", "indexdef as sql").
			From("pg_indexes").
			AndWhere(dbx.NewExp("indexdef <>''")).
			AndWhere(dbx.HashExp{
				"tablename": tableName,
			}).
			All(&indexes)
		if err != nil {
			return nil, err
		}
		for _, idx := range indexes {
			result[idx.Name] = idx.Sql
		}
	}

	return result, nil
}

// DeleteTable drops the specified table.
//
// This method is a no-op if a table with the provided name doesn't exist.
//
// Be aware that this method is vulnerable to SQL injection and the
// "tableName" argument must come only from trusted input!
func (dao *Dao) DeleteTable(tableName string) error {
	_, err := dao.DB().NewQuery(fmt.Sprintf(
		"DROP TABLE IF EXISTS {{%s}}",
		tableName,
	)).Execute()

	return err
}
