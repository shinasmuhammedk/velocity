package constants

import "time"

// PostgreSQL Defaults
const (
	DefaultDBMaxOpenConns = 25
	DefaultDBMaxIdleConns = 10
)

// Time Durations
const (
	DefaultDBConnMaxLifetime = 30 * time.Minute
	DefaultDBConnMaxIdleTime = 10 * time.Minute
)