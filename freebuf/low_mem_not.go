//go:build !freebuf_low_mem

package freebuf

const (
	PartMinimalSize = 4096
	PartIncSize     = 1 << 14
)
