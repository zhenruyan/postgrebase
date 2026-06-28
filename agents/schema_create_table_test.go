package agents

import (
	"path/filepath"
	"testing"

	"github.com/zhenruyan/postgrebase/core"
	"github.com/zhenruyan/postgrebase/migrations"
	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/models/schema"
	"github.com/zhenruyan/postgrebase/tools/migrate"
)

func TestCreateTableExecutor(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "pb_data")
	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir:       dataDir,
		DataDsn:       "sqlite://" + filepath.Join(dataDir, "test.db"),
		DisableVector: true,
	})

	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	defer app.ResetBootstrapState()

	if err := runMigrationsForTest(app); err != nil {
		t.Fatal(err)
	}
	if err := app.RefreshSettings(); err != nil {
		t.Fatal(err)
	}

	svc := NewService(app)
	result, err := svc.ExecuteTool("schema.create_table", map[string]any{
		"project":     "project-1",
		"name":        "posts",
		"displayName": "Posts",
		"fields": []any{
			map[string]any{"name": "title", "type": "text", "required": true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || result.Status != "ok" {
		t.Fatalf("unexpected result: %#v", result)
	}

	collection, err := app.Dao().FindCollectionByNameOrId("posts")
	if err != nil {
		t.Fatal(err)
	}
	if collection.Project == nil || *collection.Project != "project-1" {
		t.Fatalf("unexpected project: %#v", collection.Project)
	}
	if collection.DisplayName == nil || *collection.DisplayName != "Posts" {
		t.Fatalf("unexpected display name: %#v", collection.DisplayName)
	}
	if len(collection.Schema.Fields()) != 1 {
		t.Fatalf("unexpected field count: %d", len(collection.Schema.Fields()))
	}
	if !app.Dao().HasTable("posts") {
		t.Fatal("expected posts table to exist")
	}

	listed, err := svc.ExecuteTool("schema.list_tables", map[string]any{
		"project": "project-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	tables, _ := listed.Data.(map[string]any)["tables"].([]tableView)
	if len(tables) != 1 || tables[0].Name != "posts" {
		t.Fatalf("unexpected table list: %#v", listed.Data)
	}

	missing, err := svc.ExecuteTool("data.query", map[string]any{
		"project":    "project-1",
		"collection": "missing_posts",
	})
	if err != nil {
		t.Fatal(err)
	}
	if missing == nil || missing.Status != "error" {
		t.Fatalf("expected structured missing collection result, got %#v", missing)
	}

	added, err := svc.ExecuteTool("schema.add_field", map[string]any{
		"project":    "project-1",
		"collection": "posts",
		"field": map[string]any{
			"name": "slug",
			"type": "text",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if added == nil || added.Status != "ok" {
		t.Fatalf("unexpected add field result: %#v", added)
	}

	collection, err = app.Dao().FindCollectionByNameOrId("posts")
	if err != nil {
		t.Fatal(err)
	}
	if collection.Schema.GetFieldByName("slug") == nil {
		t.Fatal("expected slug field to exist")
	}
	slugField := collection.Schema.GetFieldByName("slug")

	preview, err := svc.ExecuteTool("dataset.preview", map[string]any{
		"project":    "project-1",
		"collection": "posts",
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview == nil || preview.Status != "ok" {
		t.Fatalf("unexpected preview result: %#v", preview)
	}

	indexed, err := svc.ExecuteTool("schema.create_index", map[string]any{
		"project":    "project-1",
		"collection": "posts",
		"index":      `CREATE INDEX idx_posts_title ON posts (title)`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if indexed == nil || indexed.Status != "ok" {
		t.Fatalf("unexpected create index result: %#v", indexed)
	}
	collection, err = app.Dao().FindCollectionByNameOrId("posts")
	if err != nil {
		t.Fatal(err)
	}
	if len(collection.Indexes) == 0 {
		t.Fatal("expected collection indexes to be persisted")
	}

	updated, err := svc.ExecuteTool("schema.update_field", map[string]any{
		"project":    "project-1",
		"collection": "posts",
		"field": map[string]any{
			"id":       slugField.Id,
			"name":     "headline",
			"required": true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated == nil || updated.Status != "ok" {
		t.Fatalf("unexpected update result: %#v", updated)
	}

	collection, err = app.Dao().FindCollectionByNameOrId("posts")
	if err != nil {
		t.Fatal(err)
	}
	if collection.Schema.GetFieldByName("headline") == nil {
		t.Fatal("expected headline field to exist")
	}
	if slugField == nil {
		t.Fatal("expected slug field reference")
	}

	dropped, err := svc.ExecuteTool("schema.drop_field", map[string]any{
		"project":    "project-1",
		"collection": "posts",
		"field": map[string]any{
			"id": slugField.Id,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if dropped == nil || dropped.Status != "ok" {
		t.Fatalf("unexpected drop result: %#v", dropped)
	}

	collection, err = app.Dao().FindCollectionByNameOrId("posts")
	if err != nil {
		t.Fatal(err)
	}
	if collection.Schema.GetFieldByName("slug") != nil {
		t.Fatal("expected slug field to be removed")
	}

	related, err := svc.ExecuteTool("schema.set_relation", map[string]any{
		"project":    "project-1",
		"collection": "posts",
		"field": map[string]any{
			"name": "author",
			"type": "relation",
		},
		"relation":      "posts",
		"cascadeDelete": true,
		"maxSelect":     1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if related == nil || related.Status != "ok" {
		t.Fatalf("unexpected relation result: %#v", related)
	}

	collection, err = app.Dao().FindCollectionByNameOrId("posts")
	if err != nil {
		t.Fatal(err)
	}
	relationField := collection.Schema.GetFieldByName("author")
	if relationField == nil || relationField.Type != schema.FieldTypeRelation {
		t.Fatalf("expected relation field, got %#v", relationField)
	}
	relationOptions, _ := relationField.Options.(*schema.RelationOptions)
	if relationOptions == nil || relationOptions.CollectionId != collection.Id {
		t.Fatalf("unexpected relation options: %#v", relationOptions)
	}

	inserted, err := svc.ExecuteTool("data.insert", map[string]any{
		"project":    "project-1",
		"collection": "posts",
		"data": map[string]any{
			"id":      "model-generated-bad-id",
			"project": "other-project",
			"title":   "hello",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if inserted == nil || inserted.Status != "ok" {
		t.Fatalf("unexpected insert result: %#v", inserted)
	}

	collection, err = app.Dao().FindCollectionByNameOrId("posts")
	if err != nil {
		t.Fatal(err)
	}
	record, err := app.Dao().FindRecordById("posts", inserted.Data.(*models.Record).Id)
	if err != nil {
		t.Fatal(err)
	}
	if record.GetString("title") != "hello" {
		t.Fatalf("unexpected record title: %s", record.GetString("title"))
	}

	updatedRecord, err := svc.ExecuteTool("data.update", map[string]any{
		"project":    "project-1",
		"collection": "posts",
		"id":         record.Id,
		"data": map[string]any{
			"title": "world",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updatedRecord == nil || updatedRecord.Status != "ok" {
		t.Fatalf("unexpected update record result: %#v", updatedRecord)
	}

	record, err = app.Dao().FindRecordById("posts", record.Id)
	if err != nil {
		t.Fatal(err)
	}
	if record.GetString("title") != "world" {
		t.Fatalf("unexpected updated record title: %s", record.GetString("title"))
	}

	deleted, err := svc.ExecuteTool("data.delete", map[string]any{
		"project":    "project-1",
		"collection": "posts",
		"id":         record.Id,
	})
	if err != nil {
		t.Fatal(err)
	}
	if deleted == nil || deleted.Status != "ok" {
		t.Fatalf("unexpected delete result: %#v", deleted)
	}

	if _, err := app.Dao().FindRecordById("posts", record.Id); err == nil {
		t.Fatal("expected record to be deleted")
	}

	_, err = svc.ExecuteTool("schema.create_table", map[string]any{
		"project":     "project-1",
		"name":        "jobs",
		"displayName": "Jobs",
		"fields": []any{
			map[string]any{"name": "project", "type": "text"},
			map[string]any{"name": "title", "type": "text"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	jobInserted, err := svc.ExecuteTool("data.insert", map[string]any{
		"project":    "project-1",
		"collection": "jobs",
		"data": map[string]any{
			"id":      "model-generated-bad-id",
			"project": "Apollo",
			"title":   "Engineer",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	jobRecord, err := app.Dao().FindRecordById("jobs", jobInserted.Data.(*models.Record).Id)
	if err != nil {
		t.Fatal(err)
	}
	if jobRecord.GetString("project") != "Apollo" {
		t.Fatalf("expected business project field to be preserved, got %q", jobRecord.GetString("project"))
	}
}

func runMigrationsForTest(app *core.BaseApp) error {
	runner, err := migrate.NewRunner(app.DB(), migrations.AppMigrations)
	if err != nil {
		return err
	}

	_, err = runner.Up()
	return err
}
