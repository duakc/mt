package freebuf

import (
	"io"
)

type FreeBuf struct {
	part *bytePart
}

func New(size int) *FreeBuf {
	p := alloc(size)
	p.limit(size)

	return &FreeBuf{part: p}
}

func (b *FreeBuf) Read(p []byte) (n int, err error) {
	return b.part.read(p)
}

func (b *FreeBuf) Write(p []byte) (n int, err error) {
	return b.part.write(p)
}

func (b *FreeBuf) WriteString(s string) (n int, err error) {
	return b.part.writeString(s)
}

func (b *FreeBuf) WriteByte(c byte) error {
	return b.part.writeByte(c)
}

func (b *FreeBuf) ReadByte() (byte, error) {
	return b.part.readByte()
}

func (b *FreeBuf) ReadFrom(r io.Reader) (n int64, err error) {
	if b.part.freeSpace() == 0 {
		return 0, io.ErrShortBuffer
	}

	for b.part.freeSpace() > 0 {
		var nn int
		nn, err = b.part.readFromOnce(r)
		n += int64(nn)
		if err != nil {
			break
		}
	}
	if err == errBytePartReadFromOnceFull || err == io.EOF {
		err = nil
	}
	return n, err
}

func (b *FreeBuf) WriteTo(w io.Writer) (n int64, err error) {
	if b.part.len() == 0 {
		return 0, io.EOF
	}
	for b.part.len() > 0 {
		var nn int
		nn, err = b.part.writeToOnce(w)
		n += int64(nn)
		if err != nil {
			break
		}
	}
	if err == errBytePartWriteToOnceEmpty {
		err = nil
	}
	return n, err
}

func (b *FreeBuf) Len() int {
	return b.part.len()
}

func (b *FreeBuf) FreeMe() {
	free(b.part)
	b.part = nil
}
