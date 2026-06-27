package models

import "github.com/zhenruyan/postgrebase/tools/types"

var _ Model = (*VectorTask)(nil)

// VectorTask stores a queued embedding job.
type VectorTask struct {
	BaseModel

	ProjectID      string        `db:"project_id" json:"projectId"`
	SourceType     string        `db:"source_type" json:"sourceType"`
	SourceID       string        `db:"source_id" json:"sourceId"`
	SourceField    string        `db:"source_field" json:"sourceField"`
	EmbeddingModel string        `db:"embedding_model" json:"embeddingModel"`
	ContentHash    string        `db:"content_hash" json:"contentHash"`
	Status         string        `db:"status" json:"status"`
	AttemptCount   int           `db:"attempt_count" json:"attemptCount"`
	Payload        types.JsonRaw `db:"payload" json:"payload"`
}

// TableName returns the vector task SQL table name.
func (m *VectorTask) TableName() string {
	return "_pb_vector_tasks_"
}
