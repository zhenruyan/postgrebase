// Package mcp implements the Model Context Protocol server for PostgreBase.
// MCP is a JSON-RPC 2.0 based protocol that allows AI models to interact
// with external tools and data sources.
package mcp

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/zhenruyan/postgrebase/agents"
	"github.com/zhenruyan/postgrebase/core"
)

// JSON-RPC 2.0 structures
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// MCP protocol structures
type InitializeParams struct {
	ProtocolVersion string      `json:"protocolVersion"`
	Capabilities    interface{} `json:"capabilities"`
	ClientInfo      interface{} `json:"clientInfo"`
}

type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
}

type Capabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe"`
	ListChanged bool `json:"listChanged"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"inputSchema"`
}

type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type ToolCallResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type ResourceReadParams struct {
	URI string `json:"uri"`
}

type ResourceReadResult struct {
	Contents []ResourceContent `json:"contents"`
}

type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

// Server represents the MCP server
type Server struct {
	app            core.App
	tools          map[string]ToolHandler
	resources      map[string]ResourceHandler
	mu             sync.RWMutex
	version        string
	agents         *agents.Service
	agentToolDefs  []Tool
	agentToolRoute map[string]string
}

// ToolHandler is a function that handles a tool call
type ToolHandler func(args map[string]interface{}) (*ToolCallResult, error)

// ResourceHandler is a function that reads a resource
type ResourceHandler func(uri string) (*ResourceReadResult, error)

// NewServer creates a new MCP server instance
func NewServer(app core.App, version string) *Server {
	s := &Server{
		app:            app,
		tools:          make(map[string]ToolHandler),
		resources:      make(map[string]ResourceHandler),
		version:        version,
		agents:         agents.NewService(app),
		agentToolRoute: make(map[string]string),
	}

	// Register tools
	s.registerTools()

	// Register the shared, project-scoped agent tool layer (proposal §8.4)
	s.registerAgentTools()

	// Register resources
	s.registerResources()

	return s
}

// HandleRequest processes a JSON-RPC request and returns a response
func (s *Server) HandleRequest(req *JSONRPCRequest) *JSONRPCResponse {
	if req.JSONRPC != "2.0" {
		return s.errorResponse(req.ID, InvalidRequest, "Invalid JSON-RPC version")
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	case "resources/list":
		return s.handleResourcesList(req)
	case "resources/read":
		return s.handleResourcesRead(req)
	case "ping":
		return s.successResponse(req.ID, map[string]interface{}{})
	default:
		return s.errorResponse(req.ID, MethodNotFound, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (s *Server) handleInitialize(req *JSONRPCRequest) *JSONRPCResponse {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: Capabilities{
			Tools: &ToolsCapability{
				ListChanged: false,
			},
			Resources: &ResourcesCapability{
				Subscribe:   false,
				ListChanged: false,
			},
		},
		ServerInfo: ServerInfo{
			Name:    "PostgreBase - AI-Native No-Code API Platform",
			Version: s.version,
		},
	}

	return s.successResponse(req.ID, result)
}

func (s *Server) handleToolsList(req *JSONRPCRequest) *JSONRPCResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := []Tool{
		{
			Name:        "list_collections",
			Description: "List all collections in the database",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "get_collection",
			Description: "Get detailed information about a specific collection including its schema",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"collection": map[string]interface{}{
						"type":        "string",
						"description": "Collection name or ID",
					},
				},
				"required": []string{"collection"},
			},
		},
		{
			Name:        "list_records",
			Description: "List records from a collection with optional filtering, sorting, and pagination",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"collection": map[string]interface{}{
						"type":        "string",
						"description": "Collection name or ID",
					},
					"page": map[string]interface{}{
						"type":        "integer",
						"description": "Page number (default: 1)",
					},
					"perPage": map[string]interface{}{
						"type":        "integer",
						"description": "Items per page (default: 30, max: 500)",
					},
					"filter": map[string]interface{}{
						"type":        "string",
						"description": "Filter expression (e.g., 'name=\"John\"')",
					},
					"sort": map[string]interface{}{
						"type":        "string",
						"description": "Sort expression (e.g., '-created', 'name')",
					},
				},
				"required": []string{"collection"},
			},
		},
		{
			Name:        "get_record",
			Description: "Get a single record by ID from a collection",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"collection": map[string]interface{}{
						"type":        "string",
						"description": "Collection name or ID",
					},
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Record ID",
					},
				},
				"required": []string{"collection", "id"},
			},
		},
		{
			Name:        "create_record",
			Description: "Create a new record in a collection",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"collection": map[string]interface{}{
						"type":        "string",
						"description": "Collection name or ID",
					},
					"data": map[string]interface{}{
						"type":        "object",
						"description": "Record data as JSON object",
					},
				},
				"required": []string{"collection", "data"},
			},
		},
		{
			Name:        "update_record",
			Description: "Update an existing record in a collection",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"collection": map[string]interface{}{
						"type":        "string",
						"description": "Collection name or ID",
					},
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Record ID",
					},
					"data": map[string]interface{}{
						"type":        "object",
						"description": "Record data to update as JSON object",
					},
				},
				"required": []string{"collection", "id", "data"},
			},
		},
		{
			Name:        "delete_record",
			Description: "Delete a record from a collection",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"collection": map[string]interface{}{
						"type":        "string",
						"description": "Collection name or ID",
					},
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Record ID",
					},
				},
				"required": []string{"collection", "id"},
			},
		},
		{
			Name:        "search_records",
			Description: "Search records in a collection using a PostgreBase filter expression. The query must be a valid filter expression such as 'name ~ \"John\"' (contains), 'age > 18', 'email = \"test@example.com\"', or 'created >= \"2024-01-01\"'. Supported operators: =, !=, >, <, >=, <=, ~ (like/contains), !~ (not like), ? (in). Combine with && (and) and || (or).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"collection": map[string]interface{}{
						"type":        "string",
						"description": "Collection name or ID",
					},
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Primary PostgreBase filter expression (e.g., 'name ~ \"John\"' or 'age > 18')",
					},
					"filter": map[string]interface{}{
						"type":        "string",
						"description": "Additional filter expression combined with query using AND",
					},
					"page": map[string]interface{}{
						"type":        "integer",
						"description": "Page number (default: 1)",
					},
					"perPage": map[string]interface{}{
						"type":        "integer",
						"description": "Items per page (default: 30, max: 500)",
					},
				},
				"required": []string{"collection", "query"},
			},
		},
	}

	tools = append(tools, s.agentToolDefs...)

	return s.successResponse(req.ID, map[string]interface{}{
		"tools": tools,
	})
}

