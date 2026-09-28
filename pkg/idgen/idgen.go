package idgen

import (
	"sync"

	"velocity/pkg/snowflake"

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

// defaultNodeID keeps existing callers (including tests that never call
// Init) working unchanged. Any process that runs more than one Velocity
// instance concurrently against the same trade/order ID space MUST call
// Init with a distinct node ID per instance before generating any IDs,
// or IDs can collide across instances.
const defaultNodeID = 0

var (
	genMu sync.Mutex
	gen   = snowflake.New(defaultNodeID)
)

// Init (re)seeds the package-level Snowflake generator with the given
// node ID. Call this once at process startup, before any engine starts
// generating trade/order IDs via Next(). Safe to call from tests too,
// but note it mutates shared package state - concurrent tests that both
// call Init and expect a particular node ID will race with each other.
func Init(nodeID int64) {
	genMu.Lock()
	defer genMu.Unlock()
	gen = snowflake.New(nodeID)
}

func Next() int64 {
	genMu.Lock()
	g := gen
	genMu.Unlock()
	return g.Next()
}