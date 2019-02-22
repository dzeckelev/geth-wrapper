package gen

import "github.com/pborman/uuid"

// NewUUID generates new UUID.
func NewUUID() string {
	return uuid.NewUUID().String()
}
