package freebuf

import (
	"io"
	"iter"
)

var _ Buffer = (*MultiPartBuffer)(nil)

// MultiPartBuffer is a chunked, growable Buffer. Bytes live in a chain of
// small pool-backed parts; chunks are released back to the pool as soon as
// they are fully consumed. Use it when data lifetime is fragmented and
// contiguity does not matter. For cache-friendly access use SerialBuffer.
//
// New chunks for Write/WriteByte/WriteString are sized to PartMinimalSize.
// ReadFrom uses PartIncSize so each underlying Read call can drain a
// larger window from the source.
type MultiPartBuffer struct {
	// parts[head:] is the active region. We never re-slice the front to keep
	// the underlying array from leaking on long-running buffers; instead we
	// compact when head occupies enough of the slice.
	parts []*bytePart
	head  int
	total int
}

func NewMultiPart() *MultiPartBuffer {
	return &MultiPartBuffer{}
}

// tail returns the trailing part with free space, allocating a new chunk of
// the requested size when the current tail is full or no parts exist.
func (c *MultiPartBuffer) tail(chunkSize int) *bytePart {
	if n := len(c.parts); n > 0 && c.parts[n-1].freeSpace() > 0 {
		return c.parts[n-1]
	}
	bp := alloc(chunkSize)
	c.parts = append(c.parts, bp)
	return bp
}

func (c *MultiPartBuffer) dropHead() {
	bp := c.parts[c.head]
	c.parts[c.head] = nil
	free(bp)
	c.head++
	if c.head == len(c.parts) {
		// rotate
		c.parts = c.parts[:0]
		c.head = 0
		return
	}
	if c.head >= 8 && c.head*2 >= len(c.parts) {
		n := copy(c.parts, c.parts[c.head:])
		for i := n; i < len(c.parts); i++ {
			c.parts[i] = nil
		}
		c.parts = c.parts[:n]
		c.head = 0
	}
}

func (c *MultiPartBuffer) Write(p []byte) (n int, err error) {
	for n < len(p) {
		bp := c.tail(PartMinimalSize)
		nn, _ := bp.write(p[n:])
		n += nn
	}
	c.total += n
	return n, nil
}

func (c *MultiPartBuffer) WriteString(s string) (n int, err error) {
	for n < len(s) {
		bp := c.tail(PartMinimalSize)
		nn, _ := bp.writeString(s[n:])
		n += nn
	}
	c.total += n
	return n, nil
}

func (c *MultiPartBuffer) WriteByte(b byte) error {
	bp := c.tail(PartMinimalSize)
	if err := bp.writeByte(b); err != nil {
		return err
	}
	c.total++
	return nil
}

func (c *MultiPartBuffer) Read(p []byte) (n int, err error) {
	for n < len(p) && c.head < len(c.parts) {
		bp := c.parts[c.head]
		if bp.len() == 0 {
			c.dropHead()
			continue
		}
		nn, _ := bp.read(p[n:])
		n += nn
		if bp.len() == 0 {
			c.dropHead()
		}
	}
	c.total -= n
	if n < len(p) {
		err = io.EOF
	}
	return n, err
}

func (c *MultiPartBuffer) ReadByte() (byte, error) {
	for c.head < len(c.parts) {
		bp := c.parts[c.head]
		if bp.len() == 0 {
			c.dropHead()
			continue
		}
		b, _ := bp.readByte()
		c.total--
		if bp.len() == 0 {
			c.dropHead()
		}
		return b, nil
	}
	return 0, io.EOF
}

func (c *MultiPartBuffer) ReadFrom(r io.Reader) (n int64, err error) {
	for {
		bp := c.tail(PartIncSize)
		nn, readErr := bp.readFromOnce(r)
		n += int64(nn)
		c.total += nn
		if readErr == errBytePartReadFromOnceFull {
			continue
		}
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

func (c *MultiPartBuffer) WriteTo(w io.Writer) (n int64, err error) {
	if c.total == 0 {
		return 0, io.EOF
	}
	for c.head < len(c.parts) {
		bp := c.parts[c.head]
		if bp.len() == 0 {
			c.dropHead()
			continue
		}
		nn, writeErr := bp.writeToOnce(w)
		n += int64(nn)
		c.total -= nn
		if bp.len() == 0 {
			c.dropHead()
		}
		if writeErr != nil {
			err = writeErr
			break
		}
	}
	return n, err
}

func (c *MultiPartBuffer) Len() int {
	return c.total
}

// PartCount returns the number of active chunks currently holding unread
// bytes. Mostly useful for monitoring fragmentation in long-running buffers.
func (c *MultiPartBuffer) PartCount() int {
	return len(c.parts) - c.head
}

// Reset discards all unread bytes, returning their backing parts to the pool.
// The parts slice header is retained so subsequent writes can reuse it without
// allocating a new slice. Call FreeMe to also release the slice header.
func (c *MultiPartBuffer) Reset() {
	for i := c.head; i < len(c.parts); i++ {
		free(c.parts[i])
		c.parts[i] = nil
	}
	c.parts = c.parts[:0]
	c.head = 0
	c.total = 0
}

// Chunks yields the active chunks of unread bytes as slices into the backing
// parts, in order. The slices are valid until the next operation that mutates
// the buffer; do not mutate the buffer during iteration. Useful for
// zero-copy traversal — e.g. feeding a hash without copying through Read.
//
// Iteration does not consume bytes; pair with Read or a per-chunk advance to
// drain the buffer.
func (c *MultiPartBuffer) Chunks() iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		for i := c.head; i < len(c.parts); i++ {
			bp := c.parts[i]
			if bp.len() == 0 {
				continue
			}
			if !yield(bp.data[bp.r:bp.w]) {
				return
			}
		}
	}
}

func (c *MultiPartBuffer) FreeMe() {
	for i := c.head; i < len(c.parts); i++ {
		free(c.parts[i])
		c.parts[i] = nil
	}
	c.parts = nil
	c.head = 0
	c.total = 0
}
