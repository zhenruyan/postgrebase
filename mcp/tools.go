package mcp

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cast"
	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/resolvers"
	"github.com/zhenruyan/postgrebase/tools/search"
)

// registerTools registers all available tools
func (s *Server) registerTools() {
	s.tools["list_collections"] = s.toolListCollections
	s.tools["get_collection"] = s.toolGetCollection
	s.tools["list_records"] = s.toolListRecords
	s.tools["get_record"] = s.toolGetRecord
	s.tools["create_record"] = s.toolCreateRecord
	s.tools["update_record"] = s.toolUpdateRecord
	s.tools["delete_record"] = s.toolDeleteRecord
	s.tools["search_records"] = s.toolSearchRecords
}

// toolListCollections lists all collections
func (s *Server) toolListCollections(args map[string]interface{}) (*ToolCallResult, error) {
	collections := []*models.Collection{}
	err := s.app.Dao().CollectionQuery().OrderBy("created ASC").All(&collections)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	result := make([]map[string]interface{}, 0, len(collections))
	for _, c := range collections {
		item := map[string]interface{}{
			"id":          c.Id,
			"name":        c.Name,
			"type":        c.Type,
			"system":      c.System,
			"displayName": c.DisplayName,
			"created":     c.Created,
			"updated":     c.Updated,
		}
		result = append(result, item)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &ToolCallResult{
		Content: []Content{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// toolGetCollection gets detailed information about a collection
func (s *Server) toolGetCollection(args map[string]interface{}) (*ToolCallResult, error) {
	collectionName, ok := args["collection"].(string)
	if !ok || collectionName == "" {
		return nil, fmt.Errorf("collection parameter is required")
	}

	collection, err := s.app.Dao().FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	data, _ := json.MarshalIndent(collection, "", "  ")
	return &ToolCallResult{
		Content: []Content{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// toolListRecords lists records from a collection
func (s *Server) toolListRecords(args map[string]interface{}) (*ToolCallResult, error) {
	collectionName, ok := args["collection"].(string)
	if !ok || collectionName == "" {
		return nil, fmt.Errorf("collection parameter is required")
	}

	collection, err := s.app.Dao().FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Build query parameters
	page := 1
	perPage := 30
	filter := ""
	sort := ""

	if v, ok := args["page"]; ok {
		page = cast.ToInt(v)
		if page < 1 {
			page = 1
		}
	}

	if v, ok := args["perPage"]; ok {
		perPage = cast.ToInt(v)
		if perPage < 1 {
			perPage = 30
		}
		if perPage > 500 {
			perPage = 500
		}
	}

	if v, ok := args["filter"].(string); ok {
		filter = v
	}

	if v, ok := args["sort"].(string); ok {
		sort = v
	}

	// Build URL-encoded query string (ParseAndExec uses url.ParseQuery internally)
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	params.Set("perPage", strconv.Itoa(perPage))
	if filter != "" {
		params.Set("filter", filter)
	}
	if sort != "" {
		params.Set("sort", sort)
	}
	queryStr := params.Encode()

	records := []*models.Record{}

	fieldsResolver := resolvers.NewRecordFieldResolver(
		s.app.Dao(),
		collection,
		nil, // requestInfo - nil for MCP (admin-like access)
		true,
	)

	searchProvider := search.NewProvider(fieldsResolver).
		Query(s.app.Dao().RecordQuery(collection))

	result, err := searchProvider.ParseAndExec(queryStr, &records)
	if err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &ToolCallResult{
		Content: []Content{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// toolGetRecord gets a single record by ID
func (s *Server) toolGetRecord(args map[string]interface{}) (*ToolCallResult, error) {
	collectionName, ok := args["collection"].(string)
	if !ok || collectionName == "" {
		return nil, fmt.Errorf("collection parameter is required")
	}

	recordID, ok := args["id"].(string)
	if !ok || recordID == "" {
		return nil, fmt.Errorf("id parameter is required")
	}

	collection, err := s.app.Dao().FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	record, err := s.app.Dao().FindRecordById(collection.Id, recordID)
	if err != nil {
		return nil, fmt.Errorf("record not found: %s", recordID)
	}

	data, _ := json.MarshalIndent(record, "", "  ")
	return &ToolCallResult{
		Content: []Content{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// toolCreateRecord creates a new record
func (s *Server) toolCreateRecord(args map[string]interface{}) (*ToolCallResult, error) {
	collectionName, ok := args["collection"].(string)
	if !ok || collectionName == "" {
		return nil, fmt.Errorf("collection parameter is required")
	}

	dataArg, ok := args["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data parameter is required and must be an object")
	}

	collection, err := s.app.Dao().FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	record := models.NewRecord(collection)

	// Set the data fields
	for key, value := range dataArg {
		record.Set(key, value)
	}

	// Save the record
	if err := s.app.Dao().SaveRecord(record); err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}

	data, _ := json.MarshalIndent(record, "", "  ")
	return &ToolCallResult{
		Content: []Content{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// toolUpdateRecord updates an existing record
func (s *Server) toolUpdateRecord(args map[string]interface{}) (*ToolCallResult, error) {
	collectionName, ok := args["collection"].(string)
	if !ok || collectionName == "" {
		return nil, fmt.Errorf("collection parameter is required")
	}

	recordID, ok := args["id"].(string)
	if !ok || recordID == "" {
		return nil, fmt.Errorf("id parameter is required")
	}

	dataArg, ok := args["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data parameter is required and must be an object")
	}

	collection, err := s.app.Dao().FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	record, err := s.app.Dao().FindRecordById(collection.Id, recordID)
	if err != nil {
		return nil, fmt.Errorf("record not found: %s", recordID)
	}

	// Update the data fields
	for key, value := range dataArg {
		record.Set(key, value)
	}

	// Save the record
	if err := s.app.Dao().SaveRecord(record); err != nil {
		return nil, fmt.Errorf("failed to update record: %w", err)
	}

	data, _ := json.MarshalIndent(record, "", "  ")
	return &ToolCallResult{
		Content: []Content{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// toolDeleteRecord deletes a record
func (s *Server) toolDeleteRecord(args map[string]interface{}) (*ToolCallResult, error) {
	collectionName, ok := args["collection"].(string)
	if !ok || collectionName == "" {
		return nil, fmt.Errorf("collection parameter is required")
	}

	recordID, ok := args["id"].(string)
	if !ok || recordID == "" {
		return nil, fmt.Errorf("id parameter is required")
	}

	collection, err := s.app.Dao().FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	record, err := s.app.Dao().FindRecordById(collection.Id, recordID)
	if err != nil {
		return nil, fmt.Errorf("record not found: %s", recordID)
	}

	if err := s.app.Dao().DeleteRecord(record); err != nil {
		return nil, fmt.Errorf("failed to delete record: %w", err)
	}

	return &ToolCallResult{
		Content: []Content{
			{
				Type: "text",
				Text: fmt.Sprintf("Record %s deleted successfully", recordID),
			},
		},
	}, nil
}

// toolSearchRecords searches records using filter expressions
func (s *Server) toolSearchRecords(args map[string]interface{}) (*ToolCallResult, error) {
	collectionName, ok := args["collection"].(string)
	if !ok || collectionName == "" {
		return nil, fmt.Errorf("collection parameter is required")
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	collection, err := s.app.Dao().FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Build query parameters
	page := 1
	perPage := 30

	if v, ok := args["page"]; ok {
		page = cast.ToInt(v)
		if page < 1 {
			page = 1
		}
	}

	if v, ok := args["perPage"]; ok {
		perPage = cast.ToInt(v)
		if perPage < 1 {
			perPage = 30
		}
		if perPage > 500 {
			perPage = 500
		}
	}

	// Build filter from query
	filter := query
	if v, ok := args["filter"].(string); ok && v != "" {
		filter = fmt.Sprintf("(%s) && (%s)", query, v)
	}

	// Build URL-encoded query string (ParseAndExec uses url.ParseQuery internally)
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	params.Set("perPage", strconv.Itoa(perPage))
	params.Set("filter", filter)
	queryStr := params.Encode()

	records := []*models.Record{}

	fieldsResolver := resolvers.NewRecordFieldResolver(
		s.app.Dao(),
		collection,
		nil,
		true,
	)

	searchProvider := search.NewProvider(fieldsResolver).
		Query(s.app.Dao().RecordQuery(collection))

	result, err := searchProvider.ParseAndExec(queryStr, &records)
	if err != nil {
		return nil, fmt.Errorf("failed to search records: %w", err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &ToolCallResult{
		Content: []Content{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// registerAgentTools exposes the shared, project-scoped agent tool layer
// (schema.* / data.* / dataset.*) over MCP, reusing the exact same executors
// as the embedded agent runtime and the REST API (proposal §4.3 single
// business kernel, §8.4 reuse to MCP). Tool names are namespaced with an
// "agent_" prefix and dots are replaced with underscores for MCP compatibility.
func (s *Server) registerAgentTools() {
	if s.agents == nil {
		return
	}
	for _, spec := range s.agents.Tools() {
		mcpName := "agent_" + strings.ReplaceAll(spec.Name, ".", "_")
		s.agentToolRoute[mcpName] = spec.Name
		s.tools[mcpName] = s.makeAgentToolHandler(spec.Name)

		description := spec.Description
		if spec.Category == "write" {
			description += " [write]"
		}
		s.agentToolDefs = append(s.agentToolDefs, Tool{
			Name:        mcpName,
			Description: description + " (project-scoped; requires 'project')",
			InputSchema: spec.InputSchema,
		})
	}
}

// makeAgentToolHandler routes an MCP tool call to the shared agent executor.
func (s *Server) makeAgentToolHandler(dottedName string) ToolHandler {
	return func(args map[string]interface{}) (*ToolCallResult, error) {
		result, err := s.agents.ExecuteTool(dottedName, args)
		if err != nil {
			return nil, err
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		return &ToolCallResult{
			Content: []Content{
				{
					Type: "text",
					Text: string(data),
				},
			},
		}, nil
	}
}
