package timeutil

import (
	"time"
)

// Now returns the current local time.
func Now() time.Time {
	return time.Now()
}

// UTCNow returns the current UTC time.
func UTCNow() time.Time {
	return time.Now().UTC()
}

// Unix returns the current Unix timestamp.
func Unix() int64 {
	return time.Now().Unix()
}

// UnixMilli returns the current Unix timestamp in milliseconds.
func UnixMilli() int64 {
	return time.Now().UnixMilli()
}

// Format formats a time using the provided layout.
func Format(t time.Time, layout string) string {
	return t.Format(layout)
}

// Parse parses a string into time.Time.
func Parse(layout, value string) (time.Time, error) {
	return time.Parse(layout, value)
}

// RFC3339 formats a time in RFC3339 format.
func RFC3339(t time.Time) string {
	return t.Format(time.RFC3339)
}

// RFC3339Nano formats a time in RFC3339Nano format.
func RFC3339Nano(t time.Time) string {
	return t.Format(time.RFC3339Nano)
}