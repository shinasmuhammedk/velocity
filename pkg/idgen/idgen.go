package idgen

import (
	"github.com/google/uuid"
)

// New returns a new UUID v7.
// Falls back to UUID v4 if v7 generation fails.
func New() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New().String()
	}

	return id.String()
}

// UUID returns a UUID object.
func UUID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}

	return id
}