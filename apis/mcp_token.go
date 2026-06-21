package apis

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/zhenruyan/postgrebase/core"
	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/tools/security"
)

// bindMcpTokenApi registers MCP token management API endpoints
func bindMcpTokenApi(app core.App, rg *echo.Group) {
	api := mcpTokenApi{app: app}

	subGroup := rg.Group("/mcp-tokens", ActivityLogger(app), RequireAdminAuth())
	subGroup.GET("", api.list)
	subGroup.POST("", api.create)
	subGroup.DELETE("/:id", api.delete)
	subGroup.POST("/generate", api.generate)
}

type mcpTokenApi struct {
	app core.App
}

// list returns all MCP tokens
func (api *mcpTokenApi) list(c echo.Context) error {
	collection, err := api.app.Dao().FindCollectionByNameOrId("_pb_mcp_tokens_")
	if err != nil {
		return NewNotFoundError("MCP tokens collection not found", err)
	}

	records := []*models.Record{}
	err = api.app.Dao().RecordQuery(collection).
		OrderBy("created DESC").
		All(&records)
	if err != nil {
		return NewBadRequestError("Failed to fetch MCP tokens", err)
	}

	// Return tokens without exposing the full token value (show only first 8 chars + "...")
	result := make([]map[string]interface{}, 0, len(records))
	for _, r := range records {
		token := r.GetString("token")
		maskedToken := ""
		if len(token) > 8 {
			maskedToken = token[:8] + "..."
		} else if len(token) > 0 {
			maskedToken = "****"
		}

		item := map[string]interface{}{
			"id":          r.Id,
			"name":        r.GetString("name"),
			"token":       maskedToken,
			"description": r.GetString("description"),
			"active":      r.GetBool("active"),
			"expiresAt":   r.GetDateTime("expiresAt"),
			"created":     r.Created,
			"updated":     r.Updated,
		}
		result = append(result, item)
	}

	return c.JSON(http.StatusOK, result)
}

// create creates a new MCP token
func (api *mcpTokenApi) create(c echo.Context) error {
	collection, err := api.app.Dao().FindCollectionByNameOrId("_pb_mcp_tokens_")
	if err != nil {
		return NewNotFoundError("MCP tokens collection not found", err)
	}

	// Parse request body
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		ExpiresDays int    `json:"expiresDays"` // 0 = never expires
	}
	if err := c.Bind(&body); err != nil {
		return NewBadRequestError("Invalid request body", err)
	}

	if body.Name == "" {
		return NewBadRequestError("Name is required", nil)
	}

	// Generate a secure token
	token := "mcp_" + security.RandomString(48)

	// Create the record
	record := models.NewRecord(collection)
	record.Set("name", body.Name)
	record.Set("token", token)
	record.Set("description", body.Description)
	record.Set("active", true)

	// Set expiration if specified
	if body.ExpiresDays > 0 {
		expiresAt := time.Now().Add(time.Duration(body.ExpiresDays) * 24 * time.Hour)
		record.Set("expiresAt", expiresAt)
	}

	if err := api.app.Dao().SaveRecord(record); err != nil {
		return NewBadRequestError("Failed to create MCP token", err)
	}

	// Return the full token only on creation (user should copy it immediately)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":          record.Id,
		"name":        record.GetString("name"),
		"token":       token, // Full token shown only once
		"description": record.GetString("description"),
		"active":      record.GetBool("active"),
		"expiresAt":   record.GetDateTime("expiresAt"),
		"created":     record.Created,
		"updated":     record.Updated,
	})
}

// delete revokes/deletes an MCP token
func (api *mcpTokenApi) delete(c echo.Context) error {
	id := c.PathParam("id")
	if id == "" {
		return NewNotFoundError("Token ID is required", nil)
	}

	collection, err := api.app.Dao().FindCollectionByNameOrId("_pb_mcp_tokens_")
	if err != nil {
		return NewNotFoundError("MCP tokens collection not found", err)
	}

	record, err := api.app.Dao().FindRecordById(collection.Id, id)
	if err != nil {
		return NewNotFoundError("Token not found", err)
	}

	if err := api.app.Dao().DeleteRecord(record); err != nil {
		return NewBadRequestError("Failed to delete MCP token", err)
	}

	return c.NoContent(http.StatusNoContent)
}

// generate generates a new token for an existing record (re-generates the token value)
func (api *mcpTokenApi) generate(c echo.Context) error {
	collection, err := api.app.Dao().FindCollectionByNameOrId("_pb_mcp_tokens_")
	if err != nil {
		return NewNotFoundError("MCP tokens collection not found", err)
	}

	// Parse request body
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		ExpiresDays int    `json:"expiresDays"`
	}
	if err := c.Bind(&body); err != nil {
		return NewBadRequestError("Invalid request body", err)
	}

	if body.Name == "" {
		return NewBadRequestError("Name is required", nil)
	}

	// Generate a secure token
	token := "mcp_" + security.RandomString(48)

	// Create the record
	record := models.NewRecord(collection)
	record.Set("name", body.Name)
	record.Set("token", token)
	record.Set("description", body.Description)
	record.Set("active", true)

	if body.ExpiresDays > 0 {
		expiresAt := time.Now().Add(time.Duration(body.ExpiresDays) * 24 * time.Hour)
		record.Set("expiresAt", expiresAt)
	}

	if err := api.app.Dao().SaveRecord(record); err != nil {
		return NewBadRequestError("Failed to generate MCP token", err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":          record.Id,
		"name":        record.GetString("name"),
		"token":       token,
		"description": record.GetString("description"),
		"active":      record.GetBool("active"),
		"expiresAt":   record.GetDateTime("expiresAt"),
		"created":     record.Created,
		"updated":     record.Updated,
	})
}
