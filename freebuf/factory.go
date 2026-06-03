package freebuf

import "github.com/duakc/mt/freebuf/internal"

// serialMultiPartCrossover is the payload size at which MultiPartBuffer
// overtakes SerialBuffer in the Write+Read roundtrip benchmarks (see
// BENCHMARK.md / BenchmarkBufferAcrossSizes). The value coincides with the
// pool ceiling: once the backing slice for a contiguous buffer outgrows
// internal.MaxAllocatableSize, it falls off the sync.Pool fast path and the
// doubling memcpy cost lets the chunked design pull ahead.
const serialMultiPartCrossover = internal.MaxAllocatableSize

// NewExcept returns a Buffer sized for the caller's expected payload. It does
// not allocate backing storage; the choice is purely about which
// implementation will handle `except` bytes most efficiently:
//
//   - except <= 64KB: SerialBuffer (contiguous, pool-backed, zero realloc)
//   - except >  64KB: MultiPartBuffer (chunked, appends pooled parts without
//     memcpy when growing)
//
// Pass the largest payload size the buffer is expected to hold at once. If
// the actual payload exceeds the hint the buffer still grows correctly, just
// with the implementation chosen for `except` rather than the actual size.
//
// The Buffer interface intentionally omits the implementation-specific
// helpers each concrete type carries (zero-copy access, Reset without
// freeing, iteration over chunks, ...). Type-assert when you need them:
//
//	buf := freebuf.NewExcept(size)
//	if sb, ok := buf.(*freebuf.SerialBuffer); ok {
//	    sb.Grow(extra)
//	    chunk := sb.Next(n) // zero-copy
//	}
//	if mp, ok := buf.(*freebuf.MultiPartBuffer); ok {
//	    for c := range mp.Chunks() { hash.Write(c) } // zero-copy
//	}
func NewExcept(except int) Buffer {
	if except <= serialMultiPartCrossover {
		return NewSerial()
	}
	return NewMultiPart()
}
