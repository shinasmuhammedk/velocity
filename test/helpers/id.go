package testhelpers

import (
	"sync/atomic"
	"time"
)

var idCounter atomic.Int64

func init() {
	idCounter.Store(time.Now().UnixNano())
}

func NextID() int64 {
	return idCounter.Add(1)
}
