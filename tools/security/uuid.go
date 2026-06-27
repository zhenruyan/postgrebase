package security

import "github.com/google/uuid"

// NewUUIDString returns a new RFC 4122 UUID string.
func NewUUIDString() string {
	return uuid.NewString()
}
