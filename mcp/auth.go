package mcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cast"
	"github.com/zhenruyan/postgrebase/dbx"
	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/tokens"
	"github.com/zhenruyan/postgrebase/tools/security"
)

// AuthInfo holds the authenticated user information
type AuthInfo struct {
	Admin      *models.Admin
	Record     *models.Record
	IsMCPToken bool // true if authenticated via MCP-specific token
	TokenName  string
}

// Authenticate validates the token and returns auth info
func (s *Server) Authenticate(token string) (*AuthInfo, error) {
	if token == "" {
		return nil, fmt.Errorf("authorization token is required")
	}

	// Remove Bearer prefix if present
	token = strings.TrimPrefix(token, "Bearer ")

	// Check for MCP-specific token (starts with "mcp_")
	if strings.HasPrefix(token, "mcp_") {
		return s.authenticateMCPToken(token)
	}

	// Otherwise, parse as JWT
	claims, err := security.ParseUnverifiedJWT(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token format: %w", err)
	}

	tokenType := cast.ToString(claims["type"])

	switch tokenType {
	case tokens.TypeAdmin:
		admin, err := s.app.Dao().FindAdminByToken(
			token,
			s.app.Settings().AdminAuthToken.Secret,
		)
		if err != nil || admin == nil {
			return nil, fmt.Errorf("invalid admin token")
		}
		return &AuthInfo{Admin: admin}, nil

	case tokens.TypeAuthRecord:
		record, err := s.app.Dao().FindAuthRecordByToken(
			token,
			s.app.Settings().RecordAuthToken.Secret,
		)
		if err != nil || record == nil {
			return nil, fmt.Errorf("invalid record token")
		}
		return &AuthInfo{Record: record}, nil

	default:
		return nil, fmt.Errorf("unsupported token type: %s", tokenType)
	}
}

// authenticateMCPToken validates an MCP-specific token from the _mcp_tokens collection
func (s *Server) authenticateMCPToken(token string) (*AuthInfo, error) {
	collection, err := s.app.Dao().FindCollectionByNameOrId("_pb_mcp_tokens_")
	if err != nil {
		return nil, fmt.Errorf("MCP tokens collection not found")
	}

	records := []*models.Record{}
	err = s.app.Dao().RecordQuery(collection).
		AndWhere(dbx.HashExp{"token": token}).
		Limit(1).
		All(&records)
	
	if err != nil || len(records) == 0 {
		return nil, fmt.Errorf("invalid MCP token")
	}

	record := records[0]

	// Check if token is active
	if !record.GetBool("active") {
		return nil, fmt.Errorf("MCP token is disabled")
	}

	// Check expiration if set
	expiresAt := record.GetDateTime("expiresAt")
	if !expiresAt.IsZero() {
		if expiresAt.Time().Before(time.Now()) {
			return nil, fmt.Errorf("MCP token has expired")
		}
	}

	return &AuthInfo{
		IsMCPToken: true,
		TokenName:  record.GetString("name"),
	}, nil
}

// AuthenticateStdio validates the token for stdio mode
func AuthenticateStdio(s *Server, token string) (*AuthInfo, error) {
	return s.Authenticate(token)
}
