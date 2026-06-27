package vector

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"strings"

	"github.com/spf13/cast"
	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/models/schema"
	"github.com/zhenruyan/postgrebase/tools/security"
)

// BuildRecordEmbeddingTask builds a queued embedding task from a record.
func BuildRecordEmbeddingTask(record *models.Record, field *schema.SchemaField, embeddingModel string) *EmbeddingTask {
	if record == nil || field == nil {
		return nil
	}

	content := recordEmbeddingContent(record, field)
	if strings.TrimSpace(content) == "" {
		return nil
	}

	return &EmbeddingTask{
		Id:           security.NewUUIDString(),
		ProjectID:    collectionProjectID(record.Collection()),
		SourceType:   "record",
		SourceID:     record.GetId(),
		SourceField:  field.Name,
		Model:        embeddingModel,
		ContentHash:  recordEmbeddingHash(content),
		Status:       "pending",
		Payload:      []byte(content),
	}
}

func collectionProjectID(collection *models.Collection) string {
	if collection == nil || collection.Project == nil {
		return ""
	}
	return *collection.Project
}

func recordEmbeddingContent(record *models.Record, field *schema.SchemaField) string {
	if record == nil || field == nil {
		return ""
	}

	var b bytes.Buffer
	writeLine := func(k, v string) {
		if strings.TrimSpace(v) == "" {
			return
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(v)
	}

	writeLine("collection", record.Collection().Name)
	writeLine("field", field.Name)
	writeLine("value", cast.ToString(record.Get(field.Name)))

	return b.String()
}

func recordEmbeddingHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// TriggerRecordEmbedding queues embedding tasks for the record.
func (m *Manager) TriggerRecordEmbedding(record *models.Record) []string {
	if m == nil || record == nil || record.Collection() == nil {
		return nil
	}

	model := m.Status().EmbeddingModel
	if model == "" {
		model = m.config.EmbeddingModel
	}

	queued := make([]string, 0)
	for _, field := range record.Collection().Schema.Fields() {
		switch field.Type {
		case schema.FieldTypeText, schema.FieldTypeEmail, schema.FieldTypeUrl, schema.FieldTypeEditor, schema.FieldTypeJson:
			task := BuildRecordEmbeddingTask(record, field, model)
			if task == nil {
				continue
			}
			if id := m.EnqueueEmbedding(*task); id != "" {
				queued = append(queued, id)
			}
		}
	}
	return queued
}
