package main

import (
	"github.com/google/uuid"
)

// NewUUID returns the string representation of a UUID
func NewUUID() string {
	v, _ := uuid.NewRandom()
	return v.String()
}