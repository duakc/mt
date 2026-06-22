package freeio

import (
	"io"

	"github.com/duakc/mt/freebuf"
)

// Writer is a buffered writer — the freebuf-backed analog of bufio.Writer. It
// accumulates writes in a pooled, fixed-size byte buffer and flushes to the
// wrapped writer when the buffer fills or Flush is called.
//
// Once the wrapped writer returns an error it is recorded and returned by every
// subsequent operation until Reset. Call Flush before discarding the Writer or
// buffered bytes are lost; Free flushes then returns the pooled buffer. A
// Writer is not safe for concurrent use.
type Writer struct {
	buf []byte
	dst io.Writer
	n   int // bytes buffered: buf[:n]
	err error
}

func NewWriter(w io.Writer) *Writer {
	return NewWriterSize(w, defaultBufSize)
}

func NewWriterSize(w io.Writer, size int) *Writer {
	if size < minBufSize {
		size = minBufSize
	}
	return &Writer{buf: poolGet(size), dst: w}
}

func (b *Writer) Reset(w io.Writer) {
	b.dst = w
	b.n = 0
	b.err = nil
}

func (b *Writer) Free() {
	_ = b.Flush()
	poolPut(b.buf)
	b.buf = nil
}

func (b *Writer) Size() int { return len(b.buf) }

func (b *Writer) Buffered() int { return b.n }

func (b *Writer) Available() int { return len(b.buf) - b.n }

func (b *Writer) Flush() error {
	if b.err != nil {
		return b.err
	}
	if b.n == 0 {
		return nil
	}
	n, err := b.dst.Write(b.buf[:b.n])
	if n < b.n && err == nil {
		err = io.ErrShortWrite
	}
	if err != nil {
		if n > 0 && n < b.n {
			copy(b.buf, b.buf[n:b.n]) // keep the bytes that weren't written
		}
		b.n -= n
		b.err = err
		return err
	}
	b.n = 0
	return nil
}

func (b *Writer) Write(p []byte) (int, error) {
	// Fast path: it all fits, just copy into the buffer.
	if b.err == nil && len(p) <= b.Available() {
		n := copy(b.buf[b.n:], p)
		b.n += n
		return n, nil
	}
	return b.writeMulti(p)
}

// writeMulti handles writes that span buffer flushes (or hit a sticky error):
// flush-and-refill, or send a large write straight to the destination.
func (b *Writer) writeMulti(p []byte) (nn int, err error) {
	for len(p) > b.Available() {
		var n int
		if b.n == 0 {
			// large write into an empty buffer: go straight to the destination,
			// skipping the buffer copy.
			n, b.err = b.dst.Write(p)
		} else {
			n = copy(b.buf[b.n:], p)
			b.n += n
			b.Flush()
		}
		nn += n
		p = p[n:]
		if b.err != nil {
			return nn, b.err
		}
	}
	n := copy(b.buf[b.n:], p)
	b.n += n
	return nn + n, nil
}

func (b *Writer) WriteByte(c byte) error {
	// Fast path: room in the buffer — a direct store, no call (inlines like
	// bufio). Flushing happens once per bufferful, never per byte.
	if b.err == nil && b.n < len(b.buf) {
		b.buf[b.n] = c
		b.n++
		return nil
	}
	return b.writeByteFull(c)
}

// writeByteFull is WriteByte's slow path: surface a sticky error, or flush the
// full buffer before storing the byte (space is then guaranteed).
func (b *Writer) writeByteFull(c byte) error {
	if b.err != nil {
		return b.err
	}
	if err := b.Flush(); err != nil {
		return err
	}
	b.buf[b.n] = c
	b.n++
	return nil
}

func (b *Writer) WriteString(s string) (int, error) {
	if b.err == nil && len(s) <= b.Available() {
		n := copy(b.buf[b.n:], s)
		b.n += n
		return n, nil
	}
	return b.writeStringMulti(s)
}

func (b *Writer) writeStringMulti(s string) (nn int, err error) {
	for len(s) > b.Available() {
		var n int
		if b.n == 0 {
			n, b.err = io.WriteString(b.dst, s)
		} else {
			n = copy(b.buf[b.n:], s)
			b.n += n
			b.Flush()
		}
		nn += n
		s = s[n:]
		if b.err != nil {
			return nn, b.err
		}
	}
	n := copy(b.buf[b.n:], s)
	b.n += n
	return nn + n, nil
}

// ReadFrom drains r into the writer, implementing io.ReaderFrom. As soon as the
// buffer is empty and the destination has its own ReaderFrom it hands r off
// directly — that reaches the os.File / net.Conn kernel zero-copy paths — after
// first flushing anything already buffered. Without a ReaderFrom destination it
// fills and flushes in a loop; bytes read after the last full buffer stay
// buffered, so Flush to commit them.
func (b *Writer) ReadFrom(r io.Reader) (n int64, err error) {
	if b.err != nil {
		return 0, b.err
	}
	rf, rfOK := b.dst.(io.ReaderFrom)
	for {
		if b.Available() == 0 {
			if err = b.Flush(); err != nil {
				return n, err
			}
		}
		if rfOK && b.n == 0 {
			m, e := rf.ReadFrom(r)
			n += m
			b.err = e
			return n, e
		}
		m, e := freebuf.ReadUntil(r, b.buf[b.n:])
		b.n += m
		n += int64(m)
		if e == io.EOF {
			return n, nil
		}
		if e != nil {
			return n, e
		}
	}
}

// ReadWriter pairs a Reader and a Writer, the freebuf-backed analog of
// bufio.ReadWriter.
type ReadWriter struct {
	*Reader
	*Writer
}

func NewReadWriter(r *Reader, w *Writer) *ReadWriter {
	return &ReadWriter{r, w}
}
