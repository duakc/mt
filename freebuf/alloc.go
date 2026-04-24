package freebuf

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/duakc/mt/freebuf/internal"
)

var bytePartPool = sync.Pool{
	New: func() any {
		return new(bytePart)
	},
}

const (
	maxSize = internal.MaxAllocatableSize
)

func alloc(size int) *bytePart {
	bp := bytePartPool.Get().(*bytePart)
	size = max(PartMinimalSize, size)
	bp.data = internal.Get(size)
	bp.w = 0
	bp.r = 0
	return bp
}

func free(b *bytePart) {
	if b == nil || b.data == nil || len(b.data) > maxSize {
		return
	}

	b.reset()
	internal.Put(b.data)
	b.data = nil
	bytePartPool.Put(b)
}

func freeMulti(bs []*bytePart) {
	for i := 0; i < len(bs); i++ {
		free(bs[i])
	}
}

type bytePart struct {
	data []byte

	r, w int
}

func (h *bytePart) reset() {
	clear(h.data)
	h.r = 0
	h.w = 0
}

func (h *bytePart) write(b []byte) (n int, err error) {
	if h.freeSpace() == 0 {
		return 0, io.ErrShortBuffer
	}
	if h.len() < len(b) {
		err = io.ErrShortBuffer
	}

	to := min(h.w+len(b), len(h.data))
	n = copy(h.data[h.w:to], b)
	h.w += n

	return n, err
}

func (h *bytePart) writeByte(b byte) error {
	if h.freeSpace() == 0 {
		return io.ErrShortBuffer
	}
	h.w++
	h.data[h.w] = b
	return nil
}

func (h *bytePart) readByte() (byte, error) {
	if h.len() == 0 {
		return 0, io.EOF
	}
	b := h.data[h.r]
	h.r++
	return b, nil
}

func (h *bytePart) writeString(s string) (n int, err error) {
	if h.freeSpace() == 0 {
		return 0, io.ErrShortBuffer
	}
	if h.len() < len(s) {
		err = io.ErrShortBuffer
	}

	to := min(h.w+len(s), len(h.data))
	n = copy(h.data[h.w:to], s)
	h.w += n

	return n, err
}

func (h *bytePart) read(b []byte) (int, error) {
	if h.len() == 0 {
		return 0, io.EOF
	}
	to := min(h.r+len(b), h.w)
	nn := copy(b[:h.w-h.r], h.data[h.r:to])
	h.r += nn
	var err error
	if nn != len(b) {
		err = io.EOF
	}
	return nn, err
}

func (h *bytePart) len() int {
	return h.w - h.r
}

func (h *bytePart) freeSpace() int {
	return len(h.data) - h.w
}

func (h *bytePart) limit(n int) {
	if n < 0 {
		panic("negative limit")
	}
	if n < h.w || n > len(h.data) {
		panic("limit out of range")
	}
	h.data = h.data[:n]
}

// due the io.Reader Read() method may return another io.ErrShortBuffer
// we need a special value to specifics the buffer is full
var errBytePartReadFromOnceFull = errors.New("buffer full")

func (h *bytePart) readFromOnce(r io.Reader) (n int, err error) {
	if h.freeSpace() == 0 {
		return 0, errBytePartReadFromOnceFull
	}
	n, err = ReadUntil(r, h.data[h.w:])
	h.w += n
	return n, err
}

var errBytePartWriteToOnceEmpty = errors.New("buffer empty")

func (h *bytePart) writeToOnce(w io.Writer) (n int, err error) {
	if h.len() == 0 {
		return 0, errBytePartWriteToOnceEmpty
	}
	n, err = WriteUntil(w, h.data[h.r:h.w])
	if n < h.len() && err == nil {
		err = io.ErrShortWrite
	}
	return n, err
}

func mustWrite(what string, n int, err error) int {
	if err != nil && err != io.ErrShortBuffer {
		panic(fmt.Sprintf("%s: written=%d: %s", what, n, err.Error()))
	}
	return n
}

func mustRead(what string, n int, err error) int {
	if err != nil && err != io.EOF {
		panic(fmt.Sprintf("%s: read=%d: %s", what, n, err.Error()))
	}
	return n
}
