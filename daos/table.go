package daos

import (
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/models"
)

// HasTable checks if a table (or view) with the provided name exists (case insensitive).
func (dao *Dao) HasTable(tableName string) bool {
	var exists bool

	err := dao.DB().Select("count(*)").
		From("information_schema.tables").
		AndWhere(dbx.HashExp{"table_type": []any{"BASE TABLE", "VIEW"}}).
		AndWhere(dbx.NewExp("LOWER([[table_name]])=LOWER({:tableName})", dbx.Params{"tableName": tableName})).
		Limit(1).
		Row(&exists)

	return err == nil && exists
}

// TableColumns returns all column names of a single table by its name.
func (dao *Dao) TableColumns(tableName string) ([]string, error) {
	columns := []string{}

	err := dao.DB().NewQuery(`SELECT column_name as name
	FROM information_schema.columns
	WHERE table_name ={:tableName}`).
		Bind(dbx.Params{"tableName": tableName}).
		Column(&columns)

	return columns, err
}

// TableInfo returns the `table_info` pragma result for the specified table.
func (dao *Dao) TableInfo(tableName string) ([]*models.TableInfoRow, error) {
	info := []*models.TableInfoRow{}

	sql := `SELECT ordinal_position as cid ,column_name as name,crdb_sql_type as type,
	CASE WHEN is_nullable='NO' THEN 1
			  ELSE 0
		 END as notnull,
   column_default as dflt_value,
	 CASE WHEN column_name='id' THEN 1
			  ELSE 0
		 END as pk
	FROM information_schema.columns
	WHERE table_name ={:tableName}`

	sql = `
	SELECT a.attnum as cid, a.attname AS name, t.typname AS type
FROM pg_class c, pg_attribute a
    LEFT JOIN pg_description b
    ON a.attrelid = b.objoid
        AND a.attnum = b.objsubid, pg_type t
WHERE c.relname = {:tableName}
    AND a.attnum > 0
    AND a.attrelid = c.oid
    AND a.atttypid = t.oid
ORDER BY a.attnum;
	
	`
	err := dao.DB().NewQuery(sql).
		Bind(dbx.Params{"tableName": tableName}).
		All(&info)
	if err != nil {
		return nil, err
	}

	// mattn/go-sqlite3 doesn't throw an error on invalid or missing table
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
	indexes := []struct {
		Name string
		Sql  string
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

	result := make(map[string]string, len(indexes))

	for _, idx := range indexes {
		result[idx.Name] = idx.Sql
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
