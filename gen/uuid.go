package gen

import "github.com/satori/go.uuid"

// NewUUID generates new UUID.
func NewUUID() string {
	return uuid.Must(uuid.NewV4()).String()
}
