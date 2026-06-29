package core

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/zhenruyan/postgrebase/models/settings"
	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/vector"
)

// StartVectorWorker starts the background task worker for embedding tasks.
func (app *BaseApp) StartVectorWorker() {
	stopChan := make(chan struct{})

	app.OnTerminate().Add(func(e *TerminateEvent) error {
		close(stopChan)
		return nil
	})

	go func() {
		// Wait a bit on startup
		time.Sleep(5 * time.Second)

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				app.processVectorTasks()
			}
		}
	}()
}

func (app *BaseApp) processVectorTasks() {
	mgr := app.VectorManager()
	if mgr == nil {
		return
	}

	// Dequeue a pending task
	task, ok := mgr.DequeueEmbedding()
	if !ok {
		return
	}

	modelName := task.Model
	if modelName == "" {
		modelName = app.Settings().Agents.EmbeddingModel()
	}

	log.Printf("[Vector Worker] Processing embedding task %s (Model: %s, SourceField: %s)", task.Id, modelName, task.SourceField)

	if modelName == "" {
		log.Printf("[Vector Worker] Skipping task %s: No embedding model is configured globally or on the task", task.Id)
		return
	}

	// Fetch settings to look up the API key and Base URL
	appSettings := app.Settings()
	apiKey, baseUrl, providerModelID, found := findEmbeddingModelConfig(appSettings, modelName)
	if !found || apiKey == "" {
		log.Printf("[Vector Worker] Skipping task %s: Config not found or API key is blank for model %s", task.Id, modelName)
		return
	}

	// Call provider API to get the embedding vector
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	vectorValues, err := vector.GetEmbedding(ctx, apiKey, baseUrl, providerModelID, string(task.Payload))
	if err != nil {
		log.Printf("[Vector Worker] Failed to get embedding for task %s: %v", task.Id, err)
		return
	}

	vectorJsonBytes, err := json.Marshal(vectorValues)
	if err != nil {
		log.Printf("[Vector Worker] Failed to marshal vector for task %s: %v", task.Id, err)
		return
	}

	// Save to dedicated vector entry store (local SQLite database)
	entry := &models.VectorEntry{
		ProjectID:      task.ProjectID,
		SourceType:     task.SourceType,
		SourceID:       task.SourceID,
		SourceField:    task.SourceField,
		EmbeddingModel: modelName,
		Vector:         vectorJsonBytes,
		ContentHash:    task.ContentHash,
	}
	if err := app.Dao().SaveVectorEntry(entry); err != nil {
		log.Printf("[Vector Worker] Failed to save vector entry to store: %v", err)
	}

	// If it is a record-bound vector field, write it back to the record in the main database
	parts := strings.Split(task.SourceType, ":")
	if len(parts) == 2 && parts[0] == "record" {
		collectionID := parts[1]
		record, err := app.Dao().FindRecordById(collectionID, task.SourceID)
		if err == nil && record != nil {
			record.Set(task.SourceField, string(vectorJsonBytes))
			if err := app.Dao().SaveRecord(record); err != nil {
				log.Printf("[Vector Worker] Failed to save vector values to record %s: %v", task.SourceID, err)
			} else {
				log.Printf("[Vector Worker] Successfully updated record %s vector field %s", task.SourceID, task.SourceField)
			}
		}
	}
}

func findEmbeddingModelConfig(settings *settings.Settings, modelName string) (apiKey string, baseUrl string, providerModelId string, found bool) {
	if settings == nil {
		return "", "", "", false
	}
	for _, provider := range settings.Agents.Embedding.Providers {
		for _, m := range provider.Models {
			if m.Name == modelName {
				key := provider.ApiKey
				if strings.HasPrefix(key, "env:") {
					key = os.Getenv(strings.TrimPrefix(key, "env:"))
				}
				return key, provider.BaseUrl, m.ProviderModelId, true
			}
		}
	}
	return "", "", "", false
}
