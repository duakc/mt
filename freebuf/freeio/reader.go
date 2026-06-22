package freeio

import (
	"bytes"
	"errors"
	"io"
	"unsafe"

	"github.com/duakc/mt/freebuf"
	"github.com/duakc/mt/freebuf/internal"
)

const (
	// defaultBufSize is NewReader/NewWriter's buffer size. It tracks freebuf's
	// PartMinimalSize, so the low-memory build (-tags freebuf_low_mem) shrinks it
	// from 4096 to 1024.
	defaultBufSize = freebuf.PartMinimalSize
	// minBufSize floors NewReaderSize/NewWriterSize so Peek/ReadSlice have room.
	minBufSize = 16
)

var (
	// ErrBufferFull is returned by Peek/ReadSlice when the delimiter or the
	// requested count does not fit in the buffer.
	ErrBufferFull = errors.New("freeio: buffer full")
	// ErrNegativeCount is returned by Peek/Discard for a negative argument.
	ErrNegativeCount = errors.New("freeio: negative count")
	// ErrInvalidUnreadByte is returned by UnreadByte when the previous operation
	// was not a successful single-byte read.
	ErrInvalidUnreadByte = errors.New("freeio: invalid use of UnreadByte")
)

// poolGet returns a pooled backing array of exactly size bytes. Its cap is the
// pool bucket (a power of two ≤ 64KB) that poolPut needs to recycle it; the
// length is the requested size, so Size() is predictable. A size above the pool
// ceiling falls back to a plain make.
func poolGet(size int) []byte {
	return internal.Get(size)
}

// poolPut returns a poolGet array to the pool; a non-bucket slice (the >64KB
// make) is dropped by internal.Put.
func poolPut(b []byte) {
	if cap(b) > 0 {
		internal.Put(b[:cap(b)])
	}
}

// Reader is a buffered reader — the freebuf-backed analog of bufio.Reader. It
// amortizes the wrapped reader's Read calls through a pooled, fixed-size byte
// buffer. The buffer is contiguous, so Peek and ReadSlice return slices that
// point straight into it, with no copy.
//
// Call Free when done to return the pooled buffer. A Reader is not safe for
// concurrent use.
type Reader struct {
	buf  []byte
	src  io.Reader
	r, w int // buf read and write positions: unread bytes are buf[r:w]
	// err is the wrapped reader's last error, held until the buffer drains and
	// then surfaced once (like bufio): the buffered bytes are delivered first.
	err      error
	lastByte int // last byte read, for UnreadByte; -1 when invalid
}

func NewReader(rd io.Reader) *Reader {
	return NewReaderSize(rd, defaultBufSize)
}

func NewReaderSize(rd io.Reader, size int) *Reader {
	if size < minBufSize {
		size = minBufSize
	}
	return &Reader{buf: poolGet(size), src: rd, lastByte: -1}
}

func (b *Reader) Reset(r io.Reader) {
	b.src = r
	b.r, b.w = 0, 0
	b.err = nil
	b.lastByte = -1
}

func (b *Reader) Free() {
	poolPut(b.buf)
	b.buf = nil
}

func (b *Reader) Size() int { return len(b.buf) }

func (b *Reader) Buffered() int { return b.w - b.r }

func (b *Reader) readErr() error {
	err := b.err
	b.err = nil
	return err
}

// fill slides the unread bytes to the front and issues one Read into the free
// space. Callers ensure the buffer is not full and no error is pending.
func (b *Reader) fill() {
	if b.r > 0 {
		b.w = copy(b.buf, b.buf[b.r:b.w])
		b.r = 0
	}
	n, err := freebuf.ReadUntil(b.src, b.buf[b.w:])
	b.w += n
	if err != nil {
		b.err = err
	}
}

func (b *Reader) Read(p []byte) (int, error) {
	// Fast path: serve from the buffer. Also covers len(p)==0 with a non-empty
	// buffer, which returns (0, nil).
	if b.r < b.w {
		n := copy(p, b.buf[b.r:b.w])
		b.r += n
		if n > 0 {
			b.lastByte = int(b.buf[b.r-1])
		}
		return n, nil
	}
	return b.readEmpty(p)
}

// readEmpty handles Read when the buffer is drained: report a pending error,
// bypass the buffer for a large read, or refill once and serve.
func (b *Reader) readEmpty(p []byte) (n int, err error) {
	if b.err != nil {
		return 0, b.readErr()
	}
	if len(p) == 0 {
		return 0, nil
	}
	// large read into an empty buffer: read straight into p, skipping the
	// buffer (and the copy) entirely.
	if len(p) >= len(b.buf) {
		n, b.err = b.src.Read(p)
		if n > 0 {
			b.lastByte = int(p[n-1])
		}
		return n, b.readErr()
	}
	b.fill()
	if b.r == b.w {
		return 0, b.readErr()
	}
	n = copy(p, b.buf[b.r:b.w])
	b.r += n
	b.lastByte = int(b.buf[b.r-1])
	return n, nil
}

func (b *Reader) ReadByte() (byte, error) {
	// Fast path: a byte is already buffered — a direct load, no call.
	if b.r < b.w {
		c := b.buf[b.r]
		b.r++
		b.lastByte = int(c)
		return c, nil
	}
	return b.readByteFill()
}

