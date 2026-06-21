package cmd

import (
	"log"

	"github.com/zhenruyan/postgrebase/core"
	"github.com/zhenruyan/postgrebase/mcp"
	"github.com/spf13/cobra"
)

// NewMCPCommand creates and returns the MCP stdio command.
func NewMCPCommand(app core.App, version string) *cobra.Command {
	var mcpToken string
	var noAuth bool

	command := &cobra.Command{
		Use:   "mcp",
		Short: "Starts the MCP server in stdio mode (AI-native no-code API platform)",
		Long: `PostgreBase — AI-Native No-Code API Platform

Starts the Model Context Protocol (MCP) server in stdio mode.
This allows AI tools like Claude Desktop, Cursor, and Windsurf to connect
directly to your PostgreBase instance via stdin/stdout.

Example Claude Desktop configuration:
{
  "mcpServers": {
    "postgrebase": {
      "command": "/path/to/pb",
      "args": ["mcp", "--dataDsn", "sqlite:///path/to/dev.db", "--mcp-no-auth"]
    }
  }
}`,
		Run: func(command *cobra.Command, args []string) {
			token := mcpToken
			if noAuth {
				token = ""
			}

			transport := mcp.NewStdioTransport(app, version)
			if err := transport.Run(token); err != nil {
				log.Fatalf("MCP server error: %v", err)
			}
		},
	}

	command.PersistentFlags().StringVar(
		&mcpToken,
		"mcp-token",
		"",
		"Admin or auth record token for MCP authentication",
	)

	command.PersistentFlags().BoolVar(
		&noAuth,
		"mcp-no-auth",
		false,
		"Disable MCP authentication (NOT recommended for production)",
	)

	return command
}
