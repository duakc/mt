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