// readByteFill is ReadByte's slow path: refill until a byte is available or the
// source errors.
func (b *Reader) readByteFill() (byte, error) {
	for b.r == b.w {
		if b.err != nil {
			return 0, b.readErr()
		}
		b.fill()
	}
	c := b.buf[b.r]
	b.r++
	b.lastByte = int(c)
	return c, nil
}

// UnreadByte unreads the last byte returned by the most recent read operation.
// Only the most recent read can be undone; any other intervening operation
// makes it invalid (ErrInvalidUnreadByte).
func (b *Reader) UnreadByte() error {
	if b.lastByte < 0 || b.r == 0 && b.w > 0 {
		return ErrInvalidUnreadByte
	}
	if b.r > 0 {
		b.r--
	} else {
		// b.r == 0 && b.w == 0
		b.w = 1
	}
	b.buf[b.r] = byte(b.lastByte)
	b.lastByte = -1
	return nil
}

// Peek returns the next n bytes without advancing the reader. The bytes stop
// being valid at the next read. If fewer than n bytes are available it returns
// what it has with ErrBufferFull (n exceeds the buffer) or the read error.
func (b *Reader) Peek(n int) ([]byte, error) {
	if n < 0 {
		return nil, ErrNegativeCount
	}
	b.lastByte = -1
	for b.w-b.r < n && b.w-b.r < len(b.buf) && b.err == nil {
		b.fill()
	}
	// A request larger than the buffer can never be satisfied, EOF or not.
	if n > len(b.buf) {
		return b.buf[b.r:b.w], ErrBufferFull
	}
	var err error
	if avail := b.w - b.r; avail < n {
		n = avail
		// short of n: a pending read error (e.g. io.EOF) explains why.
		if err = b.readErr(); err == nil {
			err = ErrBufferFull
		}
	}
	return b.buf[b.r : b.r+n], err
}

// Discard skips the next n bytes, returning the number skipped. If it skips
// fewer than n it also returns the error that cut it short.
func (b *Reader) Discard(n int) (discarded int, err error) {
	if n < 0 {
		return 0, ErrNegativeCount
	}
	if n == 0 {
		return 0, nil
	}
	b.lastByte = -1
	remain := n
	for {
		skip := b.w - b.r
		if skip == 0 {
			if b.err != nil {
				return n - remain, b.readErr()
			}
			b.fill()
			skip = b.w - b.r
		}
		if skip > remain {
			skip = remain
		}
		b.r += skip
		remain -= skip
		if remain == 0 {
			return n, nil
		}
	}
}

// ReadSlice reads until the first occurrence of delim, returning a slice that
// points into the buffer (valid only until the next read). If delim is not
// found before the buffer fills it returns the buffered bytes and
// ErrBufferFull; on EOF it returns the trailing bytes and io.EOF. Use ReadBytes
// when the result must outlive the next read or may exceed the buffer.
func (b *Reader) ReadSlice(delim byte) (line []byte, err error) {
	// offset into the unread region already scanned for delim
	s := 0
	for {
		if i := bytes.IndexByte(b.buf[b.r+s:b.w], delim); i >= 0 {
			i += s
			line = b.buf[b.r : b.r+i+1]
			b.r += i + 1
			break
		}
		if b.err != nil {
			line = b.buf[b.r:b.w]
			b.r = b.w
			err = b.readErr()
			break
		}
		if b.w-b.r >= len(b.buf) {
			// buffer full, delim not found
			b.r = b.w
			line = b.buf
			err = ErrBufferFull
			break
		}
		// don't rescan what we already searched
		s = b.w - b.r
		b.fill()
	}
	if i := len(line) - 1; i >= 0 {
		b.lastByte = int(line[i])
	}
	return
}

// ReadBytes reads until the first occurrence of delim, returning a freshly
// allocated slice of the data up to and including it. Unlike ReadSlice the
// result is owned by the caller and may span multiple buffer fills. A delim not
// found before EOF yields the data read so far and io.EOF.
func (b *Reader) ReadBytes(delim byte) ([]byte, error) {
	frag, err := b.ReadSlice(delim)
	if err == nil {
		return append([]byte(nil), frag...), nil
	}
	// delim spans more than one buffer: stitch the fragments together.
	// always a copy here
	out := append([]byte(nil), frag...)
	for err == ErrBufferFull {
		frag, err = b.ReadSlice(delim)
		out = append(out, frag...)
	}
	return out, err
}

func (b *Reader) ReadString(delim byte) (string, error) {
	bs, err := b.ReadBytes(delim)

	// use unsafe at here is safety.
	// Reader.ReadBytes already copy the buffered data
	return unsafe.String(unsafe.SliceData(bs), len(bs)), err
}

// WriteTo writes all buffered and remaining source bytes to w, implementing
// io.WriterTo. After flushing the buffer it hands off to Copy, so a file or
// socket source still reaches the kernel zero-copy paths.
func (b *Reader) WriteTo(w io.Writer) (n int64, err error) {
	b.lastByte = -1

	// before we copy , flush all the buffered data to writer.
	if b.r < b.w {
		m, e := w.Write(b.buf[b.r:b.w])
		b.r += m
		n += int64(m)
		if e != nil {
			return n, e
		}
	}
	if b.err != nil {
		return n, b.readErr()
	}

	// use Copy , so the system zero-copy will applied.
	m, e := Copy(w, b.src)
	return n + m, e
}
