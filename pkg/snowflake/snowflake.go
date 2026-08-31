package snowflake

import (
	"sync"
	"time"
)

const (
	nodeBits     = 10
	sequenceBits = 12

	maxNodeID   = -1 ^ (-1 << nodeBits)
	maxSequence = -1 ^ (-1 << sequenceBits)

	nodeShift      = sequenceBits
	timestampShift = sequenceBits + nodeBits

	epoch = int64(1704067200000) // 2024-01-01 UTC
)

type Generator struct {
	mu        sync.Mutex
	nodeID    int64
	sequence  int64
	lastStamp int64
}

func New(nodeID int64) *Generator {
	if nodeID < 0 || nodeID > maxNodeID {
		panic("invalid node id")
	}

	return &Generator{
		nodeID: nodeID,
	}
}

func (g *Generator) Next() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixMilli()

	if now == g.lastStamp {
		g.sequence = (g.sequence + 1) & maxSequence

		if g.sequence == 0 {
			for now <= g.lastStamp {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		g.sequence = 0
	}

	g.lastStamp = now

	return ((now - epoch) << timestampShift) |
		(g.nodeID << nodeShift) |
		g.sequence
}
