package freeio_test

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/duakc/mt/freebuf/freeio"
)

// These benchmarks pair freeio's buffered Reader/Writer against the bufio
// equivalents (sub-benchmarks "freeio" and "std") for direct comparison.

// BenchmarkBufWriterWriteByte — per-byte buffered writes. This is the path the
// SerialBuffer WriteByte fast path (inlinable bounds-check + store, no
// ensureFree call) is meant to make competitive with bufio's direct slice
// store. One bufferful per iteration, so the trailing Flush is amortized away.
func BenchmarkBufWriterWriteByte(b *testing.B) {
	const n = 4096
	b.Run("freeio", func(b *testing.B) {
		w := freeio.NewWriterSize(io.Discard, n)
		defer w.Free()
		b.SetBytes(n)
		b.ReportAllocs()
		for b.Loop() {
			for i := range n {
				_ = w.WriteByte(byte(i))
			}
			_ = w.Flush()
		}
	})
	b.Run("std", func(b *testing.B) {
		w := bufio.NewWriterSize(io.Discard, n)
		b.SetBytes(n)
		b.ReportAllocs()
		for b.Loop() {
			for i := range n {
				_ = w.WriteByte(byte(i))
			}
			_ = w.Flush()
		}
	})
}

// BenchmarkBufWriterWrite — many small chunked writes through the buffer.
func BenchmarkBufWriterWrite(b *testing.B) {
	chunk := payload[:256]
	const reps = 64 // 16KB total, several bufferfuls
	total := int64(len(chunk) * reps)

	b.Run("freeio", func(b *testing.B) {
		w := freeio.NewWriter(io.Discard)
		defer w.Free()
		b.SetBytes(total)
		b.ReportAllocs()
		for b.Loop() {
			for range reps {
				_, _ = w.Write(chunk)
			}
			_ = w.Flush()
		}
	})
	b.Run("std", func(b *testing.B) {
		w := bufio.NewWriter(io.Discard)
		b.SetBytes(total)
		b.ReportAllocs()
		for b.Loop() {
			for range reps {
				_, _ = w.Write(chunk)
			}
			_ = w.Flush()
		}
	})
}

// BenchmarkBufReaderReadByte — per-byte buffered reads drained to EOF.
func BenchmarkBufReaderReadByte(b *testing.B) {
	src := bytes.NewReader(payload)

	b.Run("freeio", func(b *testing.B) {
		r := freeio.NewReader(src)
		defer r.Free()
		b.SetBytes(int64(len(payload)))
		b.ReportAllocs()
		for b.Loop() {
			src.Reset(payload)
			r.Reset(src)
			for {
				if _, err := r.ReadByte(); err != nil {
					break
				}
			}
		}
	})
	b.Run("std", func(b *testing.B) {
		r := bufio.NewReader(src)
		b.SetBytes(int64(len(payload)))
		b.ReportAllocs()
		for b.Loop() {
			src.Reset(payload)
			r.Reset(src)
			for {
				if _, err := r.ReadByte(); err != nil {
					break
				}
			}
		}
	})
}

// BenchmarkBufReaderReadString — line-delimited reads (ReadSlice + copy).
func BenchmarkBufReaderReadString(b *testing.B) {
	lines := strings.Repeat("the quick brown fox jumps over the lazy dog\n", 8192)
	src := strings.NewReader(lines)

	b.Run("freeio", func(b *testing.B) {
		r := freeio.NewReader(src)
		defer r.Free()
		b.SetBytes(int64(len(lines)))
		b.ReportAllocs()
		for b.Loop() {
			src.Reset(lines)
			r.Reset(src)
			for {
				if _, err := r.ReadString('\n'); err != nil {
					break
				}
			}
		}
	})
	b.Run("std", func(b *testing.B) {
		r := bufio.NewReader(src)
		b.SetBytes(int64(len(lines)))
		b.ReportAllocs()
		for b.Loop() {
			src.Reset(lines)
			r.Reset(src)
			for {
				if _, err := r.ReadString('\n'); err != nil {
					break
				}
			}
		}
	})
}
