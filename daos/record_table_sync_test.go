package daos

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zhenruyan/postgrebase/dbx"
	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/models/schema"
	"github.com/zhenruyan/postgrebase/tools/types"
	_ "modernc.org/sqlite"
)

func TestIsMySQLRequiredJSONField(t *testing.T) {
	tests := []struct {
		name  string
		field *schema.SchemaField
		want  bool
	}{
		{
			name:  "json field remains nullable",
			field: &schema.SchemaField{Type: schema.FieldTypeJson},
			want:  false,
		},
		{
			name:  "multiple select uses required json",
			field: &schema.SchemaField{Type: schema.FieldTypeSelect, Options: &schema.SelectOptions{MaxSelect: 2}},
			want:  true,
		},
		{
			name:  "single relation is varchar",
			field: &schema.SchemaField{Type: schema.FieldTypeRelation, Options: &schema.RelationOptions{MaxSelect: types.Pointer(1)}},
			want:  false,
		},
		{
			name:  "multiple relation uses required json",
			field: &schema.SchemaField{Type: schema.FieldTypeRelation, Options: &schema.RelationOptions{}},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMySQLRequiredJSONField(tt.field); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

// TestIsMySQLRequiredJSONFieldNonPointerOptions guards the normalization added
// in addSchemaFieldColumn: ColDefinition() and isMySQLRequiredJSONField() must
// agree regardless of whether Options arrived as a value or a pointer,
// otherwise a column can be created nullable and never tightened.
func TestIsMySQLRequiredJSONFieldNonPointerOptions(t *testing.T) {
	valueField := &schema.SchemaField{
		Type:    schema.FieldTypeSelect,
		Options: schema.SelectOptions{MaxSelect: 2},
	}

	if err := valueField.InitOptions(); err != nil {
		t.Fatalf("InitOptions failed: %v", err)
	}

	if !isMySQLRequiredJSONField(valueField) {
		t.Fatal("expected multi-select field to be detected after InitOptions")
	}

	if got := valueField.ColDefinition("mysql"); got != "JSON NOT NULL" {
		t.Fatalf("expected ColDefinition to agree on JSON NOT NULL, got %q", got)
	}

	// A single-value select must stay a plain varchar on both code paths.
	single := &schema.SchemaField{
		Type:    schema.FieldTypeSelect,
		Options: schema.SelectOptions{MaxSelect: 1},
	}
	if err := single.InitOptions(); err != nil {
		t.Fatalf("InitOptions failed: %v", err)
	}
	if isMySQLRequiredJSONField(single) {
		t.Fatal("single-select field must not use required json")
	}
}

// TestIsMySQLRequiredJSONFieldFile covers the multi-file case that the helper's
// name does not obviously advertise.
func TestIsMySQLRequiredJSONFieldFile(t *testing.T) {
	multiFile := &schema.SchemaField{
		Type:    schema.FieldTypeFile,
		Options: &schema.FileOptions{MaxSelect: 3},
	}
	if !isMySQLRequiredJSONField(multiFile) {
		t.Fatal("expected multi-file field to use required json")
	}

	singleFile := &schema.SchemaField{
		Type:    schema.FieldTypeFile,
		Options: &schema.FileOptions{MaxSelect: 1},
	}
	if isMySQLRequiredJSONField(singleFile) {
		t.Fatal("single-file field must not use required json")
	}
}

func newSQLiteDao(t *testing.T) *Dao {
	t.Helper()

	path := filepath.Join(t.TempDir(), "sync.db")
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	return New(dbx.NewFromDB(sqlDB, "sqlite"))
}

// TestCreateCollectionIndexesValidatesBeforeDDL verifies that an unsupported
// index expression is reported without dropping or creating anything, so a
// rejected collection update cannot leave the table unindexed.
func TestCreateCollectionIndexesValidatesBeforeDDL(t *testing.T) {
	dao := newSQLiteDao(t)

	if _, err := dao.DB().NewQuery(
		`CREATE TABLE demo (id TEXT NOT NULL PRIMARY KEY, email TEXT DEFAULT '' NOT NULL);`,
	).Execute(); err != nil {
		t.Fatal(err)
	}

	collection := &models.Collection{Name: "demo"}
	collection.Type = models.CollectionTypeBase
	collection.Indexes = []string{
		"CREATE INDEX ON demo (email)", // missing index name -> invalid
	}

	err := dao.createCollectionIndexes(collection)
	if err == nil {
		t.Fatal("expected invalid index expression to be rejected")
	}

	// Nothing should have been created for the invalid expression.
	rows, err := dao.DB().NewQuery("SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = 'demo'").Rows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		if name == "demo_bad_idx" {
			t.Fatal("invalid index must not be created")
		}
	}
}

// TestSyncRecordTableSchemaRejectsBadIndexBeforeDrop confirms the pre-flight
// validation runs before the old indexes are dropped.
func TestSyncRecordTableSchemaRejectsBadIndexBeforeDrop(t *testing.T) {
	dao := newSQLiteDao(t)

	if _, err := dao.DB().NewQuery(
		`CREATE TABLE demo (id TEXT NOT NULL PRIMARY KEY, email TEXT DEFAULT '' NOT NULL);`,
	).Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := dao.DB().NewQuery(
		`CREATE UNIQUE INDEX demo_good_idx ON demo (email);`,
	).Execute(); err != nil {
		t.Fatal(err)
	}

	old := &models.Collection{Name: "demo"}
	old.Type = models.CollectionTypeBase
	old.Indexes = []string{"CREATE UNIQUE INDEX demo_good_idx ON demo (email)"}

	fresh := func(indexes ...string) *models.Collection {
		c := &models.Collection{Name: "demo"}
		c.Type = models.CollectionTypeBase
		c.Indexes = indexes
		return c
	}

	// A malformed expression must be rejected before the existing index is dropped.
	if err := dao.SyncRecordTableSchema(fresh("CREATE INDEX ON demo (email)"), old); err == nil {
		t.Fatal("expected malformed index expression to fail the sync")
	}

	// The previously valid index must still be present because validation
	// happened before any DDL.
	var count int
	if err := dao.DB().NewQuery(
		"SELECT COUNT(1) FROM sqlite_master WHERE type = 'index' AND name = 'demo_good_idx'",
	).Row(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected demo_good_idx to survive the rejected update, found %d", count)
	}
}

// TestSyncRecordTableSchemaAuthIndexesQuoting guards against the unquoted
// UUID-derived index names: SQLite parses "-" as subtraction, so every auth
// collection failed to create with `near "-": syntax error`.
func TestSyncRecordTableSchemaAuthIndexesQuoting(t *testing.T) {
	dao := newSQLiteDao(t)

	name := "members"
	coll := &models.Collection{Name: name, Type: models.CollectionTypeAuth, DisplayName: &name}
	coll.MarkAsNew()
	coll.RefreshId()

	if err := dao.SyncRecordTableSchema(coll, nil); err != nil {
		t.Fatalf("auth collection table sync failed: %v", err)
	}

	// The generated unique index names embed the collection id (a UUID with
	// hyphens), so they must be quoted in sqlite_master.
	var names []string
	rows, err := dao.DB().NewQuery(
		"SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = {:tbl} ORDER BY name",
	).Bind(dbx.Params{"tbl": coll.Name}).Rows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		names = append(names, n)
	}

	want := []string{
		"_" + coll.Id + "_email_idx",
		"_" + coll.Id + "_tokenKey_idx",
		"_" + coll.Id + "_username_idx",
	}
	for _, w := range want {
		var found bool
		for _, n := range names {
			if n == w {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected index %q to be created, got %v", w, names)
		}
	}

	// Multiple records with an empty email must coexist: verify the index itself
	// is a partial unique index on the raw column list.
	var idxSQL string
	if err := dao.DB().NewQuery(
		"SELECT sql FROM sqlite_master WHERE type = 'index' AND name = {:name}",
	).Bind(dbx.Params{"name": "_" + coll.Id + "_email_idx"}).Row(&idxSQL); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(idxSQL, "WHERE") {
		t.Fatalf("expected the email index to stay a partial index, got %q", idxSQL)
	}

	if _, err := dao.DB().NewQuery(
		"INSERT INTO {{" + coll.Name + "}} ([[id]], [[email]], [[username]], [[tokenKey]], [[passwordHash]]) " +
			"VALUES ('r1', '', 'u1', 'tk1', 'h'), ('r2', '', 'u2', 'tk2', 'h')",
	).Execute(); err != nil {
		t.Fatalf("two records with empty emails must coexist, got: %v", err)
	}

	// But duplicate non-empty emails must still collide.
	_, dupErr := dao.DB().NewQuery(
		"INSERT INTO {{" + coll.Name + "}} ([[id]], [[email]], [[username]], [[tokenKey]], [[passwordHash]]) " +
			"VALUES ('r3', 'a@b.c', 'u3', 'tk3', 'h'), ('r4', 'a@b.c', 'u4', 'tk4', 'h')",
	).Execute()
	if dupErr == nil {
		t.Fatal("expected duplicate non-empty emails to violate the unique index")
	}
}
