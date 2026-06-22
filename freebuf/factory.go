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
// The Buffer interface carries Grow / ReadFrom* / WriteTo* / Copy uniformly
// across implementations; type-assert only for impl-specific zero-copy helpers
// (SerialBuffer.Next, MultiPartBuffer.Chunks) and inspection helpers (Cap,
// PartCount, Reset):
//
//	buf := freebuf.NewExcept(size)
//	if sb, ok := buf.(*freebuf.SerialBuffer); ok {
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

// New returns a Buffer suited for an expected payload size of n bytes, with
// space for n bytes reserved up front. Equivalent to NewExcept(n) followed by
// Grow(n). Use when the size is known in advance and you want to avoid
// grow-on-write overhead.
//
// A SerialBuffer reserves all n bytes contiguously; a MultiPartBuffer (n above
// the 64KB crossover) reserves one pool-ceiling-sized part and grows the rest
// in pooled chunks as it fills, so it never allocates a single oversized part.
func New(n int) Buffer {
	buf := NewExcept(n)
	buf.Grow(n)
	return buf
}
