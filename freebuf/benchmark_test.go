package freebuf

import (
	"bytes"
	"fmt"
	"testing"
)

var benchData64K = func() []byte {
	b := make([]byte, 64*1024)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}()

var benchData256K = func() []byte {
	b := make([]byte, 256*1024)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}()

// One Write of 64KB into a fresh buffer per op. Results: see BENCHMARK.md.
func BenchmarkBufferWrite_64KB(b *testing.B) {
	data := benchData64K
	b.Run("BytesBuffer", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))
		for range b.N {
			var buf bytes.Buffer
			buf.Write(data)
		}
	})
	b.Run("SerialBuffer", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))
		for range b.N {
			buf := NewSerial()
			buf.Write(data)
			buf.FreeMe()
		}
	})
	b.Run("MultiPartBuffer", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))
		for range b.N {
			mp := NewMultiPart()
			mp.Write(data)
			mp.FreeMe()
		}
	})
	b.Run("SerialBufferLimited", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))
		for range b.N {
			buf := NewSerialLimited(len(data))
			buf.Write(data)
			buf.FreeMe()
		}
	})
}

// 4096 single-byte writes into a fresh buffer per op. Results: see BENCHMARK.md.
func BenchmarkBufferWriteByte_4K(b *testing.B) {
	const count = 4096
	b.Run("BytesBuffer", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(count))
		for range b.N {
			var buf bytes.Buffer
			for i := range count {
				buf.WriteByte(byte(i))
			}
		}
	})
	b.Run("SerialBuffer", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(count))
		for range b.N {
			buf := NewSerial()
			for i := range count {
				buf.WriteByte(byte(i))
			}
			buf.FreeMe()
		}
	})
	b.Run("MultiPartBuffer", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(count))
		for range b.N {
			mp := NewMultiPart()
			for i := range count {
				mp.WriteByte(byte(i))
			}
			mp.FreeMe()
		}
	})
}

// ReadFrom a 256KB in-memory source into a fresh buffer per op, exercising the
// grow-on-demand path past the 64KB pool ceiling. Results: see BENCHMARK.md.
func BenchmarkBufferReadFrom_256KB(b *testing.B) {
	src := benchData256K
	b.Run("BytesBuffer", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(src)))
		for range b.N {
			var buf bytes.Buffer
			buf.ReadFrom(bytes.NewReader(src))
		}
	})
	b.Run("SerialBuffer", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(src)))
		for range b.N {
			buf := NewSerial()
			buf.ReadFrom(bytes.NewReader(src))
			buf.FreeMe()
		}
	})
	b.Run("MultiPartBuffer", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(src)))
		for range b.N {
			mp := NewMultiPart()
			mp.ReadFrom(bytes.NewReader(src))
			mp.FreeMe()
		}
	})
}

// Write 64KB then read it all back, fresh buffer per op — the producer/
// consumer pattern. Results: see BENCHMARK.md.
func BenchmarkBufferRoundTrip_64KB(b *testing.B) {
	data := benchData64K
	sink := make([]byte, len(data))
	b.Run("BytesBuffer", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))
		for range b.N {
			var buf bytes.Buffer
			buf.Write(data)
			buf.Read(sink)
		}
	})
	b.Run("SerialBuffer", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))
		for range b.N {
			buf := NewSerial()
			buf.Write(data)
			buf.Read(sink)
			buf.FreeMe()
		}
	})
	b.Run("MultiPartBuffer", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))
		for range b.N {
			mp := NewMultiPart()
			mp.Write(data)
			mp.Read(sink)
			mp.FreeMe()
		}
	})
	b.Run("SerialBufferLimited", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))
		for range b.N {
			buf := NewSerialLimited(len(data))
			buf.Write(data)
			buf.Read(sink)
			buf.FreeMe()
		}
	})
}

// Write+Read roundtrip across a range of payload sizes to locate the crossover
// where MultiPartBuffer overtakes SerialBuffer. The threshold backing
// NewExcept is taken from this data. Results: see BENCHMARK.md.
func BenchmarkBufferAcrossSizes(b *testing.B) {
	sizes := []int{
		16 * 1024,
		32 * 1024,
		48 * 1024,
		64 * 1024,
		80 * 1024,
		96 * 1024,
		128 * 1024,
		192 * 1024,
		256 * 1024,
	}
	for _, size := range sizes {
		data := make([]byte, size)
		sink := make([]byte, size)
		name := fmt.Sprintf("%dKB", size/1024)
		b.Run(name+"/Serial", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(size))
			for range b.N {
				buf := NewSerial()
				buf.Write(data)
				buf.Read(sink)
				buf.FreeMe()
			}
		})
		b.Run(name+"/MultiPart", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(size))
			for range b.N {
				buf := NewMultiPart()
				buf.Write(data)
				buf.Read(sink)
				buf.FreeMe()
			}
		})
	}
}
