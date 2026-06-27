package models

import "github.com/zhenruyan/postgrebase/tools/types"

var _ Model = (*VectorEntry)(nil)

// VectorEntry stores a persisted embedding row.
type VectorEntry struct {
	BaseModel

	ProjectID      string        `db:"project_id" json:"projectId"`
	SourceType     string        `db:"source_type" json:"sourceType"`
	SourceID       string        `db:"source_id" json:"sourceId"`
	SourceField    string        `db:"source_field" json:"sourceField"`
	EmbeddingModel string        `db:"embedding_model" json:"embeddingModel"`
	Vector         types.JsonRaw `db:"vector" json:"vector"`
	ContentHash    string        `db:"content_hash" json:"contentHash"`
}

// TableName returns the vector entry SQL table name.
func (m *VectorEntry) TableName() string {
	return "_pb_vector_entries_"
}
