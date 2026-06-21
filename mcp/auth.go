package mcp

import (
	"fmt"
	"strings"

	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/tokens"
	"github.com/zhenruyan/postgrebase/tools/security"
	"github.com/spf13/cast"
)

// AuthInfo holds the authenticated user information
type AuthInfo struct {
	Admin  *models.Admin
	Record *models.Record
}

// Authenticate validates the token and returns auth info
func (s *Server) Authenticate(token string) (*AuthInfo, error) {
	if token == "" {
		return nil, fmt.Errorf("authorization token is required")
	}

	// Remove Bearer prefix if present
	token = strings.TrimPrefix(token, "Bearer ")

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

// AuthenticateStdio validates the token for stdio mode
func AuthenticateStdio(s *Server, token string) (*AuthInfo, error) {
	return s.Authenticate(token)
}
