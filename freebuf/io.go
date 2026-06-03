package freebuf

import "io"

// ReadAll reads from r until EOF and returns all bytes in a fresh Buffer.
// The returned Buffer is the caller's to manage — call FreeMe when done.
//
// Analogous to io.ReadAll. The returned implementation is MultiPartBuffer
// since the source size is unknown: chunked storage grows by appending pooled
// parts without memcpy, so the cost stays bounded for arbitrary payload sizes.
// A successful call returns nil error, not io.EOF.
func ReadAll(r io.Reader) (Buffer, error) {
	var buf Buffer
	if hasLen, isLen := r.(lenReader); isLen {
		buf = NewExcept(hasLen.Len())
	}
	if buf == nil {
		buf = NewMultiPart()
	}
	_, err := buf.ReadFrom(r)
	return buf, err
}

// ReadN reads exactly n bytes from r into a fresh Buffer. The implementation
// is chosen by NewExcept(n). If r EOFs after some but fewer than n bytes,
// ReadN returns the partial buffer with io.ErrUnexpectedEOF; with zero bytes
// read, io.EOF. Non-positive n returns an empty buffer.
//
// Convenience wrapper for NewExcept(n) + ReadFull. Useful for length-prefixed
// protocols where the frame size is known up front.
func ReadN(r io.Reader, n int) (Buffer, error) {
	buf := NewExcept(n)
	if n <= 0 {
		return buf, nil
	}
	_, err := ReadFull(r, buf, n)
	return buf, err
}

// ReadFull reads exactly n bytes from r and appends them to dst. Returns the
// number of bytes appended. If r EOFs after some but fewer than n bytes,
// ReadFull returns io.ErrUnexpectedEOF; with zero bytes read, io.EOF. A
// limited dst that fills up surfaces as io.ErrShortBuffer.
//
// Analogous to io.ReadFull, adapted to growable Buffer destinations: there is
// no "buffer too small" pre-check because dst grows on demand (unless its
// configured limit prevents it).
func ReadFull(r io.Reader, dst Buffer, n int) (read int64, err error) {
	if n <= 0 {
		return 0, nil
	}
	read, err = dst.ReadFrom(io.LimitReader(r, int64(n)))
	if err != nil {
		return read, err
	}
	if read < int64(n) {
		err = io.EOF
		if read != 0 {
			err = io.ErrUnexpectedEOF
		}
		return read, err
	}
	return read, nil
}

type lenReader interface {
	io.Reader

	Len() int
}
