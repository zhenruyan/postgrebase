package agents

import (
	"encoding/base64"
	"testing"

	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/models/schema"
)

func TestResolveImageInputs(t *testing.T) {
	app := newTestApp(t)
	svc := NewService(app)

	// create a project-scoped collection with a file field
	project := "proj-1"
	collection := &models.Collection{Name: "assets"}
	collection.Project = &project
	collection.Schema = schema.NewSchema(
		&schema.SchemaField{Name: "photo", Type: schema.FieldTypeFile, Options: &schema.FileOptions{MaxSelect: 1, MaxSize: 5 << 20}},
	)
	if err := app.Dao().SaveCollection(collection); err != nil {
		t.Fatal(err)
	}

	// create a record and upload a file via the filesystem at the record path
	record := models.NewRecord(collection)
	record.Set("photo", "pixel.png")
	if err := app.Dao().SaveRecord(record); err != nil {
		t.Fatal(err)
	}

	fs, err := app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Close()
	payload := []byte{0x89, 0x50, 0x4e, 0x47} // PNG magic header
	if err := fs.Upload(payload, record.BaseFilesPath()+"/pixel.png"); err != nil {
		t.Fatal(err)
	}

	// inline image passes through unchanged
	out, err := svc.resolveImageInputs(project, []AgentImageInput{{MimeType: "image/jpeg", Data: "AAAA"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Data != "AAAA" {
		t.Fatalf("inline image should pass through, got %+v", out)
	}

	// fileRef resolves to base64 + mime
	out, err = svc.resolveImageInputs(project, []AgentImageInput{
		{FileRef: &AgentFileRef{Collection: "assets", RecordId: record.Id, Filename: "pixel.png"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].FileRef != nil {
		t.Fatalf("fileRef should be resolved and cleared, got %+v", out)
	}
	if out[0].Data != base64.StdEncoding.EncodeToString(payload) {
		t.Fatal("resolved data does not match uploaded file")
	}
	if out[0].MimeType != "image/png" {
		t.Fatalf("expected image/png mime, got %q", out[0].MimeType)
	}

	// path traversal is rejected
	if _, err := svc.resolveImageInputs(project, []AgentImageInput{
		{FileRef: &AgentFileRef{Collection: "assets", RecordId: record.Id, Filename: "../secret.png"}},
	}); err == nil {
		t.Fatal("expected path traversal to be rejected")
	}

	// cross-project access is rejected
	if _, err := svc.resolveImageInputs("other-project", []AgentImageInput{
		{FileRef: &AgentFileRef{Collection: "assets", RecordId: record.Id, Filename: "pixel.png"}},
	}); err == nil {
		t.Fatal("expected cross-project access to be rejected")
	}
}
