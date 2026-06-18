package freeio

import (
	"io"
	"os"
	"syscall"

	"github.com/duakc/mt/freebuf"
)

const (
	// copyChunkSize is the staging buffer size for the userspace fallback. 32KB
	// matches io.Copy's own default scratch size and lands exactly on freebuf's
	// 32KB pool bucket, so the buffer is pooled rather than heap-allocated.
	copyChunkSize = 32 * 1024

	// oneShotLimit is freebuf's pool max single-chunk allocation (64KB,
	// internal.MaxAllocatableSize). A known length at or below it is staged in
	// one contiguous pooled buffer and read in a single pass; above it, pooling
	// no longer applies and we read in fixed chunks instead.
	oneShotLimit = 1 << 16

	// maxZeroCopyChunk caps the bytes asked of one sendfile/copy_file_range
	// syscall. Counters fire once per syscall, so this also bounds counting
	// granularity; 1MB matches splice's pipe window (below) for a uniform feel.
	maxZeroCopyChunk = 1 << 20
)

// Copy copies src to dst until EOF, returning the byte count and the first
// error. A successful copy returns nil, not io.EOF. It prefers the kernel
// zero-copy paths (splice / sendfile / copy_file_range) and otherwise falls back
// to a pooled buffered copy.
func Copy(dst io.Writer, src io.Reader) (int64, error) {
	return CopyWithCounter(dst, src, nil, nil)
}

// CopyWithCounter is Copy with progress reported per chunk: readCounters for
// bytes from src, writeCounters for bytes to dst.
//
// src and dst are unwrapped (see UnwrapReadCounter) to expose the concrete
// fd-backed stream under any counter wrappers — the kernel zero-copy paths need
// it — and the wrappers' own counters are merged with the explicit ones for
// those paths. When no zero-copy path applies the copy falls back through the
// *original* src and dst, so any counter wrappers tally themselves and only the
// explicit counters are added per chunk; nothing is re-wrapped.
func CopyWithCounter(dst io.Writer, src io.Reader, writeCounters, readCounters []CounterFunc) (int64, error) {
	rawSrc, embeddedRead := UnwrapReadCounter(src)
	rawDst, embeddedWrite := UnwrapWriteCounter(dst)
	readAll := mergeCounters(readCounters, embeddedRead)
	writeAll := mergeCounters(writeCounters, embeddedWrite)

	if len(readAll) == 0 && len(writeAll) == 0 {
		return copyFast(rawDst, rawSrc)
	}

	if n, handled, err := kernelZeroCopy(rawDst, rawSrc, writeAll, readAll); handled {
		return n, err
	}

	// Fall back through the original src/dst (wrappers count themselves); the
	// explicit counters are tallied per chunk by the loop.
	return copyGeneric(dst, src, writeCounters, readCounters)
}

// CopyBuffer is Copy staged through a caller-supplied scratch Buffer instead of
// an internally pooled one. The buffer is reset on entry and drained on return.
// It is ignored when a zero-copy fast path applies; a nil buffer behaves as Copy.
func CopyBuffer(dst io.Writer, src io.Reader, buffer freebuf.Buffer) (int64, error) {
	if buffer == nil {
		return Copy(dst, src)
	}
	if n, ok, err := fastCopy(dst, src); ok {
		return n, err
	}
	return copyWithBuffer(dst, src, buffer, nil, nil)
}

// fastCopy takes whichever single-shot built-in the standard library offers —
// WriteTo/ReadFrom drive bytes.Buffer, bufio, *os.File, net conns, ... directly,
// with no intermediate buffer, and reach the kernel zero-copy paths. ok is false
// when neither side supports it and a manual copy is required.
func fastCopy(dst io.Writer, src io.Reader) (n int64, ok bool, err error) {
	if wt, isWT := src.(io.WriterTo); isWT {
		n, err = wt.WriteTo(dst)
		return n, true, err
	}
	if rf, isRF := dst.(io.ReaderFrom); isRF {
		n, err = rf.ReadFrom(src)
		return n, true, err
	}
	return 0, false, nil
}

