package agents

import (
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"path/filepath"
	"strings"
)

// AgentFileRef references an image stored in the file subsystem by record file
// field (proposal §6.2). The agent resolves it server-side to a content block,
// so the model never touches the filesystem directly.
type AgentFileRef struct {
	Collection string `json:"collection"`
	RecordId   string `json:"recordId"`
	Filename   string `json:"filename"`
}

// resolveImageInputs expands any file-referenced images into inline base64
// payloads, enforcing the project boundary (§4.4). Inline images pass through
// unchanged. The returned slice is safe to persist and replay.
func (s *Service) resolveImageInputs(project string, images []AgentImageInput) ([]AgentImageInput, error) {
	if len(images) == 0 {
		return images, nil
	}

	resolved := make([]AgentImageInput, 0, len(images))
	for _, img := range images {
		if img.FileRef == nil {
			resolved = append(resolved, img)
			continue
		}

		data, mimeType, err := s.readProjectFile(project, img.FileRef)
		if err != nil {
			return nil, err
		}
		if img.MimeType == "" {
			img.MimeType = mimeType
		}
		img.Data = data
		img.FileRef = nil
		resolved = append(resolved, img)
	}
	return resolved, nil
}

// readProjectFile reads a record file via the file subsystem and returns its
// base64-encoded content and detected mime type. The owning collection must
// belong to the given project.
func (s *Service) readProjectFile(project string, ref *AgentFileRef) (string, string, error) {
	if ref.Collection == "" || ref.RecordId == "" || ref.Filename == "" {
		return "", "", errors.New("fileRef requires collection, recordId and filename")
	}
	// reject path traversal in the filename
	if strings.Contains(ref.Filename, "/") || strings.Contains(ref.Filename, "..") {
		return "", "", errors.New("invalid filename")
	}

	collection, err := s.app.Dao().FindCollectionByNameOrId(ref.Collection)
	if err != nil {
		return "", "", err
	}
	if collection.Project == nil || *collection.Project != project {
		return "", "", errors.New("collection is outside of the current project scope")
	}

	record, err := s.app.Dao().FindRecordById(collection.Id, ref.RecordId)
	if err != nil {
		return "", "", err
	}

	fs, err := s.app.NewFilesystem()
	if err != nil {
		return "", "", err
	}
	defer fs.Close()

	key := record.BaseFilesPath() + "/" + ref.Filename
	reader, err := fs.GetFile(key)
	if err != nil {
		return "", "", err
	}
	defer reader.Close()

	raw, err := io.ReadAll(reader)
	if err != nil {
		return "", "", err
	}

	mimeType := mime.TypeByExtension(filepath.Ext(ref.Filename))
	if mimeType == "" {
		mimeType = "image/png"
	}
	return base64.StdEncoding.EncodeToString(raw), mimeType, nil
}
