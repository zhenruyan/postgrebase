package vector

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/zhenruyan/postgrebase/dbx"
	"github.com/zhenruyan/postgrebase/daos"
)

const vectorRuntimeParamKey = "vector_runtime"

// DBEngine persists vector runtime state via the application's params table.
type DBEngine struct {
	dao *daos.Dao
	key string
}

// NewDBEngine creates a new param-backed engine.
func NewDBEngine(dao *daos.Dao, key string) *DBEngine {
	if key == "" {
		key = vectorRuntimeParamKey
	}

	return &DBEngine{dao: dao, key: key}
}

// Load restores the vector snapshot from the params table.
func (e *DBEngine) Load() (Snapshot, bool, error) {
	if e == nil || e.dao == nil {
		return Snapshot{}, false, nil
	}

	param, err := e.dao.FindParamByKey(e.key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || isMissingTableErr(err) {
			return Snapshot{}, false, nil
		}
		return Snapshot{}, false, err
	}
	if param == nil || len(param.Value) == 0 {
		return Snapshot{}, false, nil
	}

	var snapshot Snapshot
	if err := json.Unmarshal(param.Value, &snapshot); err != nil {
		return Snapshot{}, false, err
	}

	return snapshot, true, nil
}

// Save stores the snapshot in the params table.
func (e *DBEngine) Save(snapshot Snapshot) error {
	if e == nil || e.dao == nil {
		return nil
	}

	return e.dao.SaveParam(e.key, snapshot)
}

var _ Engine = (*DBEngine)(nil)

// isMissingTableErr reports whether the error is caused by a not-yet-migrated
// table. On first boot the vector runtime is initialized before the database
// migrations run, so the backing tables may not exist yet.
func isMissingTableErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "no such table"): // sqlite
		return true
	case strings.Contains(msg, "does not exist"): // postgres
		return true
	case strings.Contains(msg, "doesn't exist"): // mysql
		return true
	default:
		return false
	}
}


// -------------------------------------------------------------------

// SQLiteEngine persists vector runtime state in the dedicated SQLite database.
type SQLiteEngine struct {
	db  *dbx.DB
	key string
}

func NewSQLiteEngine(db *dbx.DB, key string) *SQLiteEngine {
	if key == "" {
		key = "vector_runtime"
	}
	return &SQLiteEngine{db: db, key: key}
}

func (e *SQLiteEngine) Load() (Snapshot, bool, error) {
	if e == nil || e.db == nil {
		return Snapshot{}, false, nil
	}

	var row struct {
		Value string `db:"value"`
	}
	err := e.db.Select("value").
		From("_pb_vector_params_").
		Where(dbx.HashExp{"key": e.key}).
		One(&row)
	if err != nil {
		return Snapshot{}, false, nil
	}

	var snapshot Snapshot
	if err := json.Unmarshal([]byte(row.Value), &snapshot); err != nil {
		return Snapshot{}, false, err
	}

	return snapshot, true, nil
}

func (e *SQLiteEngine) Save(snapshot Snapshot) error {
	if e == nil || e.db == nil {
		return nil
	}

	valBytes, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	// Upsert param
	_, err = e.db.NewQuery("INSERT INTO _pb_vector_params_ (key, value) VALUES ({:key}, {:value}) ON CONFLICT(key) DO UPDATE SET value = excluded.value").
		Bind(dbx.Params{"key": e.key, "value": string(valBytes)}).
		Execute()
	return err
}

var _ Engine = (*SQLiteEngine)(nil)
