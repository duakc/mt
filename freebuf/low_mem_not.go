//go:build !freebuf_low_mem

package freebuf

const (
	PartMinimalSize = 1024
	PartReadIncSize = 1 << 14
)
