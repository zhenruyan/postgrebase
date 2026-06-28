package replication_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zhenruyan/postgrebase/core"
	"github.com/zhenruyan/postgrebase/migrations"
	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/models/schema"
	"github.com/zhenruyan/postgrebase/replication"
	"github.com/zhenruyan/postgrebase/tools/migrate"
	"github.com/zhenruyan/postgrebase/vector"
)

func TestApplyCollectionUpsertCreatesPhysicalTable(t *testing.T) {
	app := newTestApp(t)
	defer app.ResetBootstrapState()

	collection := &models.Collection{
		Name: "replicated_posts",
		Type: models.CollectionTypeBase,
	}
	collection.Schema.AddField(&schema.SchemaField{
		Name: "title",
		Type: schema.FieldTypeText,
	})

	op, err := replication.NewCollectionUpsertOperation(collection, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := replication.Apply(app.Dao(), op); err != nil {
		t.Fatal(err)
	}

	persisted, err := app.Dao().FindCollectionByNameOrId("replicated_posts")
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Id != collection.Id {
		t.Fatalf("expected collection id %q, got %q", collection.Id, persisted.Id)
	}
	if !app.Dao().HasTable("replicated_posts") {
		t.Fatal("expected replicated_posts table to exist")
	}
}

func TestApplyCollectionDeleteDropsPhysicalTable(t *testing.T) {
	app := newTestApp(t)
	defer app.ResetBootstrapState()

	collection := &models.Collection{
		Name: "replicated_comments",
		Type: models.CollectionTypeBase,
	}
	if err := app.Dao().SaveCollection(collection); err != nil {
		t.Fatal(err)
	}
	if !app.Dao().HasTable("replicated_comments") {
		t.Fatal("expected replicated_comments table before delete")
	}

	op, err := replication.NewCollectionDeleteOperation(collection)
	if err != nil {
		t.Fatal(err)
	}
	if err := replication.Apply(app.Dao(), op); err != nil {
		t.Fatal(err)
	}

	if app.Dao().HasTable("replicated_comments") {
		t.Fatal("expected replicated_comments table to be dropped")
	}
}

func TestApplyAdminUpsertCreatesAdmin(t *testing.T) {
	app := newTestApp(t)
	defer app.ResetBootstrapState()

	admin := &models.Admin{
		Email: "admin@example.com",
	}
	if err := admin.SetPassword("1234567890"); err != nil {
		t.Fatal(err)
	}
	expectedHash := admin.PasswordHash
	expectedTokenKey := admin.TokenKey

	op, err := replication.NewAdminUpsertOperation(admin, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := replication.Apply(app.Dao(), op); err != nil {
		t.Fatal(err)
	}

	persisted, err := app.Dao().FindAdminByEmail("admin@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Id != admin.Id {
		t.Fatalf("expected admin id %q, got %q", admin.Id, persisted.Id)
	}
	if persisted.PasswordHash != expectedHash {
		t.Fatal("expected replicated password hash to be preserved")
	}
	if persisted.TokenKey != expectedTokenKey {
		t.Fatal("expected replicated token key to be preserved")
	}
}

func TestApplyRecordUpsertCreatesAndUpdatesRecord(t *testing.T) {
	app := newTestApp(t)
	defer app.ResetBootstrapState()

	collection := &models.Collection{
		Name: "replicated_records",
		Type: models.CollectionTypeBase,
	}
	collection.Schema.AddField(&schema.SchemaField{
		Name: "title",
		Type: schema.FieldTypeText,
	})
	if err := app.Dao().SaveCollection(collection); err != nil {
		t.Fatal(err)
	}

	record := models.NewRecord(collection)
	record.Set("title", "first")
	op, err := replication.NewRecordUpsertOperation(record, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := replication.Apply(app.Dao(), op); err != nil {
		t.Fatal(err)
	}

	persisted, err := app.Dao().FindRecordById(collection.Id, record.Id)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.GetString("title") != "first" {
		t.Fatalf("expected first title, got %q", persisted.GetString("title"))
	}

	persisted.Set("title", "second")
	updateOp, err := replication.NewRecordUpsertOperation(persisted, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := replication.Apply(app.Dao(), updateOp); err != nil {
		t.Fatal(err)
	}

	updated, err := app.Dao().FindRecordById(collection.Id, record.Id)
	if err != nil {
		t.Fatal(err)
	}
	if updated.GetString("title") != "second" {
		t.Fatalf("expected second title, got %q", updated.GetString("title"))
	}
}

func TestApplyRecordDeleteDeletesRecord(t *testing.T) {
	app := newTestApp(t)
	defer app.ResetBootstrapState()

	collection := &models.Collection{
		Name: "replicated_deletes",
		Type: models.CollectionTypeBase,
	}
	if err := app.Dao().SaveCollection(collection); err != nil {
		t.Fatal(err)
	}
	record := models.NewRecord(collection)
	record.Set("id", "a1111111-1111-4111-8111-111111111111")
	if err := app.Dao().SaveRecord(record); err != nil {
		t.Fatal(err)
	}

	op, err := replication.NewRecordDeleteOperation(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := replication.Apply(app.Dao(), op); err != nil {
		t.Fatal(err)
	}

	if _, err := app.Dao().FindRecordById(collection.Id, record.Id); err == nil {
		t.Fatal("expected record to be deleted")
	}
}

func TestApplyUnknownOperationFails(t *testing.T) {
	app := newTestApp(t)
	defer app.ResetBootstrapState()

	err := replication.Apply(app.Dao(), vector.ReplicatedOperation{
		Kind: vector.ReplicatedOperationKindSQLite,
		Type: "unknown.operation",
	})
	if err == nil || !strings.Contains(err.Error(), "unknown replicated operation type") {
		t.Fatalf("expected unknown operation error, got %v", err)
	}
}

func newTestApp(t *testing.T) *core.BaseApp {
	t.Helper()

	dataDir := filepath.Join(t.TempDir(), "pb_data")
	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir:       dataDir,
		DataDsn:       "sqlite://" + filepath.Join(dataDir, "test.db"),
		DisableVector: true,
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	runner, err := migrate.NewRunner(app.DB(), migrations.AppMigrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Up(); err != nil {
		t.Fatal(err)
	}
	return app
}
