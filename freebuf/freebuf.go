package freebuf

import "io"

type Buffer interface {
	io.ReadWriter
	io.StringWriter
	io.ByteWriter
	io.ByteReader
	io.ReaderFrom
	io.WriterTo

	Len() int
	FreeMe()
	Reset()

	// Copy returns a deep copy of the unread bytes in a fresh Buffer of the
	// same concrete type. Read/write cursors on the copy start at the
	// beginning; the original is untouched. The returned Buffer must be
	// released with FreeMe by the caller.
	Copy() Buffer

	// Grow makes a best effort to ensure at least n bytes of free space are
	// available for the next write. On a limited buffer the growth is capped
	// at the configured limit; subsequent writes that exceed it still return
	// io.ErrShortBuffer for the overflow.
	Grow(n int)

	// ReadFromOnce performs a single Read call against r into the buffer's free
	// space, growing by one chunk if necessary. Returns io.ErrShortBuffer if the
	// buffer is limited and already full. Useful for nonblocking or bounded-step
	// I/O loops that need to interleave reads with other work.
	ReadFromOnce(r io.Reader) (int, error)

	// WriteToOnce performs a single Write call against w consuming as many unread
	// bytes as the writer accepts. Returns io.EOF if the buffer is empty. Partial
	// writes surface as io.ErrShortWrite; the bytes that were accepted remain
	// consumed.
	WriteToOnce(w io.Writer) (int, error)
}

func ReadUntil(r io.Reader, buf []byte) (n int, err error) {
	if len(buf) == 0 {
		return 0, nil
	}

	const maxRetry = 16

	for retry := 0; retry < maxRetry; retry++ {
		n, err = r.Read(buf)
		if n != 0 || err != nil {
			return n, err
		}
	}
	return 0, io.ErrNoProgress
}

func WriteUntil(w io.Writer, buf []byte) (n int, err error) {
	if len(buf) == 0 {
		return 0, nil
	}
	n, err = w.Write(buf)
	if n < len(buf) && err == nil {
		err = io.ErrShortWrite
	}
	return n, err
}

// Deprecated: use WriteUntil
func WriteFull(w io.Writer, buf []byte) (n int, err error) {
	if len(buf) == 0 {
		return 0, nil
	}

	const maxRetry = 16
	nn := len(buf)
	for writeN, retry := 0, 0; writeN < nn; {
		written, err := w.Write(buf[writeN:nn])
		writeN += written
		if err != nil {
			return writeN, err
		}
		if written == 0 {
			retry++
		}
		if retry >= maxRetry {
			return nn, io.ErrNoProgress
		}
		retry = 0
	}
	return nn, nil
}
