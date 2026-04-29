package forms

import (
	"errors"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/free/postgresqlbaseapi/core"
	"github.com/free/postgresqlbaseapi/models/settings"
	"github.com/free/postgresqlbaseapi/tools/filesystem"
	"github.com/free/postgresqlbaseapi/tools/security"
)

const (
	webdavFilesystemStorage = "storage"
	webdavFilesystemBackups = "backups"
)

// TestWebDAVFilesystem defines a WebDAV filesystem connection test.
type TestWebDAVFilesystem struct {
	app core.App

	// The name of the filesystem - storage or backups
	Filesystem string `form:"filesystem" json:"filesystem"`
}

// NewTestWebDAVFilesystem creates and initializes new TestWebDAVFilesystem form.
func NewTestWebDAVFilesystem(app core.App) *TestWebDAVFilesystem {
	return &TestWebDAVFilesystem{app: app}
}

// Validate makes the form validatable by implementing [validation.Validatable] interface.
func (form *TestWebDAVFilesystem) Validate() error {
	return validation.ValidateStruct(form,
		validation.Field(
			&form.Filesystem,
			validation.Required,
			validation.In(webdavFilesystemStorage, webdavFilesystemBackups),
		),
	)
}

// Submit validates and performs a WebDAV filesystem connection test.
func (form *TestWebDAVFilesystem) Submit() error {
	if err := form.Validate(); err != nil {
		return err
	}

	var webdavConfig settings.WebDAVConfig

	if form.Filesystem == webdavFilesystemBackups {
		webdavConfig = form.app.Settings().Backups.WebDAV
	} else {
		webdavConfig = form.app.Settings().WebDAV
	}

	if !webdavConfig.Enabled {
		return errors.New("WebDAV storage filesystem is not enabled")
	}

	fsys, err := filesystem.NewWebDAV(
		webdavConfig.Url,
		webdavConfig.Username,
		webdavConfig.Password,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize the WebDAV filesystem: %w", err)
	}
	defer fsys.Close()

	testPrefix := "pb_settings_test_" + security.PseudorandomString(5)
	testFileKey := testPrefix + "/test.txt"

	// try to upload a test file
	if err := fsys.Upload([]byte("test"), testFileKey); err != nil {
		return fmt.Errorf("failed to upload a test file: %w", err)
	}

	// test prefix deletion (ensures that both bucket list and delete works)
	if errs := fsys.DeletePrefix(testPrefix); len(errs) > 0 {
		return fmt.Errorf("failed to delete a test file: %w", errs[0])
	}

	return nil
}
