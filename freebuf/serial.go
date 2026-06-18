package freebuf

import (
	"io"

	"github.com/duakc/mt/freebuf/internal"
)

var _ Buffer = (*SerialBuffer)(nil)

// SerialBuffer is a contiguous, growable Buffer. Writes grow the backing
// storage on demand; reads compact the unread region back to the front so the
// prefix space can be reused without reallocating.
//
// A SerialBuffer can optionally be created with a hard capacity ceiling via
// NewSerialLimited. Writes that would push past the ceiling write what they
// can and return io.ErrShortBuffer for the remainder. Reads still free space
// so the buffer can be cycled like a bounded queue.
type SerialBuffer struct {
	data  []byte
	r, w  int
	limit int // 0 means unlimited
}

// NewSerial returns an unlimited SerialBuffer with no backing allocation.
// The first write triggers the initial allocation.
func NewSerial() *SerialBuffer {
	return &SerialBuffer{}
}

// NewSerialLimited returns a SerialBuffer whose backing storage will not grow
// beyond `limit` bytes. The full backing slice is allocated up front.
// A non-positive `limit` is treated as unlimited (same as NewSerial).
func NewSerialLimited(limit int) *SerialBuffer {
	if limit <= 0 {
		return &SerialBuffer{}
	}
	b := &SerialBuffer{limit: limit}
	data := getSerialBuffer(limit)
	if len(data) > limit {
		data = data[:limit]
	}
	b.data = data
	return b
}

func (b *SerialBuffer) Bytes() []byte {
	return b.data[b.r:b.w]
}

func (b *SerialBuffer) FreeBytes() []byte {
	return b.data[b.w:]
}

// Copy returns a deep copy of the unread bytes in a fresh SerialBuffer. The
// limit is preserved; a limited source produces a limited copy with the same
// ceiling pre-allocated. Read/write cursors on the copy start at the
// beginning. The returned Buffer is the caller's to FreeMe.
func (b *SerialBuffer) Copy() Buffer {
	if b.limit > 0 {
		cp := NewSerialLimited(b.limit)
		if b.w > b.r {
			cp.w = copy(cp.data, b.data[b.r:b.w])
		}
		return cp
	}
	n := b.w - b.r
	if n == 0 {
		return NewSerial()
	}
	cp := &SerialBuffer{data: getSerialBuffer(n)}
	cp.w = copy(cp.data, b.data[b.r:b.w])
	return cp
}

func (b *SerialBuffer) Truncated(n int) {
	if n < 0 {
		n = 0
	}
	available := len(b.data) - b.r
	if n > available {
		n = available
	}
	b.w = b.r + n
}

// Grow tries to ensure at least n bytes of free space at b.w. On a limited
// buffer it is best-effort: if n exceeds the remaining headroom under the
// limit, only the remaining headroom is allocated.
func (b *SerialBuffer) Grow(n int) {
	if n > 0 {
		b.ensureFree(n)
	}
}

func (b *SerialBuffer) Write(p []byte) (n int, err error) {
	b.ensureFree(len(p))
	n = copy(b.data[b.w:], p)
	b.w += n
	if n < len(p) {
		err = io.ErrShortBuffer
	}
	return n, err
}

func (b *SerialBuffer) WriteString(s string) (n int, err error) {
	b.ensureFree(len(s))
	n = copy(b.data[b.w:], s)
	b.w += n
	if n < len(s) {
		err = io.ErrShortBuffer
	}
	return n, err
}

// WriteByte is ~30–45% slower than bytes.Buffer.WriteByte in tight loops (see
// BENCHMARK.md, BenchmarkBufferWriteByte_4K). The reason is inlining:
// bytes.Buffer.WriteByte uses tryGrowByReslice — a 5-line helper the compiler
// inlines, collapsing the hot path to "cap-len >= 1 → reslice → store byte".
// ensureFree here is too large to inline (multiple branches, in-place compact,
// pool Get/Put on growth), so every per-byte call pays a function-call
// prologue/epilogue. Keeping the growth logic in one helper makes the bulk
// paths (Write, WriteString, ReadFrom) simpler and stay equally fast; the
// price is paid on per-byte writes. If your hot path is dominated by
// WriteByte, reach for bytes.Buffer directly.
func (b *SerialBuffer) WriteByte(c byte) error {
	b.ensureFree(1)
	if len(b.data)-b.w < 1 {
		return io.ErrShortBuffer
	}
	b.data[b.w] = c
	b.w++
	return nil
}

func (b *SerialBuffer) Read(p []byte) (n int, err error) {
	if b.r == b.w {
		return 0, io.EOF
	}
	n = copy(p, b.data[b.r:b.w])
	b.r += n
	if b.r == b.w {
		b.r = 0
		b.w = 0
	}
	if n < len(p) {
		err = io.EOF
	}
	return n, err
}

func (b *SerialBuffer) ReadByte() (byte, error) {
	if b.r == b.w {
		return 0, io.EOF
	}
	c := b.data[b.r]
	b.r++
	if b.r == b.w {
		b.r = 0
		b.w = 0
	}
	return c, nil
}

func (b *SerialBuffer) ReadFromOnce(r io.Reader) (int, error) {
	b.ensureFree(PartIncSize)
	if len(b.data)-b.w == 0 {
		return 0, io.ErrShortBuffer
	}
	n, err := ReadUntil(r, b.data[b.w:])
	b.w += n
	return n, err
}

