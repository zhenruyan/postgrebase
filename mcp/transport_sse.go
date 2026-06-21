package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/zhenruyan/postgrebase/core"
)

// SSEClient represents a connected SSE client
type SSEClient struct {
	ID       string
	Messages chan []byte
	Done     chan bool
}

// SSETransport handles SSE-based MCP transport over HTTP
type SSETransport struct {
	server  *Server
	clients map[string]*SSEClient
	mu      sync.RWMutex
}

// NewSSETransport creates a new SSE transport
func NewSSETransport(app core.App, version string) *SSETransport {
	return &SSETransport{
		server:  NewServer(app, version),
		clients: make(map[string]*SSEClient),
	}
}

// HandleSSE handles the SSE connection endpoint (GET /api/mcp/sse)
func (t *SSETransport) HandleSSE(c echo.Context) error {
	// Authenticate
	token := c.Request().Header.Get("Authorization")
	if token == "" {
		token = c.QueryParam("token")
	}

	_, err := t.server.Authenticate(token)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required: " + err.Error(),
		})
	}

	// Set SSE headers
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("X-Accel-Buffering", "no")

	// Create client
	clientID := fmt.Sprintf("client-%d", time.Now().UnixNano())
	client := &SSEClient{
		ID:       clientID,
		Messages: make(chan []byte, 100),
		Done:     make(chan bool),
	}

	t.mu.Lock()
	t.clients[clientID] = client
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		delete(t.clients, clientID)
		t.mu.Unlock()
		close(client.Messages)
	}()

	// Send endpoint information
	messageURL := fmt.Sprintf("/api/mcp/message?clientId=%s", clientID)
	fmt.Fprintf(c.Response(), "event: endpoint\ndata: %s\n\n", messageURL)
	c.Response().Flush()

	// Keep connection alive and send messages
	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	ctx := c.Request().Context()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-client.Done:
			return nil
		case msg := <-client.Messages:
			fmt.Fprintf(c.Response(), "event: message\ndata: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			// Send keepalive
			fmt.Fprintf(c.Response(), ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// HandleMessage handles incoming JSON-RPC messages (POST /api/mcp/message)
func (t *SSETransport) HandleMessage(c echo.Context) error {
	// Authenticate
	token := c.Request().Header.Get("Authorization")
	if token == "" {
		token = c.QueryParam("token")
	}

	_, err := t.server.Authenticate(token)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required: " + err.Error(),
		})
	}

	// Get client ID
	clientID := c.QueryParam("clientId")
	if clientID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "clientId parameter is required",
		})
	}

	// Parse request
	var req JSONRPCRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid JSON-RPC request: " + err.Error(),
		})
	}

	// Handle request
	response := t.server.HandleRequest(&req)

	// Send response via SSE if client exists
	t.mu.RLock()
	client, exists := t.clients[clientID]
	t.mu.RUnlock()

	if exists {
		data, _ := json.Marshal(response)
		select {
		case client.Messages <- data:
		default:
			log.Printf("Client %s message queue full", clientID)
		}
	}

	// Also return response directly for synchronous clients
	return c.JSON(http.StatusOK, response)
}

// HandleStreamableHTTP handles the streamable HTTP transport (POST /api/mcp/stream)
// This is a simplified version that combines request/response in a single HTTP call
func (t *SSETransport) HandleStreamableHTTP(c echo.Context) error {
	// Authenticate
	token := c.Request().Header.Get("Authorization")
	if token == "" {
		token = c.QueryParam("token")
	}

	_, err := t.server.Authenticate(token)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required: " + err.Error(),
		})
	}

	// Parse request
	var req JSONRPCRequest
	scanner := bufio.NewScanner(c.Request().Body)
	if scanner.Scan() {
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid JSON-RPC request: " + err.Error(),
			})
		}
	}

	// Handle request
	response := t.server.HandleRequest(&req)

	return c.JSON(http.StatusOK, response)
}

// BindMCPRoutes registers MCP routes on the echo instance
func BindMCPRoutes(app core.App, e *echo.Echo, version string) {
	transport := NewSSETransport(app, version)

	mcp := e.Group("/api/mcp")

	// SSE endpoint for establishing connection
	mcp.GET("/sse", transport.HandleSSE)

	// Message endpoint for sending JSON-RPC requests
	mcp.POST("/message", transport.HandleMessage)

	// Streamable HTTP endpoint (simpler alternative)
	mcp.POST("/stream", transport.HandleStreamableHTTP)
}