func (s *Server) handleToolsCall(req *JSONRPCRequest) *JSONRPCResponse {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, InvalidParams, "Invalid parameters")
	}

	s.mu.RLock()
	handler, exists := s.tools[params.Name]
	s.mu.RUnlock()

	if !exists {
		return s.errorResponse(req.ID, MethodNotFound, fmt.Sprintf("Tool not found: %s", params.Name))
	}

	result, err := handler(params.Arguments)
	if err != nil {
		log.Printf("Tool %s error: %v", params.Name, err)
		return s.successResponse(req.ID, &ToolCallResult{
			Content: []Content{
				{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				},
			},
			IsError: true,
		})
	}

	return s.successResponse(req.ID, result)
}

func (s *Server) handleResourcesList(req *JSONRPCRequest) *JSONRPCResponse {
	resources := []Resource{
		{
			URI:         "postgrebase://collections",
			Name:        "Collections",
			Description: "List of all collections with their schemas",
			MimeType:    "application/json",
		},
		{
			URI:         "postgrebase://settings",
			Name:        "Settings",
			Description: "Application settings (sanitized)",
			MimeType:    "application/json",
		},
	}

	return s.successResponse(req.ID, map[string]interface{}{
		"resources": resources,
	})
}

func (s *Server) handleResourcesRead(req *JSONRPCRequest) *JSONRPCResponse {
	var params ResourceReadParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, InvalidParams, "Invalid parameters")
	}

	s.mu.RLock()
	handler, exists := s.resources[params.URI]
	s.mu.RUnlock()

	if !exists {
		return s.errorResponse(req.ID, MethodNotFound, fmt.Sprintf("Resource not found: %s", params.URI))
	}

	result, err := handler(params.URI)
	if err != nil {
		log.Printf("Resource %s error: %v", params.URI, err)
		return s.errorResponse(req.ID, InternalError, err.Error())
	}

	return s.successResponse(req.ID, result)
}

func (s *Server) successResponse(id interface{}, result interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

func (s *Server) errorResponse(id interface{}, code int, message string) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
}