func (b *SerialBuffer) WriteToOnce(w io.Writer) (int, error) {
	if b.r == b.w {
		return 0, io.EOF
	}
	n, err := WriteUntil(w, b.data[b.r:b.w])
	b.r += n
	if b.r == b.w {
		b.r = 0
		b.w = 0
	}
	return n, err
}

func (b *SerialBuffer) ReadFrom(r io.Reader) (n int64, err error) {
	for {
		b.ensureFree(PartIncSize)
		if len(b.data)-b.w == 0 {
			// Only reachable when a limit prevents further growth. Surface
			// the constraint regardless of whether bytes were already read
			// in this call — callers (e.g. ReadFull) need to distinguish
			// "dst is full" from "source ended early".
			err = io.ErrShortBuffer
			break
		}
		nn, readErr := ReadUntil(r, b.data[b.w:])
		b.w += nn
		n += int64(nn)
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			err = readErr
			break
		}
	}
	return n, err
}

func (b *SerialBuffer) WriteTo(w io.Writer) (n int64, err error) {
	if b.r == b.w {
		return 0, io.EOF
	}
	for b.r < b.w {
		nn, writeErr := WriteUntil(w, b.data[b.r:b.w])
		b.r += nn
		n += int64(nn)
		if writeErr != nil {
			err = writeErr
			break
		}
	}
	if b.r == b.w {
		b.r = 0
		b.w = 0
	}
	return n, err
}

func (b *SerialBuffer) Len() int {
	return b.w - b.r
}

// Cap returns the size of the backing storage. It is the largest payload the
// buffer can hold without growing (or, on a limited buffer, without writes
// starting to return io.ErrShortBuffer).
func (b *SerialBuffer) Cap() int {
	return len(b.data)
}

// Reset discards the unread bytes and rewinds the cursors. The backing
// storage is retained so subsequent writes do not need to allocate. Call
// FreeMe instead to also release the backing slice to the pool.
func (b *SerialBuffer) Reset() {
	b.r = 0
	b.w = 0
}

// Next returns at most n unread bytes as a slice into the backing storage,
// advancing the read cursor by len(returned). The slice is valid until the
// next operation that mutates the buffer (Write*, ReadFrom, FreeMe, Reset).
// Returns nil when the buffer is empty.
func (b *SerialBuffer) Next(n int) []byte {
	avail := b.w - b.r
	if n > avail {
		n = avail
	}
	if n <= 0 {
		return nil
	}
	out := b.data[b.r : b.r+n]
	b.r += n
	if b.r == b.w {
		b.r = 0
		b.w = 0
	}
	return out
}

func (b *SerialBuffer) FreeMe() {
	if cap(b.data) > 0 {
		clear(b.data)
		putSerialBuffer(b.data)
	}
	b.data = nil
	b.r = 0
	b.w = 0
	b.limit = 0
}

// ensureFree makes a best effort to expose at least `need` bytes at b.data[b.w:].
// It first compacts the unread region to the start, then grows the backing
// buffer. When a limit is set, growth is capped at the limit and the caller is
// responsible for reading off any io.ErrShortBuffer signaled by the actual
// write.
func (b *SerialBuffer) ensureFree(need int) {
	if need <= 0 {
		return
	}
	if len(b.data)-b.w >= need {
		return
	}
	if b.r > 0 {
		n := copy(b.data, b.data[b.r:b.w])
		b.w = n
		b.r = 0
		if len(b.data)-b.w >= need {
			return
		}
	}
	newCap := b.nextCap(b.w + need)
	if b.limit > 0 && newCap > b.limit {
		newCap = b.limit
	}
	if newCap <= len(b.data) {
		return
	}
	newData := getSerialBuffer(newCap)
	if len(newData) > newCap {
		newData = newData[:newCap]
	}
	if b.w > 0 {
		copy(newData, b.data[:b.w])
	}
	if cap(b.data) > 0 {
		putSerialBuffer(b.data)
		b.data = nil
	}
	b.data = newData
}

// nextCap picks the next backing-buffer length that fits `need` bytes total.
// Within the pool range it returns the next power of two >= PartMinimalSize so
// internal.Get can hand back a slice with len == cap. Past the pool ceiling it
// grows geometrically (cur + cur>>serialGrowShift: 1.5x default, 1.25x low-mem)
// to keep amortized growth O(1) while bounding overshoot.
func (b *SerialBuffer) nextCap(need int) int {
	if need < PartMinimalSize {
		return PartMinimalSize
	}

	if need <= internal.MaxAllocatableSize {
		c := PartMinimalSize
		for c < need {
			c <<= 1
		}
		return c
	}

	next := len(b.data) + len(b.data)>>serialGrowShift
	if next < need {
		return need
	}
	return next
}

func getSerialBuffer(size int) []byte {
	if size <= 0 {
		return nil
	}
	if size > internal.MaxAllocatableSize {
		return make([]byte, size)
	}
	b := internal.Get(size)
	return b[:cap(b)]
}

func putSerialBuffer(b []byte) {
	if cap(b) > 0 && cap(b) <= internal.MaxAllocatableSize {
		internal.Put(b)
	}
}