func copyFast(dst io.Writer, src io.Reader) (int64, error) {
	if n, ok, err := fastCopy(dst, src); ok {
		return n, err
	}
	return copyGeneric(dst, src, nil, nil)
}

// kernelZeroCopy drives the kernel directly, counting per syscall (the finest
// real-time granularity) — the standard library's WriteTo/ReadFrom would move
// everything in one opaque call with no per-chunk visibility. src and dst must
// be the unwrapped fds. It reports handled == false when no path applies, each
// step falling through likewise:
//
//  1. copy_file_range (file->file): one in-kernel call, reflink-capable on
//     supporting filesystems — the cheapest file copy, so it goes first.
//  2. splice (any socket pair): socket<->socket, file->socket and socket->file
//     via a pipe; counts per splice.
//  3. sendfile (file->socket): only as the splice fallback, for kernels too old
//     for splice(2).
func kernelZeroCopy(dst io.Writer, src io.Reader, writeCounters, readCounters []CounterFunc) (n int64, handled bool, err error) {
	srcConn, srcOK := src.(syscall.Conn)
	dstConn, dstOK := dst.(syscall.Conn)
	if !srcOK || !dstOK {
		return 0, false, nil
	}

	srcFile := isRegularFile(src)
	dstFile := isRegularFile(dst)

	if srcFile && dstFile {
		if n, h, e := copyFileRangeConn(srcConn, dstConn, writeCounters, readCounters); h {
			return n, true, e
		}
	}
	if n, h, e := spliceConn(srcConn, dstConn, writeCounters, readCounters); h {
		return n, true, e
	}
	if srcFile {
		if n, h, e := sendfileConn(srcConn, dstConn, writeCounters, readCounters); h {
			return n, true, e
		}
	}
	return 0, false, nil
}

// copyGeneric is the userspace fallback: CopyBuffer with a buffer we build
// instead of borrow. When src's length is known (a *io.LimitedReader or a Len()
// method) a small payload is staged in one contiguous pooled buffer; an unknown
// or large length uses a fixed chunk read repeatedly.
func copyGeneric(dst io.Writer, src io.Reader, writeCounters, readCounters []CounterFunc) (int64, error) {
	size := copyChunkSize
	if n, ok := knownLen(src); ok && n > 0 && n <= oneShotLimit {
		size = int(n)
	}
	buffer := freebuf.New(size)
	defer buffer.FreeMe()
	return copyWithBuffer(dst, src, buffer, writeCounters, readCounters)
}

func copyWithBuffer(dst io.Writer, src io.Reader, buffer freebuf.Buffer, writeCounters, readCounters []CounterFunc) (written int64, err error) {
	buffer.Reset()
	for {
		nr, rerr := buffer.ReadFromOnce(src)
		if nr > 0 {
			for _, c := range readCounters {
				c(int64(nr))
			}
			nw, werr := buffer.WriteTo(dst)
			if nw > 0 {
				written += nw
				for _, c := range writeCounters {
					c(nw)
				}
			}
			if werr != nil && werr != io.EOF {
				return written, werr
			}
		}
		if rerr != nil {
			if rerr != io.EOF {
				err = rerr
			}
			return written, err
		}
	}
}

func knownLen(src io.Reader) (int64, bool) {
	switch v := src.(type) {
	case *io.LimitedReader:
		return v.N, true
	case interface{ Len() int }:
		return int64(v.Len()), true
	}
	return 0, false
}

func isRegularFile(v any) bool {
	f, ok := v.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode().IsRegular()
}

func mergeCounters(a, b []CounterFunc) []CounterFunc {
	if len(b) == 0 {
		return a
	}
	if len(a) == 0 {
		return b
	}
	return append(append(make([]CounterFunc, 0, len(a)+len(b)), a...), b...)
}
