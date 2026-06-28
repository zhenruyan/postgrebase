package vector

import (
	"github.com/zhenruyan/postgrebase/daos"
	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/tools/types"
)

// DBTaskStore persists vector tasks in the database.
type DBTaskStore struct {
	dao *daos.Dao
}

// DBEntryStore persists vector entries in the database.
type DBEntryStore struct {
	dao *daos.Dao
}

// NewDBTaskStore creates a database-backed task store.
func NewDBTaskStore(dao *daos.Dao) *DBTaskStore {
	return &DBTaskStore{dao: dao}
}

// NewDBEntryStore creates a database-backed entry store.
func NewDBEntryStore(dao *daos.Dao) *DBEntryStore {
	return &DBEntryStore{dao: dao}
}

// Load returns all queued tasks ordered by creation time.
func (s *DBTaskStore) Load() ([]EmbeddingTask, error) {
	if s == nil || s.dao == nil {
		return nil, nil
	}

	tasks := []*models.VectorTask{}
	if err := s.dao.ModelQuery(&models.VectorTask{}).
		OrderBy("created ASC").
		All(&tasks); err != nil {
		if isMissingTableErr(err) {
			return nil, nil
		}
		return nil, err
	}

	result := make([]EmbeddingTask, 0, len(tasks))
	for _, task := range tasks {
		if task == nil {
			continue
		}
		result = append(result, embeddingTaskFromModel(task))
	}

	return result, nil
}

// Replace rewrites the task table with the provided queue state.
func (s *DBTaskStore) Replace(tasks []EmbeddingTask) error {
	if s == nil || s.dao == nil {
		return nil
	}

	return s.dao.RunInTransaction(func(txDao *daos.Dao) error {
		existing := []*models.VectorTask{}
		if err := txDao.ModelQuery(&models.VectorTask{}).All(&existing); err != nil {
			return err
		}
		for _, task := range existing {
			if task == nil {
				continue
			}
			if err := txDao.DeleteVectorTask(task); err != nil {
				return err
			}
		}

		for _, task := range tasks {
			model := vectorTaskToModel(task)
			if model.GetId() == "" {
				model.RefreshId()
			}
			if model.GetCreated().IsZero() {
				model.RefreshCreated()
			}
			if model.GetUpdated().IsZero() {
				model.RefreshUpdated()
			}
			if err := txDao.SaveVectorTask(model); err != nil {
				return err
			}
		}
		return nil
	})
}

// Load returns all persisted vector entries.
func (s *DBEntryStore) Load() ([]*models.VectorEntry, error) {
	if s == nil || s.dao == nil {
		return nil, nil
	}

	entries := []*models.VectorEntry{}
	if err := s.dao.ModelQuery(&models.VectorEntry{}).
		OrderBy("created ASC").
		All(&entries); err != nil {
		if isMissingTableErr(err) {
			return nil, nil
		}
		return nil, err
	}
	return entries, nil
}

// Replace rewrites the vector entry table with the provided state.
func (s *DBEntryStore) Replace(entries []*models.VectorEntry) error {
	if s == nil || s.dao == nil {
		return nil
	}

	return s.dao.RunInTransaction(func(txDao *daos.Dao) error {
		existing := []*models.VectorEntry{}
		if err := txDao.ModelQuery(&models.VectorEntry{}).All(&existing); err != nil {
			return err
		}
		for _, entry := range existing {
			if entry == nil {
				continue
			}
			if err := txDao.DeleteVectorEntry(entry); err != nil {
				return err
			}
		}
		for _, entry := range entries {
			if entry == nil {
				continue
			}
			if entry.GetId() == "" {
				entry.RefreshId()
			}
			if entry.GetCreated().IsZero() {
				entry.RefreshCreated()
			}
			if entry.GetUpdated().IsZero() {
				entry.RefreshUpdated()
			}
			if err := txDao.SaveVectorEntry(entry); err != nil {
				return err
			}
		}
		return nil
	})
}

func vectorTaskToModel(task EmbeddingTask) *models.VectorTask {
	model := &models.VectorTask{}
	model.SetId(task.Id)
	if task.QueuedAt.IsZero() {
		task.QueuedAt = types.NowDateTime().Time()
	}
	if created, err := types.ParseDateTime(task.QueuedAt); err == nil {
		model.Created = created
	}
	model.ProjectID = task.ProjectID
	model.SourceType = task.SourceType
	model.SourceID = task.SourceID
	model.SourceField = task.SourceField
	model.EmbeddingModel = task.Model
	model.ContentHash = task.ContentHash
	model.Status = task.Status
	model.AttemptCount = task.AttemptCount
	if len(task.Payload) > 0 {
		model.Payload = types.JsonRaw(task.Payload)
	}
	model.RefreshUpdated()
	return model
}

func embeddingTaskFromModel(task *models.VectorTask) EmbeddingTask {
	result := EmbeddingTask{
		Id:           task.GetId(),
		ProjectID:    task.ProjectID,
		SourceType:   task.SourceType,
		SourceID:     task.SourceID,
		SourceField:  task.SourceField,
		Model:        task.EmbeddingModel,
		ContentHash:  task.ContentHash,
		Status:       task.Status,
		AttemptCount: task.AttemptCount,
		QueuedAt:     task.GetCreated().Time(),
	}
	if len(task.Payload) > 0 {
		result.Payload = append([]byte(nil), task.Payload...)
	}
	return result
}

func vectorEntryToModel(entry *models.VectorEntry) *models.VectorEntry {
	model := &models.VectorEntry{}
	model.SetId(entry.Id)
	model.ProjectID = entry.ProjectID
	model.SourceType = entry.SourceType
	model.SourceID = entry.SourceID
	model.SourceField = entry.SourceField
	model.EmbeddingModel = entry.EmbeddingModel
	model.Vector = types.JsonRaw(entry.Vector)
	model.ContentHash = entry.ContentHash
	model.RefreshUpdated()
	return model
}

func embeddingEntryFromModel(entry *models.VectorEntry) *models.VectorEntry {
	result := &models.VectorEntry{
		BaseModel:      entry.BaseModel,
		ProjectID:      entry.ProjectID,
		SourceType:     entry.SourceType,
		SourceID:       entry.SourceID,
		SourceField:    entry.SourceField,
		EmbeddingModel: entry.EmbeddingModel,
		Vector:         append([]byte(nil), entry.Vector...),
		ContentHash:    entry.ContentHash,
	}
	return result
}
