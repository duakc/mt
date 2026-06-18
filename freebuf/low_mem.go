//go:build freebuf_low_mem

package freebuf

const (
	PartMinimalSize = 1024
	PartIncSize     = 1 << 8
)

// serialGrowShift sets how a SerialBuffer grows once it has outgrown the pool
// ceiling: newCap = cur + cur>>serialGrowShift. Low-memory mode favours
// footprint, so it grows 1.25x (>>2) — O(1) amortized, ~25% max overshoot.
const serialGrowShift = 2
