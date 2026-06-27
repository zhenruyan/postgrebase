package vector

import (
	"database/sql"
	"encoding/json"
	"errors"

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
		if errors.Is(err, sql.ErrNoRows) {
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
