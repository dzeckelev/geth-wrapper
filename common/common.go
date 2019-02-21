package common

import "github.com/pborman/uuid"

func NewUUID() string {
	return uuid.NewUUID().String()
}
