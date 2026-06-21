package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/zhenruyan/postgrebase/core"
)

// StdioTransport handles stdio-based MCP transport
type StdioTransport struct {
	server *Server
	reader *bufio.Reader
	writer io.Writer
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport(app core.App, version string) *StdioTransport {
	return &StdioTransport{
		server: NewServer(app, version),
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
	}
}

// Run starts the stdio transport loop
func (t *StdioTransport) Run(token string) error {
	// Authenticate if token provided
	if token != "" {
		_, err := t.server.Authenticate(token)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	log.SetOutput(os.Stderr) // Send logs to stderr to avoid interfering with stdio

	for {
		// Read line from stdin
		line, err := t.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to read from stdin: %w", err)
		}

		// Parse JSON-RPC request
		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			errResp := &JSONRPCResponse{
				JSONRPC: "2.0",
				Error: &RPCError{
					Code:    ParseError,
					Message: "Parse error: " + err.Error(),
				},
			}
			t.writeResponse(errResp)
			continue
		}

		// Handle request
		response := t.server.HandleRequest(&req)

		// Write response
		if err := t.writeResponse(response); err != nil {
			return fmt.Errorf("failed to write response: %w", err)
		}
	}
}

func (t *StdioTransport) writeResponse(resp *JSONRPCResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(t.writer, "%s\n", data)
	return err
}
