//go:build !freebuf_low_mem

package freebuf

const (
	PartMinimalSize = 4096
	PartIncSize     = 1 << 14
)

// serialGrowShift sets how a SerialBuffer grows once it has outgrown the pool
// ceiling: newCap = cur + cur>>serialGrowShift. The default build favours
// throughput, so it grows 1.5x (>>1) — O(1) amortized, ~50% max overshoot.
const serialGrowShift = 1
