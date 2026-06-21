package mcp

import (
	"encoding/json"
	"strings"

	"github.com/zhenruyan/postgrebase/models"
)

// registerResources registers all available resources
func (s *Server) registerResources() {
	s.resources["postgrebase://collections"] = s.resourceCollections
	s.resources["postgrebase://settings"] = s.resourceSettings
}

// resourceCollections returns all collections with their schemas
func (s *Server) resourceCollections(uri string) (*ResourceReadResult, error) {
	collections := []*models.Collection{}
	err := s.app.Dao().CollectionQuery().All(&collections)
	if err != nil {
		return nil, err
	}

	// Build a simplified representation
	type CollectionInfo struct {
		ID          string      `json:"id"`
		Name        string      `json:"name"`
		Type        string      `json:"type"`
		System      bool        `json:"system"`
		DisplayName interface{} `json:"displayName"`
		Schema      interface{} `json:"schema"`
		Created     interface{} `json:"created"`
		Updated     interface{} `json:"updated"`
	}

	result := make([]CollectionInfo, 0, len(collections))
	for _, c := range collections {
		info := CollectionInfo{
			ID:          c.Id,
			Name:        c.Name,
			Type:        c.Type,
			System:      c.System,
			DisplayName: c.DisplayName,
			Schema:      c.Schema,
			Created:     c.Created,
			Updated:     c.Updated,
		}
		result = append(result, info)
	}

	data, _ := json.MarshalIndent(result, "", "  ")

	return &ResourceReadResult{
		Contents: []ResourceContent{
			{
				URI:      uri,
				MimeType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

// resourceSettings returns sanitized application settings
func (s *Server) resourceSettings(uri string) (*ResourceReadResult, error) {
	settings := s.app.Settings()

	// Build sanitized settings (remove sensitive data)
	sanitized := map[string]interface{}{
		"meta": map[string]interface{}{
			"appName":        settings.Meta.AppName,
			"appUrl":         settings.Meta.AppUrl,
			"senderName":     settings.Meta.SenderName,
			"senderAddress":  settings.Meta.SenderAddress,
			"verificationTemplate": settings.Meta.VerificationTemplate,
		},
		"logs": map[string]interface{}{
			"maxDays": settings.Logs.MaxDays,
		},
	}

	// Check if admin auth is configured
	totalAdmins, _ := s.app.Dao().TotalAdmins()
	sanitized["hasAdmins"] = totalAdmins > 0

	data, _ := json.MarshalIndent(sanitized, "", "  ")

	return &ResourceReadResult{
		Contents: []ResourceContent{
			{
				URI:      uri,
				MimeType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

// resolveCollectionResource handles dynamic collection resources
// e.g., postgrebase://collections/users
func (s *Server) resolveCollectionResource(uri string) (*ResourceReadResult, error) {
	parts := strings.Split(uri, "/")
	if len(parts) < 4 {
		return nil, nil
	}

	collectionName := parts[3]
	collection, err := s.app.Dao().FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil, err
	}

	data, _ := json.MarshalIndent(collection, "", "  ")

	return &ResourceReadResult{
		Contents: []ResourceContent{
			{
				URI:      uri,
				MimeType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}
