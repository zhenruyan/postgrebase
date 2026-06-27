package daos

import (
	"github.com/zhenruyan/postgrebase/dbx"
	"github.com/zhenruyan/postgrebase/models"
)

// SaveVectorTask upserts a queued embedding task.
func (dao *Dao) SaveVectorTask(task *models.VectorTask) error {
	return dao.Save(task)
}

// DeleteVectorTask removes a queued embedding task.
func (dao *Dao) DeleteVectorTask(task *models.VectorTask) error {
	return dao.Delete(task)
}

// FindVectorTasks returns all queued tasks ordered by creation time.
func (dao *Dao) FindVectorTasks() ([]*models.VectorTask, error) {
	tasks := []*models.VectorTask{}
	if err := dao.ModelQuery(&models.VectorTask{}).
		OrderBy("created ASC").
		All(&tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// SaveVectorEntry upserts a persisted embedding entry.
func (dao *Dao) SaveVectorEntry(entry *models.VectorEntry) error {
	return dao.Save(entry)
}

// DeleteVectorEntry removes a persisted embedding entry.
func (dao *Dao) DeleteVectorEntry(entry *models.VectorEntry) error {
	return dao.Delete(entry)
}

// FindVectorEntries returns all persisted entries ordered by creation time.
func (dao *Dao) FindVectorEntries() ([]*models.VectorEntry, error) {
	entries := []*models.VectorEntry{}
	if err := dao.ModelQuery(&models.VectorEntry{}).
		OrderBy("created ASC").
		All(&entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// FindVectorEntryByKey finds a specific persisted embedding entry.
func (dao *Dao) FindVectorEntryByKey(projectID, sourceType, sourceID, sourceField, embeddingModel string) (*models.VectorEntry, error) {
	entry := &models.VectorEntry{}
	err := dao.ModelQuery(entry).
		AndWhere(dbx.HashExp{
			"project_id":      projectID,
			"source_type":     sourceType,
			"source_id":       sourceID,
			"source_field":    sourceField,
			"embedding_model": embeddingModel,
		}).
		Limit(1).
		One(entry)
	if err != nil {
		return nil, err
	}
	return entry, nil
}
